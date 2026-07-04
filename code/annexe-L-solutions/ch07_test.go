package main

import (
	"testing"
	"unicode/utf8"
)

func TestCh07ReverseString(t *testing.T) {
	got := ch07ReverseString("café")
	if got != "éfac" {
		t.Errorf("reverse = %q, veut %q", got, "éfac")
	}
	// La sortie reste de l'UTF-8 valide (contrairement à une inversion par octets).
	if !utf8.ValidString(got) {
		t.Errorf("%q n'est pas de l'UTF-8 valide", got)
	}
}

func TestCh07WordFrequencies(t *testing.T) {
	freqs := ch07WordFrequencies("le chat le chien le")
	// "le" (3) en tête, puis chat/chien (1) départagés par ordre alphabétique.
	want := []ch07WordFreq{{"le", 3}, {"chat", 1}, {"chien", 1}}
	if len(freqs) != len(want) {
		t.Fatalf("got %v", freqs)
	}
	for i := range want {
		if freqs[i] != want[i] {
			t.Errorf("rang %d : got %v, veut %v", i, freqs[i], want[i])
		}
	}
}
