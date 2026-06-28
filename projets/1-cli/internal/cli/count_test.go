package cli

import (
	"strings"
	"testing"
)

func TestCountSource(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want counts
	}{
		{"vide", "", counts{}},
		{"un mot sans newline", "hello", counts{lines: 0, words: 1, runes: 5, bytes: 5}},
		{"une ligne", "hello world\n", counts{lines: 1, words: 2, runes: 12, bytes: 12}},
		{"espaces multiples", "  a   b  \n", counts{lines: 1, words: 2, runes: 10, bytes: 10}},
		// "café" = 4 runes mais 5 octets (é = 2 octets en UTF-8) ; "🚀" = 1 rune / 4 octets.
		{"utf8", "café 🚀\n", counts{lines: 1, words: 2, runes: 7, bytes: 11}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := source{
				name: tt.name,
				open: readCloserFrom(tt.in),
			}
			got := countSource(s)
			if got.err != nil {
				t.Fatalf("erreur inattendue : %v", got.err)
			}
			if got.counts != tt.want {
				t.Errorf("counts = %+v, voulu %+v", got.counts, tt.want)
			}
		})
	}
}

func TestRunCountStdinTotal(t *testing.T) {
	// Une seule source (stdin) : pas de ligne « total » même avec -total.
	out, _, code := run(t, "a b c\n", "count")
	if code != 0 {
		t.Fatalf("code = %d, voulu 0", code)
	}
	if strings.Contains(out, "total") {
		t.Errorf("pas de total attendu pour une seule source, sortie :\n%s", out)
	}
	// 1 ligne, 3 mots, 6 runes, 6 octets.
	if !strings.Contains(out, "1") || !strings.Contains(out, "3") {
		t.Errorf("compteurs absents de la sortie :\n%s", out)
	}
}

func TestRunCountMissingFile(t *testing.T) {
	// Un fichier inexistant : code 1, message sur stderr.
	_, errOut, code := run(t, "", "count", "fichier-qui-nexiste-pas.txt")
	if code != 1 {
		t.Fatalf("code = %d, voulu 1", code)
	}
	if !strings.Contains(errOut, "fichier-qui-nexiste-pas.txt") {
		t.Errorf("stderr doit nommer le fichier fautif : %q", errOut)
	}
}
