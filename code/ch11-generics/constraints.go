package main

import "cmp"

// Number est une CONTRAINTE : tout type dont le sous-jacent est l'un de ceux-ci.
// Le ~ capte aussi les types DÉFINIS (ex. type Celsius float64), pas seulement les
// types exacts.
type Number interface {
	~int | ~int64 | ~float64
}

// Sum additionne un slice de n'importe quel Number. La contrainte garantit que
// l'opérateur + est disponible sur T.
func Sum[T Number](xs []T) T {
	var acc T // zero value de T
	for _, x := range xs {
		acc += x
	}
	return acc
}

// Max renvoie le plus grand des deux. cmp.Ordered (1.21) couvre tous les types
// ordonnables : entiers, flottants, strings.
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Map applique f à chaque élément et renvoie un nouveau slice. E et R sont libres
// (any) et indépendants : on peut transformer des string en int, par exemple.
func Map[E, R any](s []E, f func(E) R) []R {
	out := make([]R, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}

// Filter conserve les éléments pour lesquels keep renvoie true.
func Filter[T any](s []T, keep func(T) bool) []T {
	out := make([]T, 0, len(s))
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// Index renvoie la position de target, ou -1. comparable autorise == sur T.
func Index[T comparable](s []T, target T) int {
	for i, v := range s {
		if v == target {
			return i
		}
	}
	return -1
}
