package main

import (
	"errors"
	"math"
	"testing"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestTotalArea(t *testing.T) {
	shapes := []Shape{Rect{W: 2, H: 3}, Rect{W: 1, H: 1}} // 6 + 1
	if got := totalArea(shapes); !almostEqual(got, 7) {
		t.Errorf("totalArea = %v ; attendu 7", got)
	}
	if got := totalArea(nil); got != 0 {
		t.Errorf("totalArea(nil) = %v ; attendu 0", got)
	}
}

func TestBiggest(t *testing.T) {
	shapes := []Shape{Rect{W: 1, H: 1}, Circle{Radius: 10}, Rect{W: 2, H: 2}}
	b, ok := biggest(shapes)
	if !ok {
		t.Fatal("biggest: ok=false sur une liste non vide")
	}
	if _, isCircle := b.(Circle); !isCircle {
		t.Errorf("biggest = %v ; attendu le Circle(r=10)", b)
	}
	if _, ok := biggest(nil); ok {
		t.Error("biggest(nil) devrait renvoyer ok=false")
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{42, "int:42"},
		{"hi", `string:"hi"`},
		{Rect{W: 2, H: 3}, "shape:6.00"}, // matche le cas Shape (interface)
		{nil, "nil"},
		{3.14, "autre:float64"}, // aucun cas concret -> default
	}
	for _, c := range cases {
		if got := classify(c.in); got != c.want {
			t.Errorf("classify(%v) = %q ; attendu %q", c.in, got, c.want)
		}
	}
}

func TestValidateAge(t *testing.T) {
	if err := validateAge(20); err != nil {
		t.Errorf("validateAge(20) = %v ; attendu nil", err)
	}
	err := validateAge(-1)
	if err == nil {
		t.Fatal("validateAge(-1) devrait renvoyer une erreur")
	}
	// errors.As : récupérer le type concret derrière l'interface error.
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("l'erreur devrait être une *ValidationError, obtenu %T", err)
	}
	if ve.Field != "age" {
		t.Errorf("champ = %q ; attendu \"age\"", ve.Field)
	}
}

// TestTypedNilTrap verrouille le comportement surprenant : un pointeur nil typé
// rangé dans une interface n'est PAS égal à nil (contrairement à une interface
// jamais affectée, qui, elle, vaut nil).
func TestTypedNilTrap(t *testing.T) {
	if typedNilError() == nil {
		t.Error("piège attendu : un *ValidationError nil dans une interface error n'est pas nil")
	}
}
