package main

// reverseInts inverse un slice EN PLACE (deux indices qui se rejoignent).
// Le slice n'est pas recopié : on échange directement ses éléments.
func reverseInts(s []int) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// filter renvoie un NOUVEAU slice avec les éléments retenus par keep.
// On préalloue la capacité maximale pour éviter les réallocations d'append.
func filter(s []int, keep func(int) bool) []int {
	out := make([]int, 0, len(s))
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// chunk découpe s en tranches de taille `size` (la dernière peut être plus courte).
//
// On utilise l'expression à 3 INDICES s[i:end:end] pour borner la capacité de chaque
// tranche à sa longueur : ainsi un append ultérieur sur une tranche réalloue au lieu
// d'écraser les éléments de la tranche suivante (piège d'aliasing).
func chunk(s []int, size int) [][]int {
	if size <= 0 {
		return nil
	}
	var out [][]int
	for i := 0; i < len(s); i += size {
		end := min(i+size, len(s))
		out = append(out, s[i:end:end])
	}
	return out
}
