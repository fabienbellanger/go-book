package main

import (
	"slices"
	"testing"
)

// TestClassify couvre les bornes et chaque tranche du switch.
func TestClassify(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{-1, "invalide"},
		{101, "invalide"},
		{95, "A"},
		{85, "B"},
		{72, "C"},
		{60, "D"},
		{42, "F"},
	}
	for _, c := range cases {
		if got := classify(c.in); got != c.want {
			t.Errorf("classify(%d) = %q ; attendu %q", c.in, got, c.want)
		}
	}
}

// TestFizzBuzz vérifie les 15 premiers termes (slices.Equal, 🆕 1.21).
func TestFizzBuzz(t *testing.T) {
	want := []string{
		"1", "2", "Fizz", "4", "Buzz", "Fizz", "7", "8", "Fizz", "Buzz",
		"11", "Fizz", "13", "14", "FizzBuzz",
	}
	if got := fizzbuzz(15); !slices.Equal(got, want) {
		t.Errorf("fizzbuzz(15) =\n  %v\nattendu\n  %v", got, want)
	}
}

// TestFirstPair vérifie le break étiqueté (premier trouvé, et cas absent).
func TestFirstPair(t *testing.T) {
	grid := [][]int{{1, 2, 3}, {4, 5, 6}}

	if i, j, ok := firstPair(grid, 5); !ok || i != 1 || j != 1 {
		t.Errorf("firstPair(.., 5) = (%d, %d, %t) ; attendu (1, 1, true)", i, j, ok)
	}
	if _, _, ok := firstPair(grid, 99); ok {
		t.Errorf("firstPair(.., 99) : attendu found=false")
	}
}
