package main

import "testing"

func TestCh13Slugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":        "hello-world",
		"  Go 1.26  ":        "go-1-26",
		"Étude de Cas Élevé": "etude-de-cas-eleve", // exercice 1 : accents repliés
		"--déjà--slug--":     "deja-slug",
	}
	for in, want := range cases {
		if got := ch13Slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, veut %q", in, got, want)
		}
	}
}

// FuzzCh13Slugify vérifie l'IDEMPOTENCE : re-slugifier un slug ne le change pas
// (exercice 3). Le fuzzer cherche une entrée qui violerait cette invariance.
func FuzzCh13Slugify(f *testing.F) {
	f.Add("Hello World")
	f.Add("café *** 42")
	f.Fuzz(func(t *testing.T, s string) {
		once := ch13Slugify(s)
		twice := ch13Slugify(once)
		if once != twice {
			t.Errorf("non idempotent : %q -> %q -> %q", s, once, twice)
		}
	})
}
