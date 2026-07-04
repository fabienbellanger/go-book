package main

import (
	"slices"
	"testing"
)

func TestCh06ChunkNoAliasing(t *testing.T) {
	s := []int{1, 2, 3, 4, 5}
	chunks := ch06Chunk(s, 2) // [[1 2] [3 4] [5]]
	// Un append sur le premier morceau NE DOIT PAS écraser le second, grâce à
	// la capacité figée par s[i:end:end].
	chunks[0] = append(chunks[0], 99)
	if !slices.Equal(chunks[1], []int{3, 4}) {
		t.Errorf("aliasing : le 2e morceau a été corrompu -> %v", chunks[1])
	}
}

func TestCh06RemoveAt(t *testing.T) {
	got := ch06RemoveAt([]int{10, 20, 30, 40}, 1)
	if !slices.Equal(got, []int{10, 30, 40}) {
		t.Errorf("removeAt = %v", got)
	}
	// Retrait du dernier élément.
	got = ch06RemoveAt([]int{1, 2}, 1)
	if !slices.Equal(got, []int{1}) {
		t.Errorf("removeAt dernier = %v", got)
	}
}
