package main

import "testing"

var payload = []byte("une charge utile représentative pour comparer copie et zéro-copie")

// Conversion SÛRE : string([]byte) copie le backing (immutabilité garantie).
func BenchmarkSafeConvert(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = string(payload)
	}
}

// Conversion unsafe : aucune copie, aucune allocation — mais le []byte ne doit plus
// jamais être modifié ensuite.
func BenchmarkUnsafeConvert(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = BytesToString(payload)
	}
}
