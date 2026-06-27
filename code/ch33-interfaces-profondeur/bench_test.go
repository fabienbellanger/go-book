package main

import "testing"

// sumViaInterface : dispatch dynamique (la méthode passe par l'itab, non inlinable).
func sumViaInterface(shapes []Shape) float64 {
	var sum float64
	for _, s := range shapes {
		sum += s.Area()
	}
	return sum
}

// sumViaConcrete : appel direct sur le type concret, inlinable par le compilateur.
func sumViaConcrete(circles []Circle) float64 {
	var sum float64
	for _, c := range circles {
		sum += c.Area()
	}
	return sum
}

func makeShapes(n int) ([]Shape, []Circle) {
	shapes := make([]Shape, n)
	circles := make([]Circle, n)
	for i := range n {
		c := Circle{R: float64(i%10 + 1)}
		shapes[i] = c
		circles[i] = c
	}
	return shapes, circles
}

// Le dispatch par interface empêche l'inlining de Area() ; l'appel concret l'autorise.
// L'écart mesure le coût réel des interfaces sur le chemin chaud.
func BenchmarkDispatchInterface(b *testing.B) {
	shapes, _ := makeShapes(1000)
	b.ReportAllocs()
	for b.Loop() {
		_ = sumViaInterface(shapes)
	}
}

func BenchmarkDispatchConcrete(b *testing.B) {
	_, circles := makeShapes(1000)
	b.ReportAllocs()
	for b.Loop() {
		_ = sumViaConcrete(circles)
	}
}
