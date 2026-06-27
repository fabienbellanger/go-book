package main

import (
	"fmt"
	"math"
)

func main() {
	for _, n := range []int{0, 42, 1234567, -98765, math.MinInt} {
		fmt.Printf("%-21d -> %q\n", n, FormatThousands(n))
	}
	// Les deux implémentations coïncident toujours.
	fmt.Println("naïve == builder :", formatNaive(1234567) == FormatThousands(1234567))
}
