// Package main illustre l'allocation pile vs tas et l'escape analysis : où vit
// une donnée dépend de si le compilateur prouve qu'elle ne « s'échappe » pas de
// la fonction. Ce qui reste local part sur la PILE (gratuit) ; ce qui s'échappe
// part sur le TAS (alloc + travail pour le GC).
package main

// Point est un petit struct de valeur.
type Point struct{ X, Y int }

// sumLocalArray garde un tableau de taille fixe LOCAL : il ne s'échappe pas, donc
// il vit sur la pile. Aucune allocation tas. (noinline pour une mesure honnête.)
//
//go:noinline
func sumLocalArray(n int) int {
	var buf [16]int // pile : taille connue, usage local
	for i := range buf {
		buf[i] = i * n
	}
	s := 0
	for _, v := range buf {
		s += v
	}
	return s
}

// NewPoint renvoie un POINTEUR vers une variable locale : elle doit survivre à
// l'appel, donc elle « s'échappe vers le tas ». Une allocation par appel.
//
//go:noinline
func NewPoint(x, y int) *Point {
	p := Point{x, y} // moved to heap (renvoyé par pointeur)
	return &p
}

// sink simule un état qui survit à l'appel (cache, registre global, etc.) : tout
// ce qu'on y range doit donc être conservé au-delà de la fonction qui l'écrit.
var sink any

// pointToInterface boxe un Point (16 octets, deux int) dans une interface RETENUE
// par sink : même minuscule et passé PAR VALEUR (pas par pointeur, contrairement à
// NewPoint), p s'échappe vers le tas. Ce n'est pas la conversion vers `any` qui
// coûte en elle-même : c'est le fait que sink en garde une référence au-delà de
// l'appel — exactement le même principe que &p renvoyé par NewPoint, par un autre
// chemin.
//
//go:noinline
func pointToInterface(p Point) {
	sink = p // p boxé puis retenu par sink -> tas, alors que p est petit et passé par valeur
}

// sumSmallSlice construit un petit slice LOCAL et le consomme sur place. Depuis
// Go 1.25/1.26, le backing d'un slice de taille bornée et non échappé est alloué
// sur la PILE : zéro allocation tas.
//
//go:noinline
func sumSmallSlice(n int) int {
	s := make([]int, 8) // does not escape -> backing sur la pile
	for i := range s {
		s[i] = i * n
	}
	total := 0
	for _, v := range s {
		total += v
	}
	return total
}

// LeakSlice renvoie le slice : son backing s'échappe vers le tas (1 allocation).
//
//go:noinline
func LeakSlice(n int) []int {
	s := make([]int, n) // escapes to heap (renvoyé)
	for i := range s {
		s[i] = i
	}
	return s
}

// concatNoPrealloc agrandit le slice au fil de l'eau : plusieurs réallocations
// (le backing double quand cap est dépassé, cf. Ch. 6 / Ch. 30).
//
//go:noinline
func concatNoPrealloc(n int) []int {
	var out []int // cap 0 : append réalloue plusieurs fois
	for i := range n {
		out = append(out, i)
	}
	return out
}

// concatPrealloc réserve la capacité d'emblée : UNE seule allocation.
//
//go:noinline
func concatPrealloc(n int) []int {
	out := make([]int, 0, n) // cap connue : aucun réagrandissement
	for i := range n {
		out = append(out, i)
	}
	return out
}
