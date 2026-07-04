package main

import "fmt"

// ch03ToInt8Unchecked convertit sans garde : la conversion tronque sur 8 bits.
// int8(200) déborde la plage [-128, 127] et vaut -56 (200 - 256).
func ch03ToInt8Unchecked(n int) int8 {
	return int8(n) // exercice 1 : troncature silencieuse, pas d'erreur
}

// ch03ToInt8Checked garde la conversion : hors plage, elle renvoie une erreur
// plutôt qu'une valeur tronquée trompeuse.
func ch03ToInt8Checked(n int) (int8, error) {
	if n < -128 || n > 127 {
		return 0, fmt.Errorf("%d hors de la plage int8 [-128, 127]", n)
	}
	return int8(n), nil
}
