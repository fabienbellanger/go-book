package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

// mustEqual est une assertion partagée. t.Helper() fait pointer l'échec sur la
// ligne de l'APPELANT (le cas de test) plutôt que sur cette ligne interne.
func mustEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q ; want %q", got, want)
	}
}

// TestSlugify est table-driven : une table de cas, un sous-test (t.Run) par cas.
func TestSlugify(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Hello, World!", "hello-world"},
		{"ponctuation", "  Go 1.26 : Top!  ", "go-1-26-top"},
		{"déjà slug", "go-1-26-top", "go-1-26-top"},
		{"vide", "", ""},
		{"que des séparateurs", "  ---  ", ""},
		{"chiffres", "Room 101", "room-101"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mustEqual(t, Slugify(tc.in), tc.want)
		})
	}
}

// ExampleSlugify est À LA FOIS doc et test : la sortie est comparée à // Output:.
func ExampleSlugify() {
	fmt.Println(Slugify("Hello, World!"))
	fmt.Println(Slugify("  Go 1.26 : Top!  "))
	// Output:
	// hello-world
	// go-1-26-top
}

// TestSaveLines exerce t.TempDir : un répertoire isolé, effacé automatiquement.
func TestSaveLines(t *testing.T) {
	dir := t.TempDir()
	path, err := SaveLines(dir, "out.txt", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("SaveLines: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "a\nb\nc\n" {
		t.Errorf("contenu = %q ; attendu \"a\\nb\\nc\\n\"", got)
	}
}

// TestCleanupLIFO vérifie que t.Cleanup s'exécute en ordre LIFO. On l'observe via
// un sous-test : ses cleanups tournent dès qu'il se termine, avant de rendre la main.
func TestCleanupLIFO(t *testing.T) {
	var order []int
	t.Run("ressource", func(t *testing.T) {
		t.Cleanup(func() { order = append(order, 1) }) // enregistré en 1er -> exécuté en dernier
		t.Cleanup(func() { order = append(order, 2) }) // enregistré en 2nd -> exécuté en premier
	})
	if !slices.Equal(order, []int{2, 1}) {
		t.Errorf("ordre des cleanups = %v ; attendu [2 1] (LIFO)", order)
	}
}

// TestObservability exerce les nouveautés d'observabilité des tests.
func TestObservability(t *testing.T) {
	t.Attr("ticket", "GO-1234")                       // 🆕 1.25 : métadonnée
	fmt.Fprintln(t.Output(), "diagnostic via Output") // 🆕 1.25 : writer indenté
	if dir := t.ArtifactDir(); dir == "" {            // 🆕 1.26 : dossier d'artefacts
		t.Error("ArtifactDir ne devrait pas être vide")
	}
}

// --- Benchmark & fuzzing (détail au Ch. 36) ---

var sink string

func BenchmarkSlugify(b *testing.B) {
	for b.Loop() { // 🆕 1.24
		sink = Slugify("  Go 1.26 : Maîtriser les Tests !  ")
	}
}

func FuzzSlugify(f *testing.F) {
	f.Add("Hello, World!") // graines du corpus
	f.Add("  Go 1.26 : Top!  ")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		out := Slugify(s)
		// Invariant 1 : idempotence.
		if again := Slugify(out); again != out {
			t.Errorf("non idempotent : Slugify(%q)=%q puis %q", s, out, again)
		}
		// Invariant 2 : ni tiret en bordure, ni tiret doublé.
		if strings.HasPrefix(out, "-") || strings.HasSuffix(out, "-") {
			t.Errorf("tiret en bordure : %q", out)
		}
		if strings.Contains(out, "--") {
			t.Errorf("tiret doublé : %q", out)
		}
	})
}
