package main

import "testing"

// TestHumanSize vérifie le formatage lisible des tailles (constantes iota + conversions).
func TestHumanSize(t *testing.T) {
	cases := []struct {
		in   ByteSize
		want string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{5 * MB, "5.0 MB"},
		{3 * GB, "3.0 GB"},
	}
	for _, c := range cases {
		if got := humanSize(c.in); got != c.want {
			t.Errorf("humanSize(%d) = %q ; attendu %q", int64(c.in), got, c.want)
		}
	}
}

// TestToInt8 vérifie la conversion sûre et la détection de débordement.
func TestToInt8(t *testing.T) {
	cases := []struct {
		in   int
		want int8
		ok   bool
	}{
		{0, 0, true},
		{127, 127, true},
		{-128, -128, true},
		{128, 0, false},  // déborde par le haut
		{-129, 0, false}, // déborde par le bas
		{200, 0, false},
	}
	for _, c := range cases {
		got, ok := toInt8(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("toInt8(%d) = (%d, %t) ; attendu (%d, %t)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestByteSizeValues vérifie que les constantes iota valent les bons multiples binaires.
func TestByteSizeValues(t *testing.T) {
	if KB != 1024 || MB != 1024*KB || GB != 1024*MB || TB != 1024*GB {
		t.Errorf("constantes iota incorrectes : KB=%d MB=%d GB=%d TB=%d", KB, MB, GB, TB)
	}
}
