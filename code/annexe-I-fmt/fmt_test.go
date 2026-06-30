package main

import (
	"errors"
	"fmt"
	"testing"
)

// TestVerbes vérifie la sortie exacte des principaux verbes : c'est la table de
// référence de l'annexe, rendue exécutable.
func TestVerbes(t *testing.T) {
	p := point{X: 3, Y: 4}
	const han rune = 0x4E2D // le caractère 中
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"%v struct", fmt.Sprintf("%v", p), "{3 4}"},
		{"%+v struct", fmt.Sprintf("%+v", p), "{X:3 Y:4}"},
		{"%#v struct", fmt.Sprintf("%#v", p), "main.point{X:3, Y:4}"},
		{"%T", fmt.Sprintf("%T", p), "main.point"},
		{"Stringer %v", fmt.Sprintf("%v", blue), "bleu"},
		{"%t", fmt.Sprintf("%t", true), "true"},
		{"%d", fmt.Sprintf("%d", 255), "255"},
		{"%b", fmt.Sprintf("%b", 255), "11111111"},
		{"%o", fmt.Sprintf("%o", 255), "377"},
		{"%x", fmt.Sprintf("%x", 255), "ff"},
		{"%X", fmt.Sprintf("%X", 255), "FF"},
		{"%c", fmt.Sprintf("%c", han), "中"},
		{"%q rune", fmt.Sprintf("%q", han), "'中'"},
		{"%U", fmt.Sprintf("%U", han), "U+4E2D"},
		{"%.2f", fmt.Sprintf("%.2f", 1234.5678), "1234.57"},
		{"%g", fmt.Sprintf("%g", 1234.5678), "1234.5678"},
		{"%s", fmt.Sprintf("%s", "Go"), "Go"},
		{"%q string", fmt.Sprintf("%q", "Go"), `"Go"`},
		{"%x string", fmt.Sprintf("%x", "Go"), "476f"},
		{"largeur %6d", fmt.Sprintf("[%6d]", 42), "[    42]"},
		{"gauche %-6d", fmt.Sprintf("[%-6d]", 42), "[42    ]"},
		{"zéros %06d", fmt.Sprintf("[%06d]", 42), "[000042]"},
		{"signe %+d", fmt.Sprintf("%+d", 42), "+42"},
		{"précision %.3s", fmt.Sprintf("%.3s", "abcdef"), "abc"},
		{"largeur '*'", fmt.Sprintf("[%*d]", 6, 42), "[    42]"},
		{"index args", fmt.Sprintf("%[2]d %[1]d", 7, 9), "9 7"},
		{"verbe inadapté", describeBadVerb(), "%!d(string=texte)"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s : got %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestErrorfW vérifie que %w enveloppe l'erreur sous-jacente (errors.Is/As).
func TestErrorfW(t *testing.T) {
	base := errors.New("disque plein")
	wrapped := fmt.Errorf("écriture du cache : %w", base)
	if !errors.Is(wrapped, base) {
		t.Errorf("errors.Is devrait retrouver l'erreur enveloppée par %%w")
	}
}
