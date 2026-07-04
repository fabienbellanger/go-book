package main

import (
	"errors"
	"fmt"
	"strings"
)

// ch10ErrEmptyKey est une erreur sentinelle comparable via errors.Is.
var ch10ErrEmptyKey = errors.New("clé vide")

// ch10ParseError décrit une erreur d'analyse localisée à une ligne.
type ch10ParseError struct {
	Line int
	Err  error
}

func (e *ch10ParseError) Error() string {
	return fmt.Sprintf("ligne %d : %v", e.Line, e.Err)
}

func (e *ch10ParseError) Unwrap() error { return e.Err }

// Is considère deux *ch10ParseError égales si elles concernent la MÊME ligne
// (exercice 2). errors.Is consulte cette méthode avant de comparer les valeurs.
func (e *ch10ParseError) Is(target error) bool {
	var pe *ch10ParseError
	if !errors.As(target, &pe) {
		return false
	}
	return e.Line == pe.Line
}

// ch10ParseLine découpe "clé=valeur". Le `%w` PRÉSERVE la chaîne : errors.Is
// remonte jusqu'à ch10ErrEmptyKey (exercice 1 : `%v` la romprait et Is
// renverrait false).
func ch10ParseLine(line int, s string) (string, string, error) {
	k, v, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return "", "", &ch10ParseError{Line: line, Err: fmt.Errorf("%q : %w", s, ch10ErrEmptyKey)}
	}
	return k, v, nil
}
