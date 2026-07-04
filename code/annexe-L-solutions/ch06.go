package main

// ch06Chunk découpe s en tranches d'au plus size éléments. L'expression à TROIS
// indices `s[i:end:end]` fige la capacité : chaque morceau ne peut pas écrire
// dans la mémoire du morceau suivant (exercice 1 : `s[i:end]` réintroduirait
// l'aliasing, un append sur un morceau écraserait le début du suivant).
func ch06Chunk[T any](s []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	var out [][]T
	for i := 0; i < len(s); i += size {
		end := min(i+size, len(s))
		out = append(out, s[i:end:end])
	}
	return out
}

// ch06RemoveAt retire l'élément d'indice i sans fuite (exercice 3). Pour un
// slice de pointeurs, l'élément décalé laisserait une référence morte en fin de
// slice qui empêcherait le GC de libérer l'objet ; on met donc explicitement le
// dernier emplacement à zéro avant de tronquer.
func ch06RemoveAt[T any](s []T, i int) []T {
	if i < 0 || i >= len(s) {
		return s
	}
	copy(s[i:], s[i+1:])
	var zero T
	s[len(s)-1] = zero // évite la fuite : plus de référence retenue
	return s[:len(s)-1]
}
