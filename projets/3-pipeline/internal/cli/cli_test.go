package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile crée un fichier temporaire avec un contenu donné et renvoie son chemin.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// run exécute Run avec un stdin donné et capture stdout, stderr et le code.
func run(t *testing.T, stdin string, args ...string) (out, errOut string, code int) {
	t.Helper()
	var bo, be bytes.Buffer
	code = Run(context.Background(), args, strings.NewReader(stdin), &bo, &be)
	return bo.String(), be.String(), code
}

func TestRunHashesFiles(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "alpha")
	b := writeFile(t, dir, "b.txt", "beta")

	// Empreinte attendue pour « alpha », calculée indépendamment.
	wantA := sha256.Sum256([]byte("alpha"))
	wantHexA := hex.EncodeToString(wantA[:])

	out, _, code := run(t, "", "-j", "2", a, b)
	if code != 0 {
		t.Fatalf("code = %d, voulu 0", code)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lignes, voulu 2 : %q", len(lines), out)
	}
	// Sortie triée par chemin : a.txt vient avant b.txt.
	if !strings.HasPrefix(lines[0], wantHexA) || !strings.HasSuffix(lines[0], "a.txt") {
		t.Errorf("ligne 0 = %q, empreinte ou chemin inattendu", lines[0])
	}
}

func TestRunReadsStdin(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "f.txt", "x")

	out, _, code := run(t, p+"\n", "-j", "1")
	if code != 0 {
		t.Fatalf("code = %d, voulu 0", code)
	}
	if !strings.Contains(out, "f.txt") {
		t.Errorf("stdout = %q, doit contenir f.txt", out)
	}
}

func TestRunMissingFile(t *testing.T) {
	// Un fichier inexistant fait échouer l'étape : errgroup annule, code 1.
	_, errOut, code := run(t, "", filepath.Join(t.TempDir(), "absent.txt"))
	if code != 1 {
		t.Fatalf("code = %d, voulu 1", code)
	}
	if !strings.Contains(errOut, "pipe :") {
		t.Errorf("stderr = %q, doit signaler l'erreur", errOut)
	}
}

func TestRunUsageError(t *testing.T) {
	if _, _, code := run(t, "", "-nawak"); code != 2 {
		t.Errorf("flag inconnu : code = %d, voulu 2", code)
	}
}

func TestRunVersion(t *testing.T) {
	out, _, code := run(t, "", "-version")
	if code != 0 || !strings.Contains(out, "pipe ") {
		t.Errorf("version : code=%d out=%q", code, out)
	}
}
