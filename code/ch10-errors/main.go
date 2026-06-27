// Démonstrations du chapitre 10 : erreurs comme valeurs, wrapping %w, errors.Is/As,
// errors.AsType (1.26), errors.Join. Lancement : depuis code/, `go run ./ch10-errors`
package main

import (
	"errors"
	"fmt"
)

func main() {
	// =========================================================================
	// Erreur de base + wrapping %w + Unwrap
	// =========================================================================

	base := errors.New("disque plein")
	wrapped := fmt.Errorf("écriture de config.txt: %w", base) // %w ENVELOPPE base
	fmt.Printf("wrap   : %v\n", wrapped)
	fmt.Printf("wrap   : Unwrap == base ? %t\n", errors.Unwrap(wrapped) == base)

	// =========================================================================
	// errors.Is : reconnaître une SENTINELLE à travers la chaîne
	// =========================================================================

	_, _, err := parseLine(3, "  =valeur") // clé vide -> ParseError{Err: ErrEmptyKey}
	fmt.Printf("\nis     : err = %v\n", err)
	fmt.Printf("is     : errors.Is(err, ErrEmptyKey) ? %t (traverse le ParseError)\n",
		errors.Is(err, ErrEmptyKey))

	// =========================================================================
	// errors.As : récupérer le TYPE concret derrière la chaîne
	// =========================================================================

	var pe *ParseError
	if errors.As(err, &pe) {
		fmt.Printf("as     : c'est un *ParseError, à la ligne %d\n", pe.Line)
	}

	// =========================================================================
	// errors.AsType[E] (🆕 1.26) : variante générique, typée, sans variable cible
	// =========================================================================

	if pe2, ok := errors.AsType[*ParseError](err); ok {
		fmt.Printf("astype : AsType[*ParseError] -> ligne %d (pas de &pe à passer)\n", pe2.Line)
	}

	// =========================================================================
	// errors.Join : agréger plusieurs erreurs indépendantes
	// =========================================================================

	cfg, err := parseConfig([]string{
		"host = localhost", // ok
		"port8080",         // séparateur manquant
		"  = oops",         // clé vide
		"name = go-book",   // ok
	})
	fmt.Printf("\njoin   : cfg=%v\n", cfg)
	fmt.Printf("join   : erreurs agrégées :\n%v\n", err)
	fmt.Printf("join   : errors.Is(err, ErrEmptyKey) ? %t\n", errors.Is(err, ErrEmptyKey))

	// =========================================================================
	// Cas nominal : pas d'erreur
	// =========================================================================

	cfg, err = parseConfig([]string{"a = 1", "b = 2"})
	fmt.Printf("\nok     : cfg=%v err=%v\n", cfg, err)
}
