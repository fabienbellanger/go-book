package main

import (
	"fmt"
	"unsafe"
)

func main() {
	s := "héllo, 日本"
	fmt.Printf("Sizeof(string) = %d octets (2 mots : ptr, len)\n", unsafe.Sizeof(s))

	b, r := ByteVsRune(s)
	fmt.Printf("%q : %d octets, %d runes\n", s, b, r)

	fmt.Print("range -> (index octet, largeur) : ")
	for _, w := range RuneWidths(s) {
		fmt.Printf("(%d,%d) ", w[0], w[1])
	}
	fmt.Println()

	fmt.Printf("JoinCSV : %q\n", JoinCSV([]string{"go", "rust", "zig"}))
	fmt.Printf("ToUpperASCII(%q) = %q (é inchangé : hors ASCII)\n", "héllo", ToUpperASCII("héllo"))

	// Interning : mêmes contenus -> mêmes handles.
	a1, a2 := Intern("event.created"), Intern("event.created")
	fmt.Printf("Intern x2 : handles == ? %v ; Value=%q ; taille handle=%d o\n",
		a1 == a2, a1.Value(), unsafe.Sizeof(a1))
	fmt.Printf("CountDistinct([a a b a c]) = %d\n",
		CountDistinct([]string{"a", "a", "b", "a", "c"}))
}
