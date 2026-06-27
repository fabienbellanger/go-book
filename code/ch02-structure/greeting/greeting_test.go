package greeting

import "testing"

// TestGreet vérifie Greet, y compris le repli sur la langue par défaut.
func TestGreet(t *testing.T) {
	cases := []struct {
		lang, name, want string
	}{
		{"fr", "Go", "Bonjour, Go !"},
		{"en", "Go", "Hello, Go !"},
		{"es", "Go", "Hola, Go !"},
		{"xx", "Go", "Bonjour, Go !"}, // langue inconnue -> repli sur fr
	}

	for _, c := range cases {
		if got := Greet(c.lang, c.name); got != c.want {
			t.Errorf("Greet(%q, %q) = %q ; attendu %q", c.lang, c.name, got, c.want)
		}
	}
}
