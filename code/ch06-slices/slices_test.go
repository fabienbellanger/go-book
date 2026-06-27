package main

import (
	"slices"
	"testing"
)

func TestReverseInts(t *testing.T) {
	s := []int{1, 2, 3, 4}
	reverseInts(s)
	if want := []int{4, 3, 2, 1}; !slices.Equal(s, want) {
		t.Errorf("reverseInts = %v ; attendu %v", s, want)
	}
	// Cas pair/impair et vide.
	odd := []int{1, 2, 3}
	reverseInts(odd)
	if want := []int{3, 2, 1}; !slices.Equal(odd, want) {
		t.Errorf("reverseInts(impair) = %v ; attendu %v", odd, want)
	}
	reverseInts([]int{}) // ne doit pas paniquer
}

func TestFilter(t *testing.T) {
	got := filter([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 })
	if want := []int{2, 4, 6}; !slices.Equal(got, want) {
		t.Errorf("filter(pairs) = %v ; attendu %v", got, want)
	}
}

func TestChunk(t *testing.T) {
	got := chunk([]int{1, 2, 3, 4, 5}, 2)
	want := [][]int{{1, 2}, {3, 4}, {5}}
	if len(got) != len(want) {
		t.Fatalf("chunk : %d tranches ; attendu %d", len(got), len(want))
	}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Errorf("chunk[%d] = %v ; attendu %v", i, got[i], want[i])
		}
	}
	if chunk([]int{1, 2}, 0) != nil {
		t.Error("chunk avec size<=0 doit renvoyer nil")
	}
}

// TestChunkNoAliasing vérifie que le 3-index borne bien la capacité : un append
// sur une tranche ne doit PAS écraser la tranche suivante.
func TestChunkNoAliasing(t *testing.T) {
	s := []int{1, 2, 3, 4}
	parts := chunk(s, 2) // {1,2} {3,4}
	parts[0] = append(parts[0], 99)
	if s[2] != 3 {
		t.Errorf("aliasing : append sur parts[0] a corrompu s (s[2]=%d ; attendu 3)", s[2])
	}
}
