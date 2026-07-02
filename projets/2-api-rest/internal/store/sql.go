package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"
)

// migrationsFS embarque les scripts SQL dans le binaire (Ch. 12) : pas de
// fichier à déployer à côté de l'exécutable.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// SQLStore est un Store adossé à une base relationnelle via database/sql.
//
// Aucun driver n'est importé ici : database/sql est une API abstraite. L'appelant
// doit enregistrer le sien par effet de bord, par exemple :
//
//	import _ "modernc.org/sqlite" // driver SQLite pur Go (sans cgo)
//
// puis NewSQLStore(ctx, "sqlite", "tasks.db"). Voir le README, section base de
// données. Les requêtes utilisent le marqueur « ? » (SQLite/MySQL) ; pour
// PostgreSQL (lib/pq, pgx), remplacer par « $1, $2, … ».
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore ouvre la base, vérifie la connexion (avec le contexte fourni,
// donc soumise à son éventuel délai) et applique les migrations.
func NewSQLStore(ctx context.Context, driver, dsn string) (*SQLStore, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("ouverture base : %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connexion base : %w", err)
	}
	s := &SQLStore{db: db}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close ferme le pool de connexions sous-jacent.
func (s *SQLStore) Close() error { return s.db.Close() }

// migrate applique toutes les migrations embarquées, dans l'ordre lexicographique
// des noms de fichier (0001_, 0002_, …). Idempotent grâce aux « IF NOT EXISTS ».
func (s *SQLStore) migrate(ctx context.Context) error {
	files, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("liste des migrations : %w", err)
	}
	slices.Sort(files) // le préfixe numérique fixe l'ordre d'application
	for _, f := range files {
		data, err := migrationsFS.ReadFile(f)
		if err != nil {
			return fmt.Errorf("lecture migration %s : %w", f, err)
		}
		for _, stmt := range splitStatements(string(data)) {
			if _, err := s.db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migration %s : %w", f, err)
			}
		}
	}
	return nil
}

func (s *SQLStore) Create(ctx context.Context, in TaskInput) (Task, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (title, done, created_at) VALUES (?, ?, ?)`,
		strings.TrimSpace(in.Title), boolToInt(in.Done), time.Now().UTC().Unix())
	if err != nil {
		return Task{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Task{}, err
	}
	return s.Get(ctx, id)
}

func (s *SQLStore) Get(ctx context.Context, id int64) (Task, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, title, done, created_at FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return t, err
}

func (s *SQLStore) List(ctx context.Context, f ListFilter) ([]Task, error) {
	// Construction dynamique : on n'ajoute le filtre « done » que s'il est demandé.
	query := `SELECT id, title, done, created_at FROM tasks`
	args := []any{}
	if f.Done != nil {
		query += ` WHERE done = ?`
		args = append(args, boolToInt(*f.Done))
	}
	limit, offset := normalizePage(f.Limit, f.Offset)
	query += ` ORDER BY id LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLStore) Update(ctx context.Context, id int64, in TaskInput) (Task, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET title = ?, done = ? WHERE id = ?`,
		strings.TrimSpace(in.Title), boolToInt(in.Done), id)
	if err != nil {
		return Task{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Task{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *SQLStore) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// withinTx exécute fn à l'intérieur d'une transaction. Si fn renvoie une erreur,
// la transaction est annulée (Rollback) et rien n'est écrit ; sinon elle est
// validée (Commit). C'est le squelette réutilisable de toute opération « tout ou
// rien » : la logique métier vit dans fn, la gestion du cycle de vie ici.
func (s *SQLStore) withinTx(ctx context.Context, fn func(*sql.Tx) error) (err error) {
	// BeginTx accepte un *sql.TxOptions (niveau d'isolation, lecture seule) ; nil
	// = les réglages par défaut du driver. Le contexte annule la transaction s'il
	// expire avant le Commit.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ouverture transaction : %w", err)
	}
	// Rollback en defer : filet de sécurité qui annule tout si l'on retourne sur
	// une erreur (ou une panique) avant le Commit. Après un Commit réussi, ce
	// Rollback est un no-op inoffensif (il renvoie sql.ErrTxDone, qu'on ignore).
	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err // le defer Rollback annule les écritures déjà faites
	}
	return tx.Commit()
}

// ArchiveTask déplace une tâche de « tasks » vers « tasks_archive » de façon
// atomique : soit l'insertion dans l'archive ET la suppression aboutissent toutes
// les deux, soit aucune. On lit puis insère AVANT de supprimer, pour que la
// moindre erreur laisse la tâche intacte dans « tasks » (le Rollback restaure
// l'état d'avant la transaction). Renvoie ErrNotFound si la tâche n'existe pas.
func (s *SQLStore) ArchiveTask(ctx context.Context, id int64) error {
	return s.withinTx(ctx, func(tx *sql.Tx) error {
		// 1. Lire la tâche DANS la transaction (vue cohérente avec les écritures).
		row := tx.QueryRowContext(ctx,
			`SELECT id, title, done, created_at FROM tasks WHERE id = ?`, id)
		t, err := scanTask(row)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		// 2. Insérer dans l'archive (requête paramétrée, jamais de concaténation — Ch. 47).
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO tasks_archive (id, title, done, created_at, archived_at)
			 VALUES (?, ?, ?, ?, ?)`,
			t.ID, t.Title, boolToInt(t.Done), t.CreatedAt.Unix(), time.Now().UTC().Unix()); err != nil {
			return err
		}
		// 3. Retirer de la table active. Si cet ExecContext échoue, le defer
		//    Rollback annule aussi l'INSERT ci-dessus : pas de doublon orphelin.
		if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id); err != nil {
			return err
		}
		return nil // withinTx valide (Commit)
	})
}

// scanner abstrait *sql.Row et *sql.Rows : tous deux ont une méthode Scan.
type scanner interface {
	Scan(dest ...any) error
}

// scanTask lit une ligne en convertissant les colonnes « portables » (entiers)
// vers les types Go attendus.
func scanTask(sc scanner) (Task, error) {
	var (
		t       Task
		doneInt int64
		created int64
	)
	if err := sc.Scan(&t.ID, &t.Title, &doneInt, &created); err != nil {
		return Task{}, err
	}
	t.Done = doneInt != 0
	t.CreatedAt = time.Unix(created, 0).UTC()
	return t, nil
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// normalizePage applique les mêmes bornes que paginate, mais renvoie les
// valeurs pour les passer à SQL (LIMIT/OFFSET).
func normalizePage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// splitStatements découpe un script SQL en instructions sur le « ; » final.
// Naïf mais suffisant pour des migrations simples (pas de « ; » dans une chaîne
// littérale) ; on ignore les lignes de commentaire « -- » et les blancs.
func splitStatements(script string) []string {
	var out []string
	for part := range strings.SplitSeq(script, ";") {
		var lines []string
		for line := range strings.SplitSeq(part, "\n") {
			if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "--") {
				lines = append(lines, line)
			}
		}
		if stmt := strings.TrimSpace(strings.Join(lines, "\n")); stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}
