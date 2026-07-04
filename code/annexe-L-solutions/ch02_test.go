package main

import "testing"

func TestCh02Greet(t *testing.T) {
	if got := ch02Greet("de", "Go"); got != "Hallo, Go !" {
		t.Errorf("de : got %q", got)
	}
	if got := ch02Greet("fr", "Go"); got != "Bonjour, Go !" {
		t.Errorf("fr : got %q", got)
	}
	// Langue inconnue : repli sur l'anglais.
	if got := ch02Greet("xx", "Go"); got != "Hello, Go !" {
		t.Errorf("repli : got %q", got)
	}
}
