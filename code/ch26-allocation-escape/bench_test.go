package main

import "testing"

// Comparer pile vs tas : lancer avec -benchmem pour voir B/op et allocs/op.
//   go test -bench=. -benchmem ./ch26-allocation-escape/...

func BenchmarkSumLocalArray(b *testing.B) {
	for b.Loop() {
		_ = sumLocalArray(3) // 0 alloc/op : tout sur la pile
	}
}

func BenchmarkNewPoint(b *testing.B) {
	for b.Loop() {
		_ = NewPoint(1, 2) // 1 alloc/op : pointeur échappé
	}
}

func BenchmarkConcatNoPrealloc(b *testing.B) {
	for b.Loop() {
		_ = concatNoPrealloc(1000) // plusieurs réallocations
	}
}

func BenchmarkConcatPrealloc(b *testing.B) {
	for b.Loop() {
		_ = concatPrealloc(1000) // 1 allocation
	}
}
