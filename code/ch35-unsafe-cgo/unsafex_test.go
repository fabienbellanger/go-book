package main

import (
	"testing"
	"unsafe"
)

// Le padding rend Padded plus grosse que Packed alors que les champs sont identiques.
func TestPadding(t *testing.T) {
	if PaddedSize() != 24 {
		t.Errorf("PaddedSize = %d ; attendu 24", PaddedSize())
	}
	if PackedSize() != 16 {
		t.Errorf("PackedSize = %d ; attendu 16", PackedSize())
	}
	if PaddedSize() <= PackedSize() {
		t.Error("réordonner les champs devrait réduire la taille")
	}
}

func TestFieldOffsets(t *testing.T) {
	a, b, c := FieldOffsets()
	if a != 0 || b != 8 || c != 16 {
		t.Errorf("offsets = (%d,%d,%d) ; attendu (0,8,16)", a, b, c)
	}
}

// Conversion sans copie : le contenu est correct ET le backing est partagé.
func TestZeroCopyConversion(t *testing.T) {
	src := []byte("partagé sans copie")
	s := BytesToString(src)
	if s != "partagé sans copie" {
		t.Errorf("BytesToString = %q", s)
	}
	// Même pointeur de backing -> aucune copie.
	if unsafe.Pointer(unsafe.StringData(s)) != unsafe.Pointer(unsafe.SliceData(src)) {
		t.Error("BytesToString devrait partager le backing (zéro copie)")
	}

	back := StringToBytes("aller-retour")
	if string(back) != "aller-retour" {
		t.Errorf("StringToBytes = %q", back)
	}
	if BytesToString(nil) != "" || StringToBytes("") != nil {
		t.Error("les cas vides devraient donner \"\" et nil")
	}
}

// unsafe.String n'alloue pas, contrairement à string([]byte) qui copie.
func TestZeroCopyNoAlloc(t *testing.T) {
	src := []byte("charge utile pour mesurer l'absence d'allocation")
	allocs := testing.AllocsPerRun(100, func() {
		sink = BytesToString(src)
	})
	if allocs != 0 {
		t.Errorf("BytesToString = %.1f alloc/op ; attendu 0 (zéro copie)", allocs)
	}
}

func TestSecondElem(t *testing.T) {
	arr := [4]int32{10, 20, 30, 40}
	if got := SecondElem(&arr); got != 20 {
		t.Errorf("SecondElem = %d ; attendu 20", got)
	}
}

var sink string
