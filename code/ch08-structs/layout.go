package main

import "unsafe"

// Padded et Packed contiennent les MÊMES champs dans un ORDRE différent. À cause de
// l'alignement mémoire, l'ordre des champs change la taille du struct : le compilateur
// insère des octets de remplissage (padding) pour aligner chaque champ.
// (Détail au Ch. 35 — unsafe, Sizeof/Alignof/Offsetof.)

// Padded : ordre naïf -> 24 octets (1 + 7pad + 8 + 1 + 7pad).
type Padded struct {
	a bool  // offset 0
	b int64 // offset 8 (aligné sur 8)
	c bool  // offset 16
}

// Packed : champ le plus large en premier -> 16 octets (8 + 1 + 1 + 6pad).
type Packed struct {
	b int64 // offset 0
	a bool  // offset 8
	c bool  // offset 9
}

// FieldSizes renvoie la taille en octets des deux agencements, pour la comparaison.
func FieldSizes() (padded, packed uintptr) {
	return unsafe.Sizeof(Padded{}), unsafe.Sizeof(Packed{})
}
