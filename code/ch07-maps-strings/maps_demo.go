package main

import (
	"maps"
	"slices"
	"strings"
)

// wordCount compte les occurrences de chaque mot (insensible à la casse).
//
// Idiome clé : `counts[w]++` fonctionne même si `w` est absent, car la lecture
// d'une clé absente renvoie la zero value (0 pour un int) — pas besoin d'initialiser.
func wordCount(text string) map[string]int {
	counts := map[string]int{}
	for _, w := range strings.Fields(strings.ToLower(text)) {
		counts[w]++
	}
	return counts
}

// uniqueSorted renvoie les mots distincts, triés.
//
// On utilise un `set` = map[string]struct{} : struct{} ne pèse aucun octet, donc
// la map ne stocke que les clés. `slices.Sorted(maps.Keys(...))` transforme
// l'itération NON ORDONNÉE de la map en une tranche triée et déterministe.
func uniqueSorted(text string) []string {
	set := map[string]struct{}{}
	for _, w := range strings.Fields(strings.ToLower(text)) {
		set[w] = struct{}{}
	}
	return slices.Sorted(maps.Keys(set))
}
