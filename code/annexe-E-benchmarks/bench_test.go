package main

import "testing"

// Puits : empêchent le compilateur d'éliminer le résultat mesuré.
var (
	sinkInt int
	sinkStr string
	sinkPtr *point
	sinkSli []int
	sinkMap map[int]int
)

// --- Pile vs tas (escape analysis) -----------------------------------------

func BenchmarkStack(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkInt = sumOnStack(3, 4)
	}
}

func BenchmarkHeap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkPtr = newOnHeap(3, 4)
	}
}

// --- Mutex vs atomic, sous contention --------------------------------------

func BenchmarkMutexCounter(b *testing.B) {
	var c mutexCounter
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
	sinkInt = int(c.Value())
}

func BenchmarkAtomicCounter(b *testing.B) {
	var c atomicCounter
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
	sinkInt = int(c.Value())
}

// --- Interface vs générique ------------------------------------------------

var dispatchInput = func() []int {
	xs := make([]int, 1024)
	for i := range xs {
		xs[i] = i
	}
	return xs
}()

func BenchmarkDispatchInterface(b *testing.B) {
	b.ReportAllocs()
	d := intDoubler{}
	for b.Loop() {
		sinkInt = viaInterface(d, dispatchInput)
	}
}

func BenchmarkDispatchGeneric(b *testing.B) {
	b.ReportAllocs()
	d := intDoubler{}
	for b.Loop() {
		sinkInt = viaGeneric(d, dispatchInput)
	}
}

// --- Concaténation de chaînes ----------------------------------------------

var stringParts = func() []string {
	parts := make([]string, 512)
	for i := range parts {
		parts[i] = "fragment"
	}
	return parts
}()

func BenchmarkConcatPlus(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = concatPlus(stringParts)
	}
}

func BenchmarkConcatBuilder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = concatBuilder(stringParts)
	}
}

func BenchmarkConcatBuilderGrow(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = concatBuilderGrow(stringParts)
	}
}

// --- Préallocation de slice et de map --------------------------------------

const preallocN = 10000

func BenchmarkSliceNoPrealloc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkSli = sliceNoPrealloc(preallocN)
	}
}

func BenchmarkSlicePrealloc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkSli = slicePrealloc(preallocN)
	}
}

func BenchmarkMapNoPrealloc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkMap = mapNoPrealloc(preallocN)
	}
}

func BenchmarkMapPrealloc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkMap = mapPrealloc(preallocN)
	}
}
