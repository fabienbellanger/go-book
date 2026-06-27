package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

// Test table-driven classique ([Ch. 13]).
func TestFormatThousands(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1 000"},
		{1234567, "1 234 567"},
		{-98765, "-98 765"},
	}
	for _, tc := range cases {
		if got := FormatThousands(tc.in); got != tc.want {
			t.Errorf("FormatThousands(%d) = %q ; attendu %q", tc.in, got, tc.want)
		}
	}
}

// ExampleFormatThousands : documentation exécutable ([Ch. 12]).
func ExampleFormatThousands() {
	fmt.Println(FormatThousands(1234567))
	fmt.Println(FormatThousands(-98765))
	// Output:
	// 1 234 567
	// -98 765
}

// FuzzFormatThousands : Go génère des entiers pour tenter de casser deux
// invariants. Lancé en test normal, seul le corpus de seeds (f.Add) s'exécute ;
// avec -fuzz, le moteur explore de nouvelles entrées.
func FuzzFormatThousands(f *testing.F) {
	// Corpus de seeds : on sème les cas limites connus (0, signe, MinInt/MaxInt).
	for _, seed := range []int{0, 1, -1, 999, 1000, -1000, math.MaxInt, math.MinInt} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, n int) {
		got := FormatThousands(n)
		// Invariant 1 : retirer les séparateurs redonne les chiffres d'origine.
		if stripped := strings.ReplaceAll(got, " ", ""); stripped != strconv.Itoa(n) {
			t.Errorf("FormatThousands(%d) = %q ; sans espaces %q != %q", n, got, stripped, strconv.Itoa(n))
		}
		// Invariant 2 : les deux implémentations coïncident toujours.
		if naive := formatNaive(n); naive != got {
			t.Errorf("désaccord pour %d : builder=%q naive=%q", n, got, naive)
		}
	})
}
