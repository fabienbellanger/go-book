package main

import (
	"testing"
)

// La croissance est monotone et ne dépasse jamais le doublement à chaque étape.
func TestCapGrowthInvariants(t *testing.T) {
	caps := CapGrowth(3000)
	if len(caps) < 5 {
		t.Fatalf("trop peu d'étapes de croissance : %v", caps)
	}
	for i := 1; i < len(caps); i++ {
		if caps[i] <= caps[i-1] {
			t.Errorf("cap non strictement croissante : %d puis %d", caps[i-1], caps[i])
		}
		if caps[i] > 2*caps[i-1] {
			t.Errorf("croissance > 2x : %d -> %d", caps[i-1], caps[i])
		}
	}
	if last := caps[len(caps)-1]; last < 3000 {
		t.Errorf("cap finale %d < 3000", last)
	}
}

// En dessous de 256, la capacité double ; au-delà, elle croît plus lentement (~1,25x).
func TestCapGrowthSlowsDown(t *testing.T) {
	caps := CapGrowth(3000)
	var smallRatio, largeRatio float64
	for i := 1; i < len(caps); i++ {
		ratio := float64(caps[i]) / float64(caps[i-1])
		if caps[i-1] < 256 {
			smallRatio = ratio // dernier ratio observé sous 256
		}
		if caps[i-1] >= 512 {
			largeRatio = ratio // un ratio observé au-delà de 512
		}
	}
	if smallRatio != 2 {
		t.Errorf("sous 256, ratio attendu 2 ; obtenu %.2f", smallRatio)
	}
	if largeRatio == 0 || largeRatio >= 2 {
		t.Errorf("au-delà de 512, ratio attendu < 2 (ralentissement) ; obtenu %.2f", largeRatio)
	}
}

// L'expression à 3 indices borne la capacité à max-low.
func TestSubSliceCap(t *testing.T) {
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if got := SubSliceCap(s, 2, 4, 6); got != 4 {
		t.Errorf("cap(s[2:4:6]) = %d ; attendu 4", got)
	}
	if got := cap(s[2:4]); got != 8 {
		t.Errorf("cap(s[2:4]) = %d ; attendu 8 (jusqu'au bout)", got)
	}
}

// Sans borne de cap, append sur un sous-slice écrase le parent ; avec borne, non.
func TestAliasingVsIsolation(t *testing.T) {
	parent, modified := AppendAliasing()
	if parent[2] != 99 {
		t.Errorf("aliasing attendu : parent[2] = %d ; attendu 99", parent[2])
	}
	if modified[2] != 99 {
		t.Errorf("modified[2] = %d ; attendu 99", modified[2])
	}

	safeParent, safeMod := SafeAppend()
	if safeParent[2] != 3 {
		t.Errorf("isolation attendue : parent[2] = %d ; attendu 3 (intact)", safeParent[2])
	}
	if safeMod[2] != 99 {
		t.Errorf("modified[2] = %d ; attendu 99", safeMod[2])
	}
}

// Le filtrage en place ne fait aucune allocation et conserve les bons éléments.
func TestFilterInPlace(t *testing.T) {
	src := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := FilterInPlace(src, func(v int) bool { return v%2 == 0 })
	want := []int{2, 4, 6, 8, 10}
	if len(got) != len(want) {
		t.Fatalf("len = %d ; attendu %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %d ; attendu %d", i, got[i], want[i])
		}
	}

	buf := make([]int, 100)
	for i := range buf {
		buf[i] = i
	}
	allocs := testing.AllocsPerRun(100, func() {
		_ = FilterInPlace(buf, func(v int) bool { return v%3 == 0 })
	})
	if allocs != 0 {
		t.Errorf("FilterInPlace = %.0f alloc/op ; attendu 0 (réutilise le backing)", allocs)
	}
}

// Clone réduit la capacité au strict nécessaire, libérant le grand tableau.
func TestTrimRetention(t *testing.T) {
	big := make([]int, 1000)
	small := TrimRetention(big, 3)
	if len(small) != 3 {
		t.Errorf("len = %d ; attendu 3", len(small))
	}
	if cap(small) != 3 {
		t.Errorf("cap = %d ; attendu 3 (backing serré, pas 1000)", cap(small))
	}
}
