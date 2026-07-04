package site

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/gobook-site/internal/sommaire"
)

// mini-livre fixture : 1 partie, 2 pages, avec un lien interne à réécrire.
const fixtureSommaire = `# Sommaire

## Partie I — Test

- Ch. 0 — [Premier](chapitres/00.md)
- Ch. 1 — [Deuxième](chapitres/01.md)
`

const page0 = "# Premier chapitre\n\nVoir le [chapitre 1](01.md).\n\n## Une section\n\nTexte `code`.\n"
const page1 = "# Deuxième chapitre\n\nFin du mini-livre.\n"

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "SOMMAIRE.md"), fixtureSommaire)
	mustWrite(t, filepath.Join(dir, "chapitres", "00.md"), page0)
	mustWrite(t, filepath.Join(dir, "chapitres", "01.md"), page1)
	return dir
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildIntegration(t *testing.T) {
	src := writeFixture(t)
	out := t.TempDir()

	book, err := sommaire.Parse(strings.NewReader(fixtureSommaire), "Mini-livre")
	if err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Les assets réels sont à ../../assets relativement à ce paquet.
	assets := os.DirFS(filepath.Join("..", "..", "assets"))
	b, err := New(src, out, assets, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := b.Build(book); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Fichiers attendus.
	expect := []string{
		"chapitres/00.html",
		"chapitres/01.html",
		"index.html",
		"search.html",
		"search-index.json",
		"assets/css/layout.css",
		"assets/css/chroma.css",
		"assets/js/search.js",
		"assets/images/favicon-gopher.svg",
	}
	for _, rel := range expect {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("fichier manquant : %s (%v)", rel, err)
		}
	}

	// Réécriture du lien interne dans la sortie.
	got := readFile(t, filepath.Join(out, "chapitres", "00.html"))
	if !strings.Contains(got, `href="01.html"`) {
		t.Errorf("lien interne non réécrit dans la sortie")
	}
	// Le titre H1 a remplacé le titre provisoire du SOMMAIRE.
	if !strings.Contains(got, "Premier chapitre") {
		t.Errorf("titre H1 absent de la page")
	}
	// Navigation préc./suiv. présente.
	if !strings.Contains(got, "Deuxième chapitre") {
		t.Errorf("lien « suivant » absent")
	}
	// Sidebar listant les deux pages.
	if !strings.Contains(got, "Partie I — Test") {
		t.Errorf("sidebar/partie absente")
	}
}

func TestRelativeToRoot(t *testing.T) {
	cases := map[string]string{
		"index.html":            "",
		"chapitres/x.html":      "../",
		"projets/1/README.html": "../../",
	}
	for in, want := range cases {
		if got := relativeToRoot(in); got != want {
			t.Errorf("relativeToRoot(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
