package main

import (
	"io/fs"
	"strings"
	"testing"
)

// TestEmbeddedVersion vérifie que le fichier version.txt est bien embarqué dans
// la string au moment de la compilation (pas lu sur disque à l'exécution).
func TestEmbeddedVersion(t *testing.T) {
	if got := strings.TrimSpace(embeddedVersion); got != "1.0.0" {
		t.Fatalf("version embarquée = %q, voulu %q", got, "1.0.0")
	}
}

// TestTemplatesFS parcourt le système de fichiers embarqué et confirme la
// présence du gabarit. embed.FS implémente fs.FS : tout l'écosystème io/fs
// (fs.ReadFile, fs.WalkDir, fs.Sub, http.FileServerFS) fonctionne dessus.
func TestTemplatesFS(t *testing.T) {
	data, err := fs.ReadFile(templatesFS, "templates/welcome.txt")
	if err != nil {
		t.Fatalf("lecture de l'asset embarqué: %v", err)
	}
	if !strings.Contains(string(data), "{{.Name}}") {
		t.Errorf("gabarit inattendu: %q", data)
	}
}

// TestRenderWelcome exécute le gabarit embarqué de bout en bout.
func TestRenderWelcome(t *testing.T) {
	msg, err := renderWelcome("Ada")
	if err != nil {
		t.Fatalf("rendu: %v", err)
	}
	if !strings.Contains(msg, "Bonjour, Ada !") {
		t.Errorf("message rendu = %q", msg)
	}
}

// TestFeatureDefault valide le comportement SANS le tag « prod » : c'est
// feature_default.go qui est compilé. Lancé avec « -tags prod », ce test
// devrait au contraire voir la variante de production.
func TestFeatureDefault(t *testing.T) {
	if got := featureName(); !strings.HasPrefix(got, "dev") {
		t.Errorf("featureName() = %q, voulu la variante de dév (recompilé avec -tags prod ?)", got)
	}
}

// TestBuildVersion confirme la stratégie de repli : sans injection -ldflags,
// version vaut "dev" et buildVersion() se rabat sur le fichier embarqué.
func TestBuildVersion(t *testing.T) {
	if version == "dev" {
		if got := buildVersion(); got != "1.0.0" {
			t.Errorf("buildVersion() = %q, voulu le repli embarqué %q", got, "1.0.0")
		}
	}
}
