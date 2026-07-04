package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// TestWriteFileAtomic vérifie qu'après une écriture atomique le contenu est
// exact et qu'aucun fichier temporaire résiduel (.tmp-*) ne subsiste.
func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")

	if err := writeFileAtomic(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("writeFileAtomic : %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile : %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("contenu = %q, veut %q", got, "hello")
	}

	// Un second passage doit remplacer le contenu, sans laisser de temporaire.
	if err := writeFileAtomic(path, []byte("world"), 0o644); err != nil {
		t.Fatalf("writeFileAtomic (2) : %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("répertoire = %d entrées, veut 1 (temporaire résiduel ?)", len(entries))
	}
}

// TestCountGoFiles s'appuie sur fstest.MapFS : le système de fichiers est en
// mémoire, le test ne touche jamais le disque et reste instantané.
func TestCountGoFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"main.go":            {Data: []byte("package main")},
		"pkg/a.go":           {Data: []byte("package pkg")},
		"pkg/b.go":           {Data: []byte("package pkg")},
		"pkg/data.txt":       {Data: []byte("non-go")},
		"docs/readme.md":     {Data: []byte("# doc")},
		"internal/util/u.go": {Data: []byte("package util")},
	}
	got, err := countGoFiles(fsys)
	if err != nil {
		t.Fatalf("countGoFiles : %v", err)
	}
	if want := 4; got != want {
		t.Errorf("countGoFiles = %d, veut %d", got, want)
	}
}

// TestListDir vérifie le tri et le suffixe "/" sur les répertoires.
func TestListDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "z.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	names, err := listDir(dir)
	if err != nil {
		t.Fatalf("listDir : %v", err)
	}
	if len(names) != 2 || names[0] != "sub/" || names[1] != "z.txt" {
		t.Errorf("listDir = %v, veut [sub/ z.txt]", names)
	}
}

// TestSafeReadUnderConfines vérifie que os.Root lit un fichier interne mais
// refuse de sortir du répertoire via une traversée "..".
func TestSafeReadUnderConfines(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("OpenRoot : %v", err)
	}
	defer root.Close()

	got, err := safeReadUnder(root, "in.txt")
	if err != nil {
		t.Fatalf("lecture interne refusée à tort : %v", err)
	}
	if string(got) != "ok" {
		t.Errorf("contenu = %q, veut %q", got, "ok")
	}

	if _, err := safeReadUnder(root, "../secret"); err == nil {
		t.Error("la traversée '../secret' aurait dû être refusée")
	}
}

// TestErrNotExist montre la bonne façon de tester l'absence : errors.Is sur la
// sentinelle fs.ErrNotExist, pas une comparaison de chaîne.
func TestErrNotExist(t *testing.T) {
	_, err := os.Open(filepath.Join(t.TempDir(), "absent"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, veut fs.ErrNotExist", err)
	}
}
