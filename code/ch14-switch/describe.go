package main

import "fmt"

// describe branche sur le type DYNAMIQUE d'une interface (type switch, Ch. 9).
// Dans un cas à PLUSIEURS types (int, int64), la variable x garde le type de
// l'interface (any) : on ne peut pas y appliquer d'opération propre à un seul type.
func describe(v any) string {
	switch x := v.(type) {
	case int, int64:
		return fmt.Sprintf("entier : %v", x)
	case string:
		return fmt.Sprintf("texte de %d octets", len(x)) // x est un string ici
	case nil:
		return "nil"
	default:
		return fmt.Sprintf("autre : %T", x)
	}
}
