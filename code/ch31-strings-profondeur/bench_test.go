package main

import (
	"strings"
	"testing"
)

// concatPlus : concaténation naïve par += . Chaque + crée une NOUVELLE string
// (immuabilité) et recopie tout l'accumulateur -> O(n^2) octets copiés.
func concatPlus(parts []string) string {
	out := ""
	for _, p := range parts {
		out += p
	}
	return out
}

// concatBuilder : strings.Builder amortit la croissance comme un slice -> O(n).
func concatBuilder(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

func makeParts() []string {
	parts := make([]string, 500)
	for i := range parts {
		parts[i] = "fragment"
	}
	return parts
}

func BenchmarkConcatPlus(b *testing.B) {
	parts := makeParts()
	b.ReportAllocs()
	for b.Loop() {
		_ = concatPlus(parts)
	}
}

func BenchmarkConcatBuilder(b *testing.B) {
	parts := makeParts()
	b.ReportAllocs()
	for b.Loop() {
		_ = concatBuilder(parts)
	}
}
