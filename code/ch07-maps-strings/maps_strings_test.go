package main

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestWordCount(t *testing.T) {
	got := wordCount("Go go GO rust Rust")
	want := map[string]int{"go": 3, "rust": 2}
	if !maps.Equal(got, want) {
		t.Errorf("wordCount = %v ; attendu %v", got, want)
	}
	// Texte vide -> map vide (non nil), len 0.
	if got := wordCount("   "); len(got) != 0 {
		t.Errorf("wordCount(vide) = %v ; attendu une map vide", got)
	}
}

func TestUniqueSorted(t *testing.T) {
	got := uniqueSorted("le chat le chien le chat")
	want := []string{"chat", "chien", "le"}
	if !slices.Equal(got, want) {
		t.Errorf("uniqueSorted = %v ; attendu %v (trié, distinct)", got, want)
	}
}

func TestReverseString(t *testing.T) {
	cases := map[string]string{
		"":     "",
		"go":   "og",
		"café": "éfac", // l'accent reste intact (reverse par runes)
		"Go🚀!": "!🚀oG", // l'emoji (4 octets) n'est pas découpé
	}
	for in, want := range cases {
		if got := reverseString(in); got != want {
			t.Errorf("reverseString(%q) = %q ; attendu %q", in, got, want)
		}
		// Involution : inverser deux fois redonne l'original.
		if got := reverseString(reverseString(in)); got != in {
			t.Errorf("reverseString∘reverseString(%q) = %q ; attendu %q", in, got, in)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		max  int
		want string
	}{
		{"bonjour", 100, "bonjour"}, // plus court que max -> inchangé
		{"bonjour", 3, "bon…"},      // coupe + ellipse
		{"café", 3, "caf…"},         // compte en runes, pas en octets
		{"café", 4, "café"},         // 4 runes -> pas de coupe (même si 5 octets)
		{"x", 0, ""},                // max <= 0
	}
	for _, c := range cases {
		if got := truncate(c.s, c.max); got != c.want {
			t.Errorf("truncate(%q, %d) = %q ; attendu %q", c.s, c.max, got, c.want)
		}
	}
}

// TestCutLast couvre strings.CutLast (🆕 Go 1.27) : découpe autour de la
// DERNIÈRE occurrence du séparateur, pendant de strings.Cut.
func TestCutLast(t *testing.T) {
	cases := []struct {
		name              string
		in                string
		wantBase, wantExt string
		wantOK            bool
	}{
		{"extension simple", "rapport.pdf", "rapport", "pdf", true},
		{"plusieurs points", "archive.tar.gz", "archive.tar", "gz", true},
		{"sans point", "LICENSE", "LICENSE", "", false},
		{"point final", "fin.", "fin", "", true},
		{"point initial", ".gitignore", "", "gitignore", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base, ext, ok := splitName(c.in)
			if base != c.wantBase || ext != c.wantExt || ok != c.wantOK {
				t.Errorf("splitName(%q) = (%q, %q, %t) ; attendu (%q, %q, %t)",
					c.in, base, ext, ok, c.wantBase, c.wantExt, c.wantOK)
			}
		})
	}
}

// TestCutVsCutLast montre la différence sur une même entrée : Cut prend la
// première occurrence, CutLast la dernière.
func TestCutVsCutLast(t *testing.T) {
	const in = "a=b=c"

	if cle, val, _ := firstKeyValue(in); cle != "a" || val != "b=c" {
		t.Errorf("Cut(%q) = (%q, %q) ; attendu (\"a\", \"b=c\")", in, cle, val)
	}
	if avant, apres, _ := strings.CutLast(in, "="); avant != "a=b" || apres != "c" {
		t.Errorf("CutLast(%q) = (%q, %q) ; attendu (\"a=b\", \"c\")", in, avant, apres)
	}
}
