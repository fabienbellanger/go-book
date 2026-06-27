package main

import (
	"strconv"
	"strings"
)

// FormatThousands insère une espace tous les trois chiffres, en partant de la
// droite : 1234567 -> "1 234 567". Le signe est conservé. strconv.Itoa gère
// correctement math.MinInt (pas de débordement à la négation).
//
// Version idiomatique O(n) : un seul strings.Builder, préalloué.
func FormatThousands(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	var b strings.Builder
	b.Grow(len(s) + len(s)/3 + 1) // place pour les chiffres + les séparateurs
	if neg {
		b.WriteByte('-')
	}
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteByte(c)
	}
	return b.String()
}

// formatNaive produit le MÊME résultat mais par concaténation de chaînes : à
// chaque tour, "sep + reste" réalloue toute la chaîne -> O(n^2) et beaucoup
// d'allocations. Sert de point de comparaison pour benchstat.
func formatNaive(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	out := ""
	for i := len(s); i > 0; i -= 3 {
		start := max(i-3, 0)
		group := s[start:i]
		if out == "" {
			out = group
		} else {
			out = group + " " + out // réalloue toute la chaîne à chaque tour
		}
	}
	if neg {
		out = "-" + out
	}
	return out
}
