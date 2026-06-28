package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTokenCompare vérifie qu'un jeton est égal à lui-même mais pas à un autre,
// et que la comparaison reste en temps constant (peu importe ici la durée, on
// teste la correction logique).
func TestTokenCompare(t *testing.T) {
	tok := newToken()
	if tok == "" {
		t.Fatal("jeton vide")
	}
	if !equalTokens(tok, tok) {
		t.Error("un jeton devrait être égal à lui-même")
	}
	if equalTokens(tok, newToken()) {
		t.Error("deux jetons distincts ne devraient pas être égaux")
	}
}

// TestHTMLEscaping verrouille le réflexe anti-XSS : html/template échappe,
// text/template non.
func TestHTMLEscaping(t *testing.T) {
	const payload = "<script>alert(1)</script>"

	html, err := renderUserHTML(payload)
	if err != nil {
		t.Fatalf("renderUserHTML: %v", err)
	}
	if strings.Contains(html, "<script>") {
		t.Errorf("html/template aurait dû échapper la balise : %q", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("échappement HTML attendu, obtenu : %q", html)
	}

	text, err := renderUserText(payload)
	if err != nil {
		t.Fatalf("renderUserText: %v", err)
	}
	if !strings.Contains(text, "<script>") {
		t.Errorf("text/template ne devrait rien échapper (démonstration du piège) : %q", text)
	}
}

// TestSafeRelPath couvre la validation des chemins relatifs.
func TestSafeRelPath(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"notes.txt", true},
		{"sub/notes.txt", true},
		{"../etc/passwd", false},
		{"/etc/passwd", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isSafeRelPath(c.name); got != c.want {
			t.Errorf("isSafeRelPath(%q) = %t, voulu %t", c.name, got, c.want)
		}
	}
}

// TestReadWithinRoot prouve qu'os.Root confine l'accès : on lit un fichier interne
// mais une tentative de traversée « ../ » échoue, même si la cible existe dehors.
func TestReadWithinRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Une cible « secrète » HORS du sous-arbre confiné.
	parent := filepath.Dir(dir)
	secret := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(secret) })

	got, err := readWithinRoot(dir, "notes.txt")
	if err != nil {
		t.Fatalf("lecture interne refusée à tort : %v", err)
	}
	if string(got) != "ok" {
		t.Errorf("contenu = %q, voulu \"ok\"", got)
	}

	if _, err := readWithinRoot(dir, "../secret.txt"); err == nil {
		t.Error("la traversée « ../secret.txt » aurait dû être refusée par os.Root")
	}
}

// TestHardenedTLS vérifie que la configuration impose au moins TLS 1.2 et ne
// désactive pas la vérification de certificat.
func TestHardenedTLS(t *testing.T) {
	cfg := hardenedTLSConfig()
	if cfg.MinVersion < tls12 {
		t.Errorf("MinVersion = 0x%04x, voulu >= TLS 1.2 (0x%04x)", cfg.MinVersion, tls12)
	}
	if cfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify ne doit jamais être activé")
	}
}

// tls12 duplique tls.VersionTLS12 pour garder le test lisible sans import en plus.
const tls12 = 0x0303
