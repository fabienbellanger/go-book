package main

import "testing"

func TestCh09TotalAreaWithTriangle(t *testing.T) {
	// Triangle s'ajoute sans que TotalArea ait été modifiée (satisfaction implicite).
	shapes := []ch09Shape{
		ch09Rectangle{W: 2, H: 3}, // 6
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
