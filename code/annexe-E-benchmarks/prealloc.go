package main

// Démonstration « préallocation » des slices et des maps.
//
// append fait croître la capacité par doublements successifs : chaque
// redimensionnement réalloue et recopie. Connaître la taille finale et la
// réserver supprime ces copies (voir Ch. 30 pour les slices, Ch. 32 pour les maps).

// sliceNoPrealloc part d'un slice nil : la capacité grandit par paliers.
func sliceNoPrealloc(n int) []int {
	var xs []int
	for i := range n {
		xs = append(xs, i)
	}
	return xs
}

// slicePrealloc réserve la capacité une fois pour toutes.
func slicePrealloc(n int) []int {
	xs := make([]int, 0, n)
	for i := range n {
		xs = append(xs, i)
	}
	return xs
}

// mapNoPrealloc laisse la map se redimensionner au fil des insertions.
func mapNoPrealloc(n int) map[int]int {
	m := make(map[int]int)
	for i := range n {
		m[i] = i
	}
	return m
}

// mapPrealloc indique la taille attendue : moins de redimensionnements internes.
func mapPrealloc(n int) map[int]int {
	m := make(map[int]int, n)
	for i := range n {
		m[i] = i
	}
	return m
}
