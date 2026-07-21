package main

import "testing"

func TestCh09TotalAreaWithTriangle(t *testing.T) {
	// Triangle s'ajoute sans que TotalArea ait été modifiée (satisfaction implicite).
	shapes := []ch09Shape{
		ch09Rectangle{W: 2, H: 3},        // 6
		ch09Triangle{Base: 4, Height: 3}, // 6
	}
	if got := ch09Round(ch09TotalArea(shapes)); got != 12 {
		t.Errorf("aire totale = %v, veut 12", got)
	}
}

type ch09fahrenheit float64

func (f ch09fahrenheit) String() string { return "temp" }

func TestCh09Describe(t *testing.T) {
	if got := ch09Describe(ch09fahrenheit(451)); got != "Stringer: temp" {
		t.Errorf("Stringer : got %q", got)
	}
	if got := ch09Describe(42); got != "int: 42" {
		t.Errorf("non-Stringer : got %q", got)
	}
}

func TestCh09MethodSet(t *testing.T) {
	// Récepteur VALEUR : la valeur ET le pointeur satisfont error.
	var _ error = ch09ValErr{msg: "x"}
	var _ error = &ch09ValErr{msg: "x"}
	// Récepteur POINTEUR : seul le pointeur satisfait error. La ligne
	//   var _ error = ch09PtrErr{msg: "x"}
	// NE compilerait PAS (ch09PtrErr{} n'a pas Error() dans son method set).
	var _ error = &ch09PtrErr{msg: "x"}

	if (ch09ValErr{msg: "boom"}).Error() != "boom" {
		t.Error("Error() à récepteur valeur")
	}
}
