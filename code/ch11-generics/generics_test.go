package main

import (
	"slices"
	"testing"
)

func TestMax(t *testing.T) {
	if got := Max(3, 7); got != 7 {
		t.Errorf("Max(3, 7) = %d ; attendu 7", got)
	}
	if got := Max("go", "rust"); got != "rust" {
		t.Errorf("Max(go, rust) = %q ; attendu \"rust\"", got)
	}
}

func TestSum(t *testing.T) {
	if got := Sum([]int{1, 2, 3, 4}); got != 10 {
		t.Errorf("Sum = %d ; attendu 10", got)
	}
	// Type défini sur float64 : capté par ~float64.
	if got := Sum([]Celsius{19.5, 0.5}); got != 20 {
		t.Errorf("Sum(Celsius) = %g ; attendu 20", got)
	}
}

func TestMapFilter(t *testing.T) {
	got := Map([]string{"a", "bb", "ccc"}, func(s string) int { return len(s) })
	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("Map = %v ; attendu [1 2 3]", got)
	}
	odd := Filter([]int{1, 2, 3, 4, 5}, func(n int) bool { return n%2 == 1 })
	if !slices.Equal(odd, []int{1, 3, 5}) {
		t.Errorf("Filter = %v ; attendu [1 3 5]", odd)
	}
}

func TestIndex(t *testing.T) {
	s := []string{"x", "y", "z"}
	if got := Index(s, "y"); got != 1 {
		t.Errorf("Index(y) = %d ; attendu 1", got)
	}
	if got := Index(s, "absent"); got != -1 {
		t.Errorf("Index(absent) = %d ; attendu -1", got)
	}
}

func TestStack(t *testing.T) {
	var s Stack[int]
	if _, ok := s.Pop(); ok {
		t.Error("Pop sur pile vide devrait renvoyer ok=false")
	}
	s.Push(1)
	s.Push(2)
	if s.Len() != 2 {
		t.Errorf("Len = %d ; attendu 2", s.Len())
	}
	if v, ok := s.Pop(); !ok || v != 2 {
		t.Errorf("Pop = (%d, %t) ; attendu (2, true)", v, ok)
	}
}

func TestSet(t *testing.T) {
	s := NewSet("go", "rust", "go") // doublon absorbé
	if len(s) != 2 {
		t.Errorf("len(set) = %d ; attendu 2", len(s))
	}
	if _, ok := s["go"]; !ok {
		t.Error("\"go\" devrait être présent")
	}
	if got := SortedKeys(s); !slices.Equal(got, []string{"go", "rust"}) {
		t.Errorf("SortedKeys = %v ; attendu [go rust]", got)
	}
}

func TestSumAll(t *testing.T) {
	// Contrainte auto-référentielle (1.26) : addition « sur soi-même ».
	got := SumAll([]Vec2{{1, 1}, {2, 3}, {0, 1}})
	want := Vec2{X: 3, Y: 5}
	if got != want {
		t.Errorf("SumAll = %+v ; attendu %+v", got, want)
	}
}

// Benchmarks comparatifs : générique (résolu à la compilation) vs interface
// (dispatch par élément). Lancement : go test -bench=. -benchmem ./ch11-generics/...
var (
	ints  = makeInts(1000)
	boxed = makeBoxed(1000)
	sink  int
)

func makeInts(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func makeBoxed(n int) []Valuer {
	s := make([]Valuer, n)
	for i := range s {
		s[i] = myInt(i)
	}
	return s
}

func BenchmarkGeneric(b *testing.B) {
	for b.Loop() { // 🆕 1.24 : idiome de benchmark
		sink = Sum(ints)
	}
}

func BenchmarkIface(b *testing.B) {
	for b.Loop() {
		sink = sumIface(boxed)
	}
}
