package main

import (
	"fmt"
	"testing"
)

func main() {
	// testing.AllocsPerRun mesure le nombre moyen d'allocations TAS par appel.
	// (Hors d'un test, on l'utilise ici juste pour illustrer.)
	stack := testing.AllocsPerRun(100, func() { _ = sumLocalArray(3) })
	heap := testing.AllocsPerRun(100, func() { _ = NewPoint(1, 2) })
	smallSlice := testing.AllocsPerRun(100, func() { _ = sumSmallSlice(3) })
	leak := testing.AllocsPerRun(100, func() { _ = LeakSlice(8) })
	boxed := testing.AllocsPerRun(100, func() { pointToInterface(Point{1, 2}) })

	fmt.Printf("sumLocalArray   : %.0f alloc/op (pile)\n", stack)
	fmt.Printf("NewPoint        : %.0f alloc/op (tas, pointeur renvoyé)\n", heap)
	fmt.Printf("sumSmallSlice   : %.0f alloc/op (backing sur la pile, 1.25/1.26)\n", smallSlice)
	fmt.Printf("LeakSlice       : %.0f alloc/op (tas, slice renvoyé)\n", leak)
	fmt.Printf("pointToInterface: %.0f alloc/op (tas, boxé puis retenu par une interface)\n", boxed)

	fmt.Println("\nPour voir les décisions de l'escape analysis :")
	fmt.Println("  go build -gcflags=-m ./ch26-allocation-escape")
}
