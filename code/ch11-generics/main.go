// Démonstrations du chapitre 11 : types paramétrés, contraintes, alias génériques
// (1.24), contrainte auto-référentielle (1.26), packages slices/maps/cmp.
// Lancement : depuis code/, `go run ./ch11-generics`
package main

import (
	"cmp"
	"fmt"
	"slices"
)

// Person sert à illustrer un tri multi-critères avec cmp.Or.
type Person struct {
	Name string
	Age  int
}

// Celsius est défini sur float64 : grâce au ~ de Number, il satisfait la contrainte.
type Celsius float64

func main() {
	// =========================================================================
	// Inférence de type : on n'écrit pas [string, int], le compilateur déduit.
	// =========================================================================

	words := []string{"go", "rust", "zig"}
	lengths := Map(words, func(s string) int { return len(s) })
	fmt.Printf("map      : %v -> %v\n", words, lengths)

	even := Filter([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 })
	fmt.Printf("filter   : %v\n", even)

	// =========================================================================
	// Contrainte Number (~) : marche pour int ET pour un type défini (Celsius).
	// =========================================================================

	fmt.Printf("sum int  : %d\n", Sum([]int{1, 2, 3, 4}))
	fmt.Printf("sum °C   : %g\n", Sum([]Celsius{19.5, 0.5, 1}))

	// =========================================================================
	// cmp.Ordered : Max sur int comme sur string.
	// =========================================================================

	fmt.Printf("max      : %d / %q\n", Max(3, 7), Max("go", "rust"))
	fmt.Printf("index    : %d\n", Index(words, "zig"))

	// =========================================================================
	// Type générique : Stack[int].
	// =========================================================================

	var st Stack[int]
	st.Push(10)
	st.Push(20)
	st.Push(30)
	top, _ := st.Pop()
	fmt.Printf("stack    : pop=%d len=%d\n", top, st.Len())

	// =========================================================================
	// Alias générique (1.24) : Set[string] EST un map[string]struct{}.
	// =========================================================================

	langs := NewSet("go", "rust", "go", "zig") // doublon "go" absorbé
	fmt.Printf("set      : %d éléments, triés=%v\n", len(langs), SortedKeys(langs))

	// =========================================================================
	// Contrainte auto-référentielle (1.26) : SumAll sur un type « additionnable ».
	// =========================================================================

	total := SumAll([]Vec2{{1, 1}, {2, 3}, {0, 1}})
	fmt.Printf("sumAll   : %+v\n", total)

	// =========================================================================
	// slices + cmp : tri multi-critères (âge croissant, puis nom).
	// =========================================================================

	people := []Person{{"Bob", 30}, {"Ann", 30}, {"Cy", 25}}
	slices.SortFunc(people, func(a, b Person) int {
		return cmp.Or(cmp.Compare(a.Age, b.Age), cmp.Compare(a.Name, b.Name))
	})
	fmt.Printf("sortFunc : %v\n", people)
}
