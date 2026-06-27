package main

import "testing"

var (
	benchShapes = makeShapes(1000)
	sinkF       float64
	sinkI       int
)

// BenchmarkTotalArea sert de cible PGO : reconstruire avec -pgo=auto (après
// `go run . profile`) dévirtualise l'appel s.Area() vers Square (type dominant).
// Comparez : go test -bench=TotalArea -pgo=off  vs  -pgo=auto.
func BenchmarkTotalArea(b *testing.B) {
	for b.Loop() {
		sinkF = TotalArea(benchShapes)
	}
}

// Contraste BCE : range (sans contrôle) vs gather (contrôle conservé).
func BenchmarkSumRange(b *testing.B) {
	xs := benchInts(1024)
	for b.Loop() {
		sinkI = SumRange(xs)
	}
}

func BenchmarkSumGather(b *testing.B) {
	xs := benchInts(1024)
	idx := benchInts(1024)
	for b.Loop() {
		sinkI = SumGather(xs, idx)
	}
}

func benchInts(n int) []int {
	xs := make([]int, n)
	for i := range xs {
		xs[i] = i % n
	}
	return xs
}
