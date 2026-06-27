package main

import "fmt"

func main() {
	// La stratégie de croissance d'append, rendue visible.
	fmt.Printf("cap au fil des append (nil -> 2000 ints) : %v\n", CapGrowth(2000))

	// Expression à 3 indices : borne la capacité.
	arr := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	fmt.Printf("cap(arr[2:4])   = %d (cap par défaut = jusqu'au bout)\n", cap(arr[2:4]))
	fmt.Printf("cap(arr[2:4:6]) = %d (cap bornée par le 3e indice)\n", SubSliceCap(arr, 2, 4, 6))

	// Aliasing vs isolation.
	p1, m1 := AppendAliasing()
	fmt.Printf("aliasing : parent=%v (parent[2] écrasé par %v)\n", p1, m1)
	p2, m2 := SafeAppend()
	fmt.Printf("isolé   : parent=%v (intact) modified=%v\n", p2, m2)

	// Filtrage sans allocation.
	src := []int{1, 2, 3, 4, 5, 6, 7, 8}
	even := FilterInPlace(src, func(v int) bool { return v%2 == 0 })
	fmt.Printf("FilterInPlace (pairs) : %v\n", even)

	// Rétention mémoire corrigée par Clone.
	big := make([]int, 1_000_000)
	small := TrimRetention(big, 3)
	fmt.Printf("TrimRetention : len=%d cap=%d (le grand backing est libérable)\n",
		len(small), cap(small))
}
