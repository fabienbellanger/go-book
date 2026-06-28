package analyze

import (
	"reflect"
	"testing"
)

const sample = "Le chat dort. Le CHAT mange ; un chien dort. Un chat ! 42 fois 42."

func TestTopWordsBasic(t *testing.T) {
	got := TopWordsScan(sample, 3)
	// À égalité de compte, ordre lexicographique : "42" < "dort" < "le" < "un".
	want := []Count{
		{Word: "chat", Count: 3},
		{Word: "42", Count: 2},
		{Word: "dort", Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TopWordsScan = %+v\nvoulu %+v", got, want)
	}
}

// TestImplementationsAgree est le filet de sécurité du refactor : la version
// optimisée doit renvoyer EXACTEMENT le même résultat que la version naïve.
// C'est ce qui autorise à profiler/optimiser sereinement.
func TestImplementationsAgree(t *testing.T) {
	texts := []string{
		sample,
		"",
		"un",
		"a a a b b c",
		"Éléphant éléphant ÉLÉPHANT déjà Déjà",
		"mix3d alpha2 alpha2 123 123 123",
	}
	for _, txt := range texts {
		a := TopWordsRegexp(txt, -1)
		b := TopWordsScan(txt, -1)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("désaccord sur %q :\n  regexp = %+v\n  scan   = %+v", txt, a, b)
		}
	}
}

func TestTopNLimit(t *testing.T) {
	if got := TopWordsScan(sample, 1); len(got) != 1 || got[0].Word != "chat" {
		t.Errorf("top 1 = %+v, voulu [{chat 3}]", got)
	}
	if got := TopWordsScan(sample, 0); len(got) != 0 {
		t.Errorf("top 0 devrait être vide, obtenu %+v", got)
	}
	if got := TopWordsScan(sample, 999); len(got) == 0 {
		t.Error("n > nombre de mots devrait renvoyer tous les mots")
	}
}
