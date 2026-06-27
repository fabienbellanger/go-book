package main

import (
	"math"
	"testing"
	"unsafe"
)

func TestTotalArea(t *testing.T) {
	shapes := []Shape{Circle{R: 1}, Rectangle{W: 2, H: 3}}
	got := TotalArea(shapes)
	want := math.Pi + 6
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("TotalArea = %g ; attendu %g", got, want)
	}
}

func TestDescribe(t *testing.T) {
	if got := Describe(Circle{R: 2}); got != "cercle r=2" {
		t.Errorf("Describe(Circle) = %q", got)
	}
	if got := Describe(Rectangle{W: 2, H: 3}); got != "rectangle 2x3" {
		t.Errorf("Describe(Rectangle) = %q", got)
	}
}

// Le cœur du piège : un pointeur nil typé rangé dans une interface n'est PAS nil.
func TestNilInterfacePitfall(t *testing.T) {
	if FailBuggy(true) == nil {
		t.Error("FailBuggy(true) devrait être != nil (c'est justement le piège)")
	}
	if FailCorrect(true) != nil {
		t.Error("FailCorrect(true) devrait être == nil")
	}
	if FailCorrect(false) == nil {
		t.Error("FailCorrect(false) devrait porter une erreur")
	}
}

// Les interfaces (vide ou non) font 2 mots = 16 octets sur une machine 64 bits.
func TestInterfaceSize(t *testing.T) {
	var e any
	var s Shape
	if sz := unsafe.Sizeof(e); sz != 16 {
		t.Errorf("Sizeof(any) = %d ; attendu 16", sz)
	}
	if sz := unsafe.Sizeof(s); sz != 16 {
		t.Errorf("Sizeof(Shape) = %d ; attendu 16", sz)
	}
}

// Boxing : 0..255 mis en cache (0 alloc) ; au-delà, 1 alloc par valeur.
func TestBoxingAllocations(t *testing.T) {
	cached := testing.AllocsPerRun(100, func() {
		for i := range 256 {
			BoxValue(i)
		}
	})
	if cached != 0 {
		t.Errorf("boxing int 0..255 = %.1f alloc ; attendu 0 (cache runtime)", cached)
	}

	vals := []int{1000, 2000, 3000, 4000, 5000}
	big := testing.AllocsPerRun(100, func() {
		for _, v := range vals {
			BoxValue(v)
		}
	})
	if big != float64(len(vals)) {
		t.Errorf("boxing %d grands entiers = %.1f alloc ; attendu %d", len(vals), big, len(vals))
	}
}

func TestReflectTypeAssert(t *testing.T) {
	if c, ok := AsCircle(Circle{R: 3}); !ok || c.R != 3 {
		t.Errorf("AsCircle(Circle) = (%v,%v) ; attendu ({3},true)", c, ok)
	}
	if _, ok := AsCircle(Rectangle{W: 1, H: 1}); ok {
		t.Error("AsCircle(Rectangle) devrait échouer (ok=false)")
	}
}
