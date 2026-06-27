package main

import (
	"cmp"
	"maps"
	"slices"
)

// Set est un ALIAS de type GÉNÉRIQUE (🆕 1.24). Auparavant un alias ne pouvait pas
// porter de paramètre de type. Comme c'est un alias, Set[T] EST exactement un
// map[T]struct{} : on peut donc le manipuler avec les littéraux et builtins de map.
type Set[T comparable] = map[T]struct{}

// NewSet construit un ensemble à partir d'éléments. struct{} pèse 0 octet : la map
// ne sert que de table d'appartenance.
func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	for _, it := range items {
		s[it] = struct{}{}
	}
	return s
}

// SortedKeys renvoie les éléments triés. La contrainte cmp.Ordered (plus stricte que
// comparable) est nécessaire pour les ranger. maps.Keys renvoie un itérateur (Ch. 18)
// que slices.Sorted matérialise en slice trié.
func SortedKeys[T cmp.Ordered](s Set[T]) []T {
	return slices.Sorted(maps.Keys(s))
}
