package main

import (
	"slices"
	"testing"
)

// AllocsPerRun vérifie le NOMBRE d'allocations tas par appel : c'est un test, pas
// un benchmark — il échoue si une optimisation régresse (allocation inattendue).

func TestSumLocalArrayNoAlloc(t *testing.T) {
	if got := testing.AllocsPerRun(100, func() { _ = sumLocalArray(3) }); got != 0 {
		t.Errorf("sumLocalArray = %.0f alloc/op ; attendu 0 (pile)", got)
	}
}

func TestSumSmallSliceNoAlloc(t *testing.T) {
	// Backing de slice non échappé sur la pile (Go 1.25/1.26).
	if got := testing.AllocsPerRun(100, func() { _ = sumSmallSlice(3) }); got != 0 {
		t.Errorf("sumSmallSlice = %.0f alloc/op ; attendu 0 (backing sur pile)", got)
	}
}

func TestNewPointEscapes(t *testing.T) {
	if got := testing.AllocsPerRun(100, func() { _ = NewPoint(1, 2) }); got != 1 {
		t.Errorf("NewPoint = %.0f alloc/op ; attendu 1 (échappe au tas)", got)
	}
}

// La préallocation transforme N réallocations en UNE seule.
func TestPreallocReducesAllocs(t *testing.T) {
	no := testing.AllocsPerRun(10, func() { _ = concatNoPrealloc(1000) })
	pre := testing.AllocsPerRun(10, func() { _ = concatPrealloc(1000) })
	if pre != 1 {
		t.Errorf("concatPrealloc = %.0f alloc/op ; attendu 1", pre)
	}
	if no <= pre {
		t.Errorf("sans préalloc (%.0f) devrait allouer plus qu'avec (%.0f)", no, pre)
	}
}

// Les deux variantes produisent le MÊME résultat : l'optimisation ne change rien
// au comportement, seulement au coût.
func TestConcatEquivalent(t *testing.T) {
	if !slices.Equal(concatNoPrealloc(50), concatPrealloc(50)) {
		t.Error("concatNoPrealloc et concatPrealloc divergent")
	}
}
