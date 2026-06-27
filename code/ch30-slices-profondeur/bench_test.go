package main

import "testing"

// filterNewSlice : variante qui ALLOUE un nouveau backing à chaque appel.
func filterNewSlice(src []int, keep func(int) bool) []int {
	var out []int // cap 0 -> réallocations successives
	for _, v := range src {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func makeInput() []int {
	buf := make([]int, 1000)
	for i := range buf {
		buf[i] = i
	}
	return buf
}

// En place : 0 allocation, on réécrit le backing existant.
func BenchmarkFilterInPlace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		src := makeInput()
		_ = FilterInPlace(src, func(v int) bool { return v%2 == 0 })
	}
}

// Nouveau slice : plusieurs réallocations le temps que cap rattrape la taille.
func BenchmarkFilterNewSlice(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		src := makeInput()
		_ = filterNewSlice(src, func(v int) bool { return v%2 == 0 })
	}
}
