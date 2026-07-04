package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// mustOpen ouvre un pool isolé (un DSN distinct par test) pour que les tests
// ne partagent pas de données.
func mustOpen(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := openDB(context.Background(), dsn)
	if err != nil {
		t.Fatalf("openDB : %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestInsertAndGet(t *testing.T) {
	db := mustOpen(t, "test-insert")
	ctx := context.Background()

	id, err := insertUser(ctx, db, "Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("insert : %v", err)
	}
	if id != 1 {
		t.Fatalf("LastInsertId = %d, veut 1", id)
	}
	u, err := getUser(ctx, db, id)
	if err != nil {
		t.Fatalf("get : %v", err)
	}
	if u.Name != "Alice" || u.Email != "alice@example.com" {
		t.Fatalf("lu %+v", u)
	}
}

func TestErrNoRows(t *testing.T) {
	db := mustOpen(t, "test-norows")
	_, err := getUser(context.Background(), db, 42)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("veut sql.ErrNoRows, obtenu %v", err)
	}
}

func TestListAndRowsAffected(t *testing.T) {
	db := mustOpen(t, "test-list")
	ctx := context.Background()
	for _, name := range []string{"A", "B", "C"} {
		if _, err := insertUser(ctx, db, name, name+"@x"); err != nil {
			t.Fatal(err)
		}
	}
	users, err := listUsers(ctx, db)
	if err != nil {
		t.Fatalf("list : %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("veut 3 lignes, obtenu %d", len(users))
	}

	res, err := db.ExecContext(ctx,
		"update users set name = ? where id = ?", "Z", int64(2))
	if err != nil {
		t.Fatal(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("RowsAffected = %d, veut 1", n)
	}
}

func TestTxCommit(t *testing.T) {
	db := mustOpen(t, "test-commit")
	ctx := context.Background()
	a, _ := insertUser(ctx, db, "Alice", "a@x")
	b, _ := insertUser(ctx, db, "Bob", "b@x")

	if err := renameInTx(ctx, db, a, b, "A2", "B2"); err != nil {
		t.Fatalf("tx : %v", err)
	}
	ua, _ := getUser(ctx, db, a)
	ub, _ := getUser(ctx, db, b)
	if ua.Name != "A2" || ub.Name != "B2" {
		t.Fatalf("commit non appliqué : %+v %+v", ua, ub)
	}
}

func TestTxRollback(t *testing.T) {
	db := mustOpen(t, "test-rollback")
	ctx := context.Background()
	a, _ := insertUser(ctx, db, "Alice", "a@x")

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx,
		"update users set name = ? where id = ?", "CHANGED", a); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	u, _ := getUser(ctx, db, a)
	if u.Name != "Alice" {
		t.Fatalf("rollback non respecté : name = %q", u.Name)
	}
}

func TestContextCanceled(t *testing.T) {
	db := mustOpen(t, "test-ctx")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // contexte déjà annulé

	_, err := db.ExecContext(ctx,
		"insert into users(name, email) values (?, ?)", "X", "x@x")
	if err == nil {
		t.Fatal("attendu une erreur de contexte annulé")
	}
}
