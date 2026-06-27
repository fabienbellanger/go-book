package main

import "fmt"

// ValidationError est un type d'erreur personnalisé : il satisfait l'interface
// standard `error` en implémentant `Error() string`. Récepteur POINTEUR (idiome
// courant pour les erreurs : on compare souvent par identité de pointeur).
type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: champ %q: %s", e.Field, e.Msg)
}

// validateAge renvoie correctement nil (interface vraiment nil) ou une *ValidationError.
func validateAge(age int) error {
	if age < 0 {
		return &ValidationError{Field: "age", Msg: "doit être positif"}
	}
	return nil
}

// typedNilError illustre LE PIÈGE : renvoyer un pointeur nil TYPÉ via l'interface
// error produit une interface NON nil (elle porte le type *ValidationError, valeur
// nil). Comparer son résultat à nil donne donc false. NE PAS écrire ce genre de code.
func typedNilError() error {
	var p *ValidationError // nil
	return p               // (type=*ValidationError, valeur=nil) -> != nil !
}
