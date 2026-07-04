package main

import (
	"slices"
	"strings"
)

// ch07ReverseString inverse une chaîne PAR RUNES. Convertir en []rune (et non
// []byte) préserve les caractères multi-octets : "café" inversé donne "éfac",
// tandis que []byte couperait l'octet médian du « é » et produirait de l'UTF-8
// invalide (exercice 2).
func ch07ReverseString(s string) string {
	r := []rune(s)
	slices.Reverse(r)
	return string(r)
}

// ch07WordFreq est un couple mot/occurrences.
type ch07WordFreq struct {
	Word string
	N    int
}

// ch07WordFrequencies compte les mots de text et les renvoie triés par
// fréquence décroissante, puis par ordre alphabétique à fréquence égale
// (exercice 3). Le tri stable secondaire rend le résultat déterministe malgré
// l'ordre d'itération aléatoire d'une map.
func ch07WordFrequencies(text string) []ch07WordFreq {
	counts := make(map[string]int)
	for _, w := range strings.Fields(text) {
		counts[strings.ToLower(w)]++
	}
	out := make([]ch07WordFreq, 0, len(counts))
	for w, n := range counts {
		out = append(out, ch07WordFreq{Word: w, N: n})
	}
	slices.SortFunc(out, func(a, b ch07WordFreq) int {
		if a.N != b.N {
			return b.N - a.N // fréquence décroissante
		}
		return strings.Compare(a.Word, b.Word) // départage stable
	})
	return out
}
