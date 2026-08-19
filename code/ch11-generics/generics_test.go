package main

import (
	"math/rand/v2"
	"slices"
	"testing"
	"time"
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

// TestGenericMethod couvre la nouveauté Go 1.27 : une méthode peut déclarer ses
// propres paramètres de type, en plus de ceux du récepteur.
func TestGenericMethod(t *testing.T) {
	l := List[int]{1, 2, 3}

	// F inféré depuis la fonction : List[int] -> List[string].
	got := l.Apply(itoa)
	want := List[string]{"1", "2", "3"}
	if !slices.Equal(got, want) {
		t.Errorf("Apply(itoa) = %v ; attendu %v", got, want)
	}

	// Chaînage : c'est tout l'intérêt d'une méthode plutôt que d'une fonction.
	chained := l.Filter(func(i int) bool { return i%2 == 1 }).Apply(itoa)
	if !slices.Equal(chained, List[string]{"1", "3"}) {
		t.Errorf("Filter puis Apply = %v ; attendu [1 3]", chained)
	}

	// Instanciation explicite du paramètre de méthode.
	lens := got.Apply[int](func(s string) int { return len(s) })
	if !slices.Equal(lens, List[int]{1, 1, 1}) {
		t.Errorf("Apply[int](len) = %v ; attendu [1 1 1]", lens)
	}
}

// TestGenericMethodOnRand vérifie la première méthode générique de la stdlib :
// (*rand.Rand).N, jumelle de la fonction rand.N.
func TestGenericMethodOnRand(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))

	// N est générique : il s'adapte à n'importe quel type entier, y compris
	// time.Duration (dont le type sous-jacent est int64).
	if n := r.N(int32(10)); n < 0 || n >= 10 {
		t.Errorf("r.N(int32(10)) = %d ; attendu dans [0,10)", n)
	}
	if d := r.N(500 * time.Millisecond); d < 0 || d >= 500*time.Millisecond {
		t.Errorf("r.N(500ms) = %v ; attendu dans [0,500ms)", d)
	}
}
