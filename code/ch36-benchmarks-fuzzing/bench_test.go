package main

import (
	"fmt"
	"testing"
)

// sink reçoit le résultat de chaque itération : sans cette affectation à une
// variable de package, le compilateur pourrait éliminer l'appel mesuré (le
// résultat n'étant pas utilisé) et le benchmark mesurerait... rien.
var sink string

// Benchmark avec b.Loop (🆕 1.24) : la boucle cadence les itérations et fige le
// timer pendant le setup/cleanup. On compare les deux implémentations.
func BenchmarkBuilder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = FormatThousands(1234567)
	}
}

func BenchmarkNaive(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = formatNaive(1234567)
	}
}

// Sous-benchmarks : b.Run balaie plusieurs tailles d'entrée et nomme chaque
// résultat (BenchmarkSizes/digits=...), pour voir le coût croître.
func BenchmarkSizes(b *testing.B) {
	inputs := []struct {
		name string
		n    int
	}{
		{"digits=3", 999},
		{"digits=7", 1234567},
		{"digits=13", 1234567890123},
	}
	for _, in := range inputs {
		b.Run(in.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink = FormatThousands(in.n)
			}
		})
	}
}

// ExampleFormatThousands_sizes garde la variable sink « vivante » côté vet sans
// l'exposer ; sert aussi de garde-fou pédagogique.
func ExampleFormatThousands_sizes() {
	fmt.Println(len(FormatThousands(1234567890123)))
	// Output: 17
}
