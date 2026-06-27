// Package main illustre la boucle de performance du Ch. 40 : une fonction
// naïve, un profil qui désigne le coupable, une correction principielle, puis
// une re-mesure. Les deux versions produisent un résultat IDENTIQUE.
package main

import "slices"

// DedupNaive retire les doublons en préservant l'ordre — version « évidente »
// mais quadratique : slices.Contains rescanne tout le résultat à chaque élément
// (appartenance en O(n) -> O(n^2) au total), et `out` n'est pas préalloué
// (réallocations en cascade). slices.Contains est LE point chaud révélé par le
// profil : le coupable n'est pas la fonction, c'est la STRUCTURE choisie.
func DedupNaive(items []string) []string {
	var out []string
	for _, it := range items {
		if !slices.Contains(out, it) {
			out = append(out, it)
		}
	}
	return out
}

// Dedup : même résultat, mais l'appartenance passe par une map (O(1) amorti) et
// les deux slices/map sont préalloués. Complexité O(n).
func Dedup(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		if _, ok := seen[it]; !ok {
			seen[it] = struct{}{}
			out = append(out, it)
		}
	}
	return slices.Clip(out) // rend le cap au plus juste ([Ch. 30])
}
