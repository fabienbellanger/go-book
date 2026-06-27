package main

import (
	"errors"
	"fmt"
	"strings"
)

// ErrEmptyKey est une erreur SENTINELLE : une valeur exportée et stable que les
// appelants peuvent reconnaître avec errors.Is.
var ErrEmptyKey = errors.New("clé vide")

// ParseError est un TYPE d'erreur : il transporte du contexte (le numéro de ligne)
// et enveloppe la cause. Sa méthode Unwrap l'insère dans la chaîne d'erreurs, ce qui
// rend errors.Is et errors.As capables de la traverser.
type ParseError struct {
	Line int
	Err  error
}

func (e *ParseError) Error() string { return fmt.Sprintf("ligne %d: %v", e.Line, e.Err) }
func (e *ParseError) Unwrap() error { return e.Err }

// parseLine découpe une ligne "clé=valeur". En cas d'échec, elle renvoie un
// *ParseError enveloppant la cause précise (séparateur manquant ou clé vide).
func parseLine(line int, s string) (key, value string, err error) {
	k, v, found := strings.Cut(s, "=") // 🆕 1.18 : strings.Cut
	if !found {
		return "", "", &ParseError{Line: line, Err: fmt.Errorf("séparateur '=' manquant dans %q", s)}
	}
	k = strings.TrimSpace(k)
	if k == "" {
		return "", "", &ParseError{Line: line, Err: ErrEmptyKey}
	}
	return k, strings.TrimSpace(v), nil
}

// parseConfig analyse plusieurs lignes et AGRÈGE toutes les erreurs rencontrées avec
// errors.Join, au lieu de s'arrêter à la première. Renvoie nil si tout est valide.
func parseConfig(lines []string) (map[string]string, error) {
	cfg := make(map[string]string)
	var errs []error
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // on ignore les lignes vides
		}
		k, v, err := parseLine(i+1, line)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		cfg[k] = v
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...) // nil si errs est vide
	}
	return cfg, nil
}
