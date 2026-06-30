package main

import (
	"slices"
	"testing"
)

// range-over-func : Count produit 0..n-1.
func TestCount(t *testing.T) {
	got := slices.Collect(Count(5))
	if want := []int{0, 1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("Count(5) = %v ; attendu %v", got, want)
	}
}

// Composition paresseuse sur une source infinie + Take.
func TestLazyPipeline(t *testing.T) {
	got := slices.Collect(Take(Map(Filter(Naturals(), even), square), 3))
	if want := []int{0, 4, 16}; !slices.Equal(got, want) {
		t.Errorf("pipeline = %v ; attendu %v", got, want)
	}
}

// Take borne une séquence infinie : sans cela, le test ne terminerait pas.
func TestTakeBoundsInfinite(t *testing.T) {
	got := slices.Collect(Take(Naturals(), 4))
	if want := []int{0, 1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("Take(Naturals, 4) = %v ; attendu %v", got, want)
	}
}

// Arrêt anticipé : un break ne consomme que le nécessaire.
func TestEarlyBreakStopsSource(t *testing.T) {
	consumed := 0
	for n := range Naturals() {
		consumed++
		if n == 3 {
			break
		}
	}
	if consumed != 4 { // 0,1,2,3 puis break
		t.Errorf("éléments consommés = %d ; attendu 4", consumed)
	}
}

// Enumerate : Seq2 (index, valeur).
func TestEnumerate(t *testing.T) {
	var idx []int
	var val []string
	for i, w := range Enumerate(slices.Values([]string{"a", "b", "c"})) {
		idx = append(idx, i)
		val = append(val, w)
	}
	if !slices.Equal(idx, []int{0, 1, 2}) || !slices.Equal(val, []string{"a", "b", "c"}) {
		t.Errorf("Enumerate = %v / %v", idx, val)
	}
}

// Piège documenté au Ch. 18 : un itérateur qui rappelle yield après qu'il a
// renvoyé false fait paniquer le runtime, pour l'empêcher de continuer à
// pousser des valeurs alors que le consommateur a déjà dit stop.
func TestYieldAfterStopPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("attendu une panique : yield rappelé après un arrêt")
		}
	}()
	for v := range BrokenAfterStop() {
		if v == 1 {
			break // BrokenAfterStop rappelle quand même yield(2) -> panique
		}
	}
}

// Zip via iter.Pull : paires jusqu'à la source la plus courte.
func TestZip(t *testing.T) {
	var keys []string
	var vals []int
	letters := slices.Values([]string{"x", "y", "z"})
	nums := slices.Values([]int{1, 2}) // plus courte : limite à 2 paires
	for k, v := range Zip(letters, nums) {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	if !slices.Equal(keys, []string{"x", "y"}) || !slices.Equal(vals, []int{1, 2}) {
		t.Errorf("Zip = %v / %v ; attendu [x y] / [1 2]", keys, vals)
	}
}
