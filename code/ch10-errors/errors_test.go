package main

import (
	"errors"
	"testing"
)

func TestParseLineValid(t *testing.T) {
	k, v, err := parseLine(1, "  host = localhost  ")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if k != "host" || v != "localhost" {
		t.Errorf("parseLine = (%q, %q) ; attendu (\"host\", \"localhost\")", k, v)
	}
}

func TestParseLineWrapsSentinel(t *testing.T) {
	_, _, err := parseLine(7, "   =x") // clé vide
	if err == nil {
		t.Fatal("attendu une erreur sur clé vide")
	}
	// errors.Is traverse le *ParseError jusqu'à la sentinelle.
	if !errors.Is(err, ErrEmptyKey) {
		t.Errorf("errors.Is(err, ErrEmptyKey) = false ; attendu true")
	}
	// errors.As récupère le type concret et son contexte.
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("errors.As n'a pas reconnu *ParseError (%T)", err)
	}
	if pe.Line != 7 {
		t.Errorf("Line = %d ; attendu 7", pe.Line)
	}
}

func TestAsType126(t *testing.T) {
	_, _, err := parseLine(2, "noEqualSign")
	// 🆕 1.26 : variante générique d'errors.As.
	pe, ok := errors.AsType[*ParseError](err)
	if !ok {
		t.Fatalf("AsType[*ParseError] = false ; attendu true (%T)", err)
	}
	if pe.Line != 2 {
		t.Errorf("Line = %d ; attendu 2", pe.Line)
	}
	// Cas négatif : un type non présent dans l'arbre.
	if _, ok := errors.AsType[*ParseError](errors.New("autre")); ok {
		t.Error("AsType ne devrait pas reconnaître une erreur simple comme *ParseError")
	}
}

func TestParseConfigJoin(t *testing.T) {
	cfg, err := parseConfig([]string{
		"host = localhost", // ok
		"bad-line",         // séparateur manquant
		"  = oops",         // clé vide
	})
	if cfg != nil {
		t.Errorf("cfg = %v ; attendu nil quand il y a des erreurs", cfg)
	}
	if err == nil {
		t.Fatal("attendu une erreur agrégée")
	}
	// L'erreur jointe contient toujours la sentinelle.
	if !errors.Is(err, ErrEmptyKey) {
		t.Error("l'erreur jointe devrait contenir ErrEmptyKey")
	}
}

func TestParseConfigValid(t *testing.T) {
	cfg, err := parseConfig([]string{"a = 1", "", "b = 2"}) // ligne vide ignorée
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(cfg) != 2 || cfg["a"] != "1" || cfg["b"] != "2" {
		t.Errorf("cfg = %v ; attendu {a:1, b:2}", cfg)
	}
}
