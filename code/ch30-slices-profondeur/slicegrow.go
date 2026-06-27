package main

import "slices"

// CapGrowth ajoute n entiers à un slice initialement nil et renvoie la suite des
// capacités DISTINCTES observées. C'est la stratégie de croissance d'append rendue
// visible : double tant que cap < 256, puis ~1,25x, le tout arrondi à une size class.
func CapGrowth(n int) []int {
	var s []int
	caps := make([]int, 0, 16)
	prev := -1
	for i := range n {
		s = append(s, i)
		if cap(s) != prev {
			caps = append(caps, cap(s))
			prev = cap(s)
		}
	}
	return caps
}

// SubSliceCap renvoie la capacité de s[low:high:max] (expression a 3 indices, dite
// "full slice expression"). La capacite resultante vaut max-low : elle BORNE ce que
// le sous-slice peut atteindre sans reallouer, donc limite l'aliasing.
func SubSliceCap(s []int, low, high, max int) int {
	return cap(s[low:high:max])
}

// AppendAliasing illustre le piege : append sur un sous-slice qui possede encore de
// la capacite ECRASE le backing array partage avec le slice parent.
func AppendAliasing() (parent, modified []int) {
	parent = []int{1, 2, 3, 4}
	head := parent[:2]          // len=2, cap=4 : partage le backing de parent
	modified = append(head, 99) // cap suffit -> ecrit dans parent[2] !
	return parent, modified
}

// SafeAppend borne la capacite (3e indice) pour qu'un futur append REALLOUE au lieu
// d'ecraser le parent : le sous-slice devient independant.
func SafeAppend() (parent, modified []int) {
	parent = []int{1, 2, 3, 4}
	head := parent[:2:2]        // cap bornee a 2 : plus de place disponible
	modified = append(head, 99) // realloue un nouveau backing -> parent intact
	return parent, modified
}

// FilterInPlace filtre un slice SANS allocation en reutilisant son backing array :
// out part de src[:0] et reecrit par-dessus les elements conserves.
func FilterInPlace(src []int, keep func(int) bool) []int {
	out := src[:0] // meme pointeur de backing, len remise a 0
	for _, v := range src {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// TrimRetention corrige une retention memoire : garder un petit sous-slice d'un grand
// tableau empeche le GC de liberer ce grand tableau. Clone copie vers un backing juste
// assez grand, liberant l'original.
func TrimRetention(big []int, n int) []int {
	return slices.Clone(big[:n]) // nouveau backing de taille n, l'ancien devient collectable
}
