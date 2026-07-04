// memdb.go — un DRIVER SQL FACTICE, en mémoire, écrit UNIQUEMENT avec la
// bibliothèque standard (database/sql/driver). Il n'existe que pour rendre les
// exemples de ce chapitre EXÉCUTABLES hors ligne, sans dépendance externe.
//
// Il illustre surtout le point central du chapitre : database/sql est une API
// ABSTRAITE. Le code utilisateur (sql.Open, QueryRowContext, Scan, BeginTx…) ne
// connaît PAS ce driver ; il ne parle qu'aux types de database/sql. On pourrait
// remplacer "memdb" par "sqlite" ou "pgx" sans changer une ligne côté appelant.
//
// Un VRAI driver parse le SQL, dialogue avec un moteur, gère les types et la
// pagination réseau. Ici on reconnaît un jeu FIXE de requêtes sur une unique
// table users(id, name, email). Ne copiez pas ce code en production : utilisez
// modernc.org/sqlite (pur Go, sans cgo) ou github.com/jackc/pgx (PostgreSQL).
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

func init() {
	// C'est EXACTEMENT ce que fait l'import blanc d'un vrai driver
	// (ex. `import _ "modernc.org/sqlite"`) : son init() appelle sql.Register.
	sql.Register("memdb", &memDriver{stores: map[string]*store{}})
}

// userRow est une ligne de la table users.
type userRow struct {
	id    int64
	name  string
	email string
}

// store contient les données d'UNE base, identifiée par son DSN. Le mutex le
// protège car database/sql ouvre plusieurs connexions concurrentes vers le
// même store (le pool).
type store struct {
	mu     sync.Mutex
	users  map[int64]userRow
	nextID int64
}

func newStore() *store { return &store{users: map[int64]userRow{}, nextID: 1} }

// snapshot renvoie une copie profonde des données, utilisée au début d'une
// transaction comme copie de travail.
func (s *store) snapshot() (map[int64]userRow, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make(map[int64]userRow, len(s.users))
	for k, v := range s.users {
		cp[k] = v
	}
	return cp, s.nextID
}

// memDriver implémente driver.Driver. Il conserve un store par DSN : deux
// sql.Open avec le même DSN partagent donc les mêmes données.
type memDriver struct {
	mu     sync.Mutex
	stores map[string]*store
}

// Open est appelée par database/sql pour créer une nouvelle connexion.
func (d *memDriver) Open(dsn string) (driver.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	st := d.stores[dsn]
	if st == nil {
		st = newStore()
		d.stores[dsn] = st
	}
	return &conn{st: st}, nil
}

// conn implémente driver.Conn (+ PrepareContext et BeginTx). database/sql
// garantit qu'une connexion n'est jamais utilisée par deux goroutines à la
// fois : l'état ci-dessous n'a donc pas besoin de verrou.
type conn struct {
	st *store
	tx *txn // non nil pendant une transaction
}

// txn est la copie de travail d'une transaction : les écritures s'y appliquent
// jusqu'au Commit (qui la publie dans le store) ou au Rollback (qui la jette).
type txn struct {
	users  map[int64]userRow
	nextID int64
}

func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *conn) PrepareContext(_ context.Context, query string) (driver.Stmt, error) {
	kind, err := classify(query)
	if err != nil {
		return nil, err
	}
	return &stmt{c: c, kind: kind}, nil
}

func (c *conn) Close() error { return nil }

func (c *conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *conn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if c.tx != nil {
		return nil, fmt.Errorf("memdb : transaction déjà ouverte sur cette connexion")
	}
	users, nextID := c.st.snapshot()
	c.tx = &txn{users: users, nextID: nextID}
	return &memTx{c: c}, nil
}

// memTx implémente driver.Tx.
type memTx struct{ c *conn }

func (t *memTx) Commit() error {
	tx := t.c.tx
	if tx == nil {
		return fmt.Errorf("memdb : pas de transaction à valider")
	}
	// Publie la copie de travail dans le store partagé.
	t.c.st.mu.Lock()
	t.c.st.users = tx.users
	t.c.st.nextID = tx.nextID
	t.c.st.mu.Unlock()
	t.c.tx = nil
	return nil
}

func (t *memTx) Rollback() error {
	// On jette simplement la copie de travail ; le store partagé reste intact.
	t.c.tx = nil
	return nil
}

// queryKind identifie l'une des requêtes reconnues par ce driver factice.
type queryKind int

const (
	kindInsert queryKind = iota
	kindSelectByID
	kindSelectAll
	kindUpdateName
	kindDeleteByID
)

func (k queryKind) numInput() int {
	switch k {
	case kindInsert, kindUpdateName:
		return 2
	case kindSelectByID, kindDeleteByID:
		return 1
	default: // kindSelectAll
		return 0
	}
}

// classify reconnaît le SQL parmi un jeu fixe. Un vrai driver PARSE la requête ;
// ici on se contente d'une correspondance après normalisation des espaces.
func classify(query string) (queryKind, error) {
	switch normalize(query) {
	case "insert into users(name, email) values (?, ?)":
		return kindInsert, nil
	case "select id, name, email from users where id = ?":
		return kindSelectByID, nil
	case "select id, name, email from users":
		return kindSelectAll, nil
	case "update users set name = ? where id = ?":
		return kindUpdateName, nil
	case "delete from users where id = ?":
		return kindDeleteByID, nil
	default:
		return 0, fmt.Errorf("memdb : requête non reconnue par ce driver factice : %q", query)
	}
}

func normalize(q string) string {
	return strings.Join(strings.Fields(strings.ToLower(q)), " ")
}

// stmt implémente driver.Stmt (+ ExecContext et QueryContext).
type stmt struct {
	c    *conn
	kind queryKind
}

func (s *stmt) Close() error  { return nil }
func (s *stmt) NumInput() int { return s.kind.numInput() }

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), toNamed(args))
}

func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err // honore l'annulation / le timeout du contexte
	}
	c := s.c
	if c.tx != nil { // écrit dans la copie de transaction
		return execOn(s.kind, c.tx.users, &c.tx.nextID, args)
	}
	c.st.mu.Lock() // sinon dans le store partagé, sous verrou
	defer c.st.mu.Unlock()
	return execOn(s.kind, c.st.users, &c.st.nextID, args)
}

func execOn(kind queryKind, users map[int64]userRow, nextID *int64, args []driver.NamedValue) (driver.Result, error) {
	switch kind {
	case kindInsert:
		id := *nextID
		*nextID++
		users[id] = userRow{id: id, name: asString(args[0]), email: asString(args[1])}
		return result{lastID: id, rows: 1}, nil
	case kindUpdateName:
		id := asInt64(args[1])
		u, ok := users[id]
		if !ok {
			return result{}, nil // 0 ligne affectée
		}
		u.name = asString(args[0])
		users[id] = u
		return result{rows: 1}, nil
	case kindDeleteByID:
		id := asInt64(args[0])
		if _, ok := users[id]; !ok {
			return result{}, nil
		}
		delete(users, id)
		return result{rows: 1}, nil
	default:
		return nil, fmt.Errorf("memdb : Exec non supporté pour cette requête")
	}
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), toNamed(args))
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c := s.c

	// On lit sur une copie des lignes : le curseur (rows) doit rester valide
	// après avoir relâché le verrou du store.
	var users map[int64]userRow
	if c.tx != nil {
		users = c.tx.users
	} else {
		c.st.mu.Lock()
		users = make(map[int64]userRow, len(c.st.users))
		for k, v := range c.st.users {
			users[k] = v
		}
		c.st.mu.Unlock()
	}

	var out []userRow
	switch s.kind {
	case kindSelectByID:
		if u, ok := users[asInt64(args[0])]; ok {
			out = append(out, u)
		}
	case kindSelectAll:
		for _, u := range users {
			out = append(out, u)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	default:
		return nil, fmt.Errorf("memdb : Query non supporté pour cette requête")
	}
	return &rows{data: out}, nil
}

// rows implémente driver.Rows : un curseur qu'on avance ligne par ligne.
type rows struct {
	data []userRow
	pos  int
}

func (r *rows) Columns() []string { return []string{"id", "name", "email"} }
func (r *rows) Close() error      { return nil }

func (r *rows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF // signale la fin du curseur à database/sql
	}
	u := r.data[r.pos]
	r.pos++
	dest[0] = u.id
	dest[1] = u.name
	dest[2] = u.email
	return nil
}

// result implémente driver.Result.
type result struct {
	lastID int64
	rows   int64
}

func (r result) LastInsertId() (int64, error) { return r.lastID, nil }
func (r result) RowsAffected() (int64, error) { return r.rows, nil }

// toNamed convertit les arguments positionnels en arguments nommés (ordinaux),
// pour partager le même chemin de code que les versions *Context.
func toNamed(args []driver.Value) []driver.NamedValue {
	nv := make([]driver.NamedValue, len(args))
	for i, v := range args {
		nv[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return nv
}

func asString(v driver.NamedValue) string {
	s, _ := v.Value.(string)
	return s
}

func asInt64(v driver.NamedValue) int64 {
	switch n := v.Value.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}
