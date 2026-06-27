package main

import "testing"

// materialize construit une slice de 0..n-1. //go:noinline force la slice à
// échapper sur le tas (cas réaliste : construite ici, consommée ailleurs).
//
//go:noinline
func materialize(n int) []int {
	s := make([]int, 0, n)
	for i := range n {
		s = append(s, i)
	}
	return s
}

var sink int

// Sommer via l'itérateur : aucune slice intermédiaire.
func BenchmarkIterator(b *testing.B) {
	for b.Loop() {
		total := 0
		for v := range Count(1000) {
			total += v
		}
		sink = total
	}
}

// Sommer via une slice matérialisée : alloue 1000 entiers (8 Ko) sur le tas.
func BenchmarkMaterialized(b *testing.B) {
	for b.Loop() {
		total := 0
		for _, v := range materialize(1000) {
			total += v
		}
		sink = total
	}
}
