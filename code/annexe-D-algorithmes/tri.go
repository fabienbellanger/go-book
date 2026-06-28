package main

import "cmp"

// QuickSort trie s EN PLACE par tri rapide (pivot = dernier élément, partition de
// Lomuto). Complexité moyenne O(n log n), pire cas O(n²) sur entrée déjà triée.
// En pratique : utiliser slices.Sort (introsort optimisé) — 🔁 Ch. 30.
func QuickSort[T cmp.Ordered](s []T) {
	if len(s) < 2 {
		return
	}
	pivot := s[len(s)-1]
	i := 0 // frontière des éléments < pivot
	for j := 0; j < len(s)-1; j++ {
		if s[j] < pivot {
			s[i], s[j] = s[j], s[i]
			i++
		}
	}
	s[i], s[len(s)-1] = s[len(s)-1], s[i] // place le pivot à sa position finale
	QuickSort(s[:i])
	QuickSort(s[i+1:])
}

// MergeSort renvoie une NOUVELLE tranche triée (tri fusion), sans modifier
// l'entrée. Complexité O(n log n) GARANTIE et stable, au prix de O(n) mémoire.
func MergeSort[T cmp.Ordered](s []T) []T {
	if len(s) < 2 {
		return append([]T(nil), s...)
	}
	mid := len(s) / 2
	left := MergeSort(s[:mid])
	right := MergeSort(s[mid:])
	return merge(left, right)
}

// merge fusionne deux tranches déjà triées en une seule tranche triée.
func merge[T cmp.Ordered](a, b []T) []T {
	out := make([]T, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	out = append(out, a[i:]...)
	out = append(out, b[j:]...)
	return out
}
