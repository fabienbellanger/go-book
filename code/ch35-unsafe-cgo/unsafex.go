package main

import "unsafe"

// Padded : champs dans un ordre qui force du PADDING. Le compilateur aligne chaque
// champ sur son alignement naturel ; un bool avant un int64 laisse 7 octets vides.
type Padded struct {
	a bool  // offset 0, puis 7 octets de padding
	b int64 // offset 8
	c bool  // offset 16, puis 7 octets de padding final (alignement de la struct = 8)
}

// Packed : MÊMES champs, réordonnés du plus grand au plus petit alignement -> compact.
type Packed struct {
	b int64 // offset 0
	a bool  // offset 8
	c bool  // offset 9 (2 octets de padding final seulement)
}

// PaddedSize et PackedSize exposent les tailles pour comparaison.
func PaddedSize() uintptr { return unsafe.Sizeof(Padded{}) }
func PackedSize() uintptr { return unsafe.Sizeof(Packed{}) }

// FieldOffsets renvoie les offsets des trois champs de Padded.
func FieldOffsets() (a, b, c uintptr) {
	var p Padded
	return unsafe.Offsetof(p.a), unsafe.Offsetof(p.b), unsafe.Offsetof(p.c)
}

// BytesToString convertit un []byte en string SANS copie (unsafe.String).
// ⚠️ La string partage le backing du []byte : ce dernier ne doit PLUS JAMAIS être
// modifié, sinon on viole l'immutabilité de la string (comportement indéfini).
func BytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// StringToBytes : l'inverse, sans copie. Le []byte renvoyé est en LECTURE SEULE de
// fait — le backing d'une string peut résider en mémoire constante du binaire ;
// y écrire est un comportement indéfini.
func StringToBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// SecondElem accède au 2e élément d'un tableau via l'arithmétique de pointeur
// contrôlée d'unsafe.Add (pattern valide : base + offset calculé avec Sizeof).
func SecondElem(arr *[4]int32) int32 {
	base := unsafe.Pointer(arr)
	second := (*int32)(unsafe.Add(base, unsafe.Sizeof(arr[0])))
	return *second
}
