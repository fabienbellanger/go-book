package main

import (
	"slices"
	"testing"
)

func TestDivmod(t *testing.T) {
	if q, r := divmod(17, 5); q != 3 || r != 2 {
		t.Errorf("divmod(17,5) = (%d,%d) ; attendu (3,2)", q, r)
	}
}

func TestSafeDivide(t *testing.T) {
	if got, err := safeDivide(10, 2); err != nil || got != 5 {
		t.Errorf("safeDivide(10,2) = (%d, %v) ; attendu (5, nil)", got, err)
	}
	if _, err := safeDivide(1, 0); err == nil {
		t.Error("safeDivide(1,0) : une erreur était attendue")
	}
}

func TestSum(t *testing.T) {
	if sum() != 0 {
		t.Error("sum() devrait valoir 0")
	}
	if sum(1, 2, 3) != 6 {
		t.Error("sum(1,2,3) devrait valoir 6")
	}
	xs := []int{4, 5, 6}
	if sum(xs...) != 15 {
		t.Error("sum(xs...) devrait valoir 15")
	}
}

func TestMinMax(t *testing.T) {
	if lo, hi := minMax(3, 9, 1, 7); lo != 1 || hi != 9 {
		t.Errorf("minMax = (%d,%d) ; attendu (1,9)", lo, hi)
	}
	if lo, hi := minMax(); lo != 0 || hi != 0 {
		t.Errorf("minMax() = (%d,%d) ; attendu (0,0)", lo, hi)
	}
}

func TestApply(t *testing.T) {
	got := apply([]int{1, 2, 3, 4}, func(n int) int { return n * n })
	if want := []int{1, 4, 9, 16}; !slices.Equal(got, want) {
		t.Errorf("apply = %v ; attendu %v", got, want)
	}
}

func TestFactorial(t *testing.T) {
	if factorial(5) != 120 {
		t.Errorf("factorial(5) = %d ; attendu 120", factorial(5))
	}
}

// TestPassByValue vérifie la sémantique : valeur copiée vs pointeur partagé.
func TestPassByValue(t *testing.T) {
	c := counter{n: 0}
	incVal(c)
	if c.n != 0 {
		t.Errorf("incVal a modifié l'original (c.n=%d) ; attendu 0", c.n)
	}
	incPtr(&c)
	if c.n != 1 {
		t.Errorf("incPtr n'a pas modifié l'original (c.n=%d) ; attendu 1", c.n)
	}
}

func TestScale(t *testing.T) {
	nums := []int{1, 2, 3}
	scale(nums, 10)
	if want := []int{10, 20, 30}; !slices.Equal(nums, want) {
		t.Errorf("scale = %v ; attendu %v", nums, want)
	}
}

func TestNewServer(t *testing.T) {
	// Défauts seuls.
	d := NewServer()
	if d.host != "localhost" || d.port != 8080 || d.tls {
		t.Errorf("défauts incorrects : %+v", *d)
	}
	// Options appliquées.
	s := NewServer(WithHost("example.com"), WithPort(9000), WithTLS())
	if s.host != "example.com" || s.port != 9000 || !s.tls {
		t.Errorf("options non appliquées : %+v", *s)
	}
}
