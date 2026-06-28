package main

// Démonstration « pile vs tas » (escape analysis).
//
// point est un petit agrégat. Selon la façon de le renvoyer, le compilateur le
// place sur la pile (libéré gratuitement au retour) ou le fait « s'échapper »
// vers le tas (à la charge du ramasse-miettes).
//
// La commande de référence pour le constater :
//
//	go build -gcflags="-m" ./annexe-E-benchmarks/
//
// révèle « moved to heap » / « escapes to heap » sur les valeurs concernées.
type point struct{ x, y int }

// sumOnStack additionne sans rien laisser fuir : p ne survit pas à l'appel, donc
// il reste sur la pile. Aucune allocation.
func sumOnStack(a, b int) int {
	p := point{a, b} // ne s'échappe pas : usage purement local
	return p.x + p.y
}

// newOnHeap renvoie un *point : la valeur doit survivre à l'appel, elle
// s'échappe donc vers le tas. Une allocation par appel.
func newOnHeap(a, b int) *point {
	p := point{a, b}
	return &p // l'adresse fuit => escapes to heap
}
