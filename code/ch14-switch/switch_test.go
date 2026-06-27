package main

import (
	"slices"
	"testing"
)

func TestGrade(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{95, "A"}, {90, "A"}, {89, "B"}, {72, "C"}, {70, "C"}, {40, "F"},
	}
	for _, tc := range cases {
		if got := grade(tc.in); got != tc.want {
			t.Errorf("grade(%d) = %q ; attendu %q", tc.in, got, tc.want)
		}
	}
}

func TestDayKind(t *testing.T) {
	cases := []struct{ in, want string }{
		{"samedi", "week-end"},
		{"dimanche", "week-end"},
		{"vendredi", "presque le week-end"},
		{"mardi", "semaine"},
	}
	for _, tc := range cases {
		if got := dayKind(tc.in); got != tc.want {
			t.Errorf("dayKind(%q) = %q ; attendu %q", tc.in, got, tc.want)
		}
	}
}

// TestCapabilities vérifie l'effet du fallthrough : chaque rôle hérite des droits
// inférieurs.
func TestCapabilities(t *testing.T) {
	cases := []struct {
		role string
		want []string
	}{
		{"admin", []string{"delete", "write", "read"}},
		{"editor", []string{"write", "read"}},
		{"viewer", []string{"read"}},
		{"ghost", nil}, // aucun cas ne correspond, aucun default
	}
	for _, tc := range cases {
		if got := capabilities(tc.role); !slices.Equal(got, tc.want) {
			t.Errorf("capabilities(%q) = %v ; attendu %v", tc.role, got, tc.want)
		}
	}
}

func TestDescribe(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{42, "entier : 42"},
		{int64(7), "entier : 7"},
		{"go", "texte de 2 octets"},
		{nil, "nil"},
		{3.14, "autre : float64"},
	}
	for _, tc := range cases {
		if got := describe(tc.in); got != tc.want {
			t.Errorf("describe(%v) = %q ; attendu %q", tc.in, got, tc.want)
		}
	}
}

// TestLevelsAgree : switch et map doivent renvoyer le même niveau.
func TestLevelsAgree(t *testing.T) {
	for _, s := range []string{"trace", "debug", "info", "warn", "error", "fatal", "absent"} {
		if sw, mp := levelFromString(s), levelFromMap(s); sw != mp {
			t.Errorf("désaccord sur %q : switch=%d map=%d", s, sw, mp)
		}
	}
	if got := levelFromInt(5); got != 503 {
		t.Errorf("levelFromInt(5) = %d ; attendu 503", got)
	}
	if got := levelFromInt(7); got != 700 {
		t.Errorf("levelFromInt(7) = %d ; attendu 700", got)
	}
	if got := levelFromInt(9); got != -1 {
		t.Errorf("levelFromInt(9) = %d ; attendu -1 (default)", got)
	}
}

// Benchmarks switch vs map (cf. ⚡ Performance du chapitre).
var (
	inputs = []string{"trace", "warn", "fatal", "info", "absent", "debug", "error"}
	sink   int
)

func BenchmarkSwitch(b *testing.B) {
	for b.Loop() {
		for _, s := range inputs {
			sink += levelFromString(s)
		}
	}
}

func BenchmarkMap(b *testing.B) {
	for b.Loop() {
		for _, s := range inputs {
			sink += levelFromMap(s)
		}
	}
}
