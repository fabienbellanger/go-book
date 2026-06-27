package main

import "testing"

// TestGreet vérifie le message produit par greet (test table-driven, voir Ch. 13).
func TestGreet(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "prénom simple", in: "Go", want: "Bonjour, Go ! 👋"},
		{name: "chaîne vide", in: "", want: "Bonjour,  ! 👋"},
		{name: "accents", in: "Amélie", want: "Bonjour, Amélie ! 👋"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := greet(c.in)
			if got != c.want {
				t.Errorf("greet(%q) = %q ; attendu %q", c.in, got, c.want)
			}
		})
	}
}
