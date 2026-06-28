package main

import "strings"

// Démonstration « concaténation de chaînes » (voir Ch. 31).
//
// Les chaînes sont immuables : « s += x » alloue une NOUVELLE chaîne à chaque
// tour. strings.Builder accumule dans un tampon d'octets réutilisé, surtout si
// l'on réserve la capacité avec Grow.

// concatPlus construit la chaîne par += : O(n²) copies, beaucoup d'allocations.
func concatPlus(parts []string) string {
	s := ""
	for _, p := range parts {
		s += p
	}
	return s
}

// concatBuilder utilise strings.Builder sans réservation préalable.
func concatBuilder(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

// concatBuilderGrow réserve d'abord la capacité totale : idéalement une seule
// allocation de tampon.
func concatBuilderGrow(parts []string) string {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	var b strings.Builder
	b.Grow(total)
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}
