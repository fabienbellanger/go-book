package lru_test

import (
	"fmt"

	"example.com/gends/lru"
)

func ExampleCache() {
	c := lru.New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Get("a")    // « a » redevient récent ; « b » devient le plus ancien
	c.Put("c", 3) // capacité dépassée : « b » est évincé

	fmt.Println(c.Contains("b"))
	fmt.Println(c.Keys()) // du plus récent au plus ancien
	// Output:
	// false
	// [c a]
}
