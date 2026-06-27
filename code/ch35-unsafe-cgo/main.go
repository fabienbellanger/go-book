package main

import (
	"fmt"
	"unsafe"
)

func main() {
	// Padding : l'ordre des champs change la taille.
	a, b, c := FieldOffsets()
	fmt.Printf("Padded : taille=%d offsets(a,b,c)=(%d,%d,%d)\n", PaddedSize(), a, b, c)
	fmt.Printf("Packed : taille=%d (réordonner économise %d octets)\n",
		PackedSize(), PaddedSize()-PackedSize())

	// Alignements de quelques types.
	fmt.Printf("Alignof : int64=%d int32=%d byte=%d\n",
		unsafe.Alignof(int64(0)), unsafe.Alignof(int32(0)), unsafe.Alignof(byte(0)))

	// Conversion sans copie []byte <-> string.
	src := []byte("zéro-copie")
	s := BytesToString(src)
	fmt.Printf("BytesToString(%q) = %q (même backing : %v)\n", src, s,
		unsafe.Pointer(unsafe.StringData(s)) == unsafe.Pointer(unsafe.SliceData(src)))

	// Arithmétique de pointeur via unsafe.Add.
	arr := [4]int32{10, 20, 30, 40}
	fmt.Printf("SecondElem = %d (via unsafe.Add)\n", SecondElem(&arr))
}
