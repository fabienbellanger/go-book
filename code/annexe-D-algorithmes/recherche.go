package main

import "cmp"

// BinarySearch cherche target dans la tranche TRIÉE s. Renvoie (index, true) si
// trouvé, sinon (point d'insertion, false) — même contrat que slices.BinarySearch.
// Complexité O(log n). ⚠️ La tranche DOIT être triée au préalable.
func BinarySearch[T cmp.Ordered](s []T, target T) (int, bool) {
	lo, hi := 0, len(s)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1) // (lo+hi)/2 sans risque de débordement
		switch {
		case s[mid] < target:
			lo = mid + 1
		case s[mid] > target:
			hi = mid
		default:
			return mid, true
		}
	}
	return lo, false // lo est l'indice où insérer target pour garder l'ordre
}
