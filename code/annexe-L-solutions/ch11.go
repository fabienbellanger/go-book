package main

import (
	"cmp"
	"slices"
)

// ch11Number contraint aux types numériques. Les `~` autorisent les types
// DÉFINIS sur ces bases (ex. `type Celsius float64`) : sans eux, passer un
// Celsius provoquerait une erreur de compilation (exercice 1).
type ch11Number interface {
	~int | ~int64 | ~float64
}

// ch11Sum additionne n'importe quel slice de Number.
func ch11Sum[T ch11Number](xs []T) T {
	var total T
	for _, x := range xs {
		total += x
	}
	return total
}

// ch11Index réécrit la recherche linéaire avec slices.Index de la stdlib
// (exercice 3) : même résultat, zéro code à maintenir.
func ch11Index[T comparable](xs []T, target T) int {
	return slices.Index(xs, target)
}

// ch11Zero renvoie la valeur zéro de T (exercice 4). Appeler `ch11Zero()` sans
// argument échoue (« cannot infer T ») car rien ne permet de déduire T ; il faut
// l'expliciter : `ch11Zero[int]()`.
func ch11Zero[T any]() T {
	var z T
	return z
}

// ch11Max s'appuie sur cmp.Ordered pour rester générique et sûr.
func ch11Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
