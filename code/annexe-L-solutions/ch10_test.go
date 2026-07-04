package main

import (
	"errors"
	"testing"
)

func TestCh10WrapChain(t *testing.T) {
	_, _, err := ch10ParseLine(3, "=valeur")
	// Le %w préserve la chaîne : errors.Is remonte jusqu'à la sentinelle.
	if !errors.Is(err, ch10ErrEmptyKey) {
		t.Errorf("errors.Is(err, ErrEmptyKey) = false, chaîne rompue ? err=%v", err)
	}
}

func TestCh10ParseErrorIs(t *testing.T) {
	err := &ch10ParseError{Line: 7, Err: errors.New("x")}
	// Is compare sur la ligne : même ligne -> égales.
	if !errors.Is(err, &ch10ParseError{Line: 7}) {
		t.Error("même ligne devrait matcher")
	}
	if errors.Is(err, &ch10ParseError{Line: 8}) {
		t.Error("ligne différente ne devrait pas matcher")
	}
}
