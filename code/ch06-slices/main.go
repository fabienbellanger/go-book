// Démonstrations du chapitre 6 : header de slice, append, aliasing, arrays.
// Lancement : depuis code/, `go run ./ch06-slices`
package main

import (
	"fmt"
	"slices"
)

func main() {
	// --- Header : len et cap distincts.
	s := make([]int, 3, 8) // len=3, cap=8
	fmt.Printf("make([]int,3,8) : len=%d cap=%d %v\n", len(s), cap(s), s)

	// --- Croissance de cap au fil des append (valeurs observées sur 1.26).
	g := []int{}
	prev := -1
	fmt.Print("cap growth      : ")
	for i := range 600 {
		if cap(g) != prev {
			fmt.Printf("%d ", cap(g))
			prev = cap(g)
		}
		g = append(g, i)
	}
	fmt.Println()

	// --- nil slice vs slice vide.
	var nilS []int
	fmt.Printf("nil slice       : ==nil? %t len=%d (append marche : %v)\n",
		nilS == nil, len(nilS), append(nilS, 1))

	// --- ALIASING : un sous-slice partage le tableau du parent.
	base := []int{1, 2, 3, 4, 5}
	sub := base[1:3]      // [2 3], cap=4 -> jusqu'à la fin de base
	sub = append(sub, 99) // cap le permet : écrit dans base[3] !
	fmt.Println("aliasing        :", base, "<- append(sub,99) a modifié base")

	// --- Three-index : borne le cap, l'append réalloue, le parent est protégé.
	base2 := []int{1, 2, 3, 4, 5}
	sub2 := append(base2[1:3:3], 99)
	fmt.Println("3-index protégé :", base2, "| sub2:", sub2)

	// --- Array : type valeur (copié), comparable.
	a := [3]int{1, 2, 3}
	b := a
	b[0] = 99
	fmt.Printf("array           : a=%v b=%v a==b? %t\n", a, b, a == b)

	// --- Helpers du package slices (🆕 1.21).
	xs := []int{3, 1, 2}
	clone := slices.Clone(xs) // copie indépendante
	slices.Sort(clone)
	fmt.Printf("slices          : Sort(clone)=%v Contains(xs,2)=%t xs=%v\n",
		clone, slices.Contains(xs, 2), xs)

	// --- Fonctions de l'exemple.
	reverseInts(xs)
	fmt.Println("reverseInts     :", xs)
	fmt.Println("filter (pairs)  :", filter([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 }))
	fmt.Println("chunk (taille 2):", chunk([]int{1, 2, 3, 4, 5}, 2))
}
