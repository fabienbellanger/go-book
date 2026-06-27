package main

import (
	"runtime"
	"testing"
)

func sequentialSum(nums []int) int {
	s := 0
	for _, v := range nums {
		s += v
	}
	return s
}

// Le parallélisme ne change PAS le résultat : c'est l'invariant fondamental.
// On vérifie pour plusieurs nombres de workers et plusieurs GOMAXPROCS.
func TestParallelSumCorrect(t *testing.T) {
	nums := make([]int, 10_000)
	for i := range nums {
		nums[i] = i + 1
	}
	want := sequentialSum(nums)

	for _, workers := range []int{1, 2, 3, 7, 16, 1000} {
		for _, gmp := range []int{1, 2, 4} {
			var got int
			WithGOMAXPROCS(gmp, func() { got = parallelSum(nums, workers) })
			if got != want {
				t.Errorf("parallelSum(workers=%d, GOMAXPROCS=%d) = %d ; attendu %d",
					workers, gmp, got, want)
			}
		}
	}
}

// Cas limites : slice vide, un seul élément, plus de workers que d'éléments.
func TestParallelSumEdges(t *testing.T) {
	if got := parallelSum(nil, 4); got != 0 {
		t.Errorf("somme du nil = %d ; attendu 0", got)
	}
	if got := parallelSum([]int{42}, 8); got != 42 {
		t.Errorf("somme d'un seul élément = %d ; attendu 42", got)
	}
}

// WithGOMAXPROCS restaure bien la valeur précédente.
func TestWithGOMAXPROCSRestores(t *testing.T) {
	before := runtime.GOMAXPROCS(0)
	WithGOMAXPROCS(1, func() {
		if got := runtime.GOMAXPROCS(0); got != 1 {
			t.Errorf("dans le bloc, GOMAXPROCS = %d ; attendu 1", got)
		}
	})
	if after := runtime.GOMAXPROCS(0); after != before {
		t.Errorf("GOMAXPROCS non restauré : %d ; attendu %d", after, before)
	}
}
