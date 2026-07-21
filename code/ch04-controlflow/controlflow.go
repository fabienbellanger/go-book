package main

import "fmt"

// classify renvoie une mention selon un score, via un switch SANS condition
// (équivalent lisible d'une cascade if / else if).
func classify(score int) string {
	switch {
	case score < 0 || score > 100:
		return "invalide"
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// fizzbuzz produit les n premiers termes du jeu FizzBuzz (pour k de 1 à n).
// Illustre `for range n` (range sur un entier, 🆕 1.22) et le switch d'expression.
func fizzbuzz(n int) []string {
	out := make([]string, 0, n)
	for i := range n { // i va de 0 à n-1
		k := i + 1
		switch {
		case k%15 == 0:
			out = append(out, "FizzBuzz")
		case k%3 == 0:
			out = append(out, "Fizz")
		case k%5 == 0:
			out = append(out, "Buzz")
		default:
			out = append(out, fmt.Sprintf("%d", k))
		}
	}
	return out
}

// firstPair cherche, dans une grille, le premier indice (i, j) dont la valeur
// vaut target. Illustre le `break` ÉTIQUETÉ pour sortir des deux boucles d'un coup.
func firstPair(grid [][]int, target int) (i, j int, found bool) {
search:
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c] == target {
				i, j, found = r, c, true
				break search
			}
		}
	}
	return i, j, found
}
