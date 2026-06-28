package set_test

import (
	"fmt"

	"example.com/gends/set"
)

func ExampleOf() {
	s := set.Of(3, 1, 2, 2, 1) // doublons fusionnés
	fmt.Println(s.Len())
	fmt.Println(s.Contains(2))
	fmt.Println(set.Sorted(s)) // tri pour un affichage déterministe
	// Output:
	// 3
	// true
	// [1 2 3]
}

func ExampleSet_Union() {
	a := set.Of("go", "rust")
	b := set.Of("rust", "zig")
	fmt.Println(set.Sorted(a.Union(b)))
	// Output: [go rust zig]
}

func ExampleSet_Intersect() {
	a := set.Of(1, 2, 3, 4)
	b := set.Of(2, 4, 6)
	fmt.Println(set.Sorted(a.Intersect(b)))
	// Output: [2 4]
}
