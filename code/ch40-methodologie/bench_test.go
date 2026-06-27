package main

import "testing"

var sink []string

// Mesure les deux versions sur la MÊME entrée représentative. C'est le « avant »
// et l'« après » de la boucle d'optimisation : on les compare avec benchstat.
func BenchmarkDedupNaive(b *testing.B) {
	items := makeItems(4000)
	b.ReportAllocs()
	for b.Loop() {
		sink = DedupNaive(items)
	}
}

func BenchmarkDedup(b *testing.B) {
	items := makeItems(4000)
	b.ReportAllocs()
	for b.Loop() {
		sink = Dedup(items)
	}
}
