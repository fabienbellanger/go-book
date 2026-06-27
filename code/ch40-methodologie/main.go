package main

import (
	"fmt"
	"slices"
	"strconv"
)

// makeItems fabrique n éléments dont seulement n/8 sont distincts : beaucoup de
// doublons, charge représentative pour la déduplication.
func makeItems(n int) []string {
	items := make([]string, n)
	for i := range items {
		items[i] = "item-" + strconv.Itoa(i%(n/8))
	}
	return items
}

func main() {
	items := makeItems(4000)

	naive := DedupNaive(items)
	fast := Dedup(items)

	fmt.Printf("entrée : %d éléments\n", len(items))
	fmt.Printf("DedupNaive -> %d distincts\n", len(naive))
	fmt.Printf("Dedup      -> %d distincts\n", len(fast))
	fmt.Printf("résultats identiques : %v\n", slices.Equal(naive, fast))
}
