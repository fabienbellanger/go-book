package cli

import (
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Go", "go"},
		{"Go,", "go"},
		{"\"hello\"", "hello"},
		{"...", ""},
		{"Été", "été"},
		{"v1.26", "v126"},
	}
	for _, tt := range tests {
		if got := normalize(tt.in); got != tt.want {
			t.Errorf("normalize(%q) = %q, voulu %q", tt.in, got, tt.want)
		}
	}
}

func TestTopWords(t *testing.T) {
	freq := map[string]int{"go": 3, "rust": 1, "java": 1, "python": 2}
	got := topWords(freq, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, voulu 2", len(got))
	}
	// "go" (3) puis "python" (2).
	if got[0].word != "go" || got[1].word != "python" {
		t.Errorf("top2 = %v, voulu [go python]", got)
	}
	// À égalité (java/rust = 1), l'ordre alphabétique tranche : ici hors top 2.
	all := topWords(freq, 0)
	if all[2].word != "java" || all[3].word != "rust" {
		t.Errorf("ordre à égalité = %v, voulu java avant rust", all)
	}
}

func TestRunFreq(t *testing.T) {
	in := "Go go GO rust\nGo, rust! python\n"
	out, _, code := run(t, in, "freq", "-n", "2")
	if code != 0 {
		t.Fatalf("code = %d, voulu 0", code)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("voulu 2 lignes, obtenu %d :\n%s", len(lines), out)
	}
	// go = 4, rust = 2.
	if !strings.Contains(lines[0], "go") || !strings.Contains(lines[0], "4") {
		t.Errorf("1re ligne = %q, voulu « 4 go »", lines[0])
	}
	if !strings.Contains(lines[1], "rust") || !strings.Contains(lines[1], "2") {
		t.Errorf("2e ligne = %q, voulu « 2 rust »", lines[1])
	}
}
