package main

import "testing"

const benchN = 1000

// Sans préallocation : la table croît et ré-évacue plusieurs fois.
func BenchmarkMapBuildNoPrealloc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		m := make(map[int]int)
		for i := range benchN {
			m[i] = i
		}
	}
}

// Avec make(map, n) : la table est dimensionnée d'emblée, bien moins d'allocations.
func BenchmarkMapBuildPrealloc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		m := make(map[int]int, benchN)
		for i := range benchN {
			m[i] = i
		}
	}
}
