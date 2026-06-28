package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/enumgen/internal/cli"
)

const sample = `package weekday

//enumgen:stringer trimprefix=Day
type Day int

const (
	DayMon Day = iota
	DayTue
	DayWed
)
`

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, err bytes.Buffer
	code = cli.Run(args, &out, &err)
	return code, out.String(), err.String()
}

func TestRunDryRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "day.go"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := run(t, "-dir", dir, "-n")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, "func (v Day) String() string") {
		t.Errorf("stdout n'a pas le String() attendu :\n%s", stdout)
	}
	// -n ne doit écrire aucun fichier.
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("-n a écrit un fichier : %d entrées", len(entries))
	}
}

func TestRunWritesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "day.go"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := run(t, "-dir", dir)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	got, err := os.ReadFile(filepath.Join(dir, "weekday_enum.go"))
	if err != nil {
		t.Fatalf("fichier de sortie absent : %v", err)
	}
	if !strings.Contains(string(got), `DayMon: "Mon"`) {
		t.Errorf("contenu inattendu :\n%s", got)
	}
}

func TestRunNoAnnotation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\n\ntype T int\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := run(t, "-dir", dir)
	if code != 1 {
		t.Errorf("code = %d, voulu 1", code)
	}
	if !strings.Contains(stderr, "aucun type annoté") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestRunVersion(t *testing.T) {
	code, stdout, _ := run(t, "-version")
	if code != 0 || !strings.HasPrefix(stdout, "enumgen ") {
		t.Errorf("version : code=%d stdout=%q", code, stdout)
	}
}

func TestRunBadFlag(t *testing.T) {
	code, _, _ := run(t, "-nope")
	if code != 2 {
		t.Errorf("drapeau inconnu : code = %d, voulu 2", code)
	}
}
