package main

import (
	"fmt"
	"slices"
	"strings"
)

func even(n int) bool  { return n%2 == 0 }
func square(n int) int { return n * n }

func main() {
	// 1. range-over-func sur un itérateur maison.
	fmt.Print("Count(5) : ")
	for i := range Count(5) {
		fmt.Print(i, " ")
	}
	fmt.Println()

	// 2. Composition PARESSEUSE sur une source INFINIE : carrés des pairs, 3
	// premiers. Rien n'est matérialisé ; seuls 3 résultats sont calculés.
	pipeline := Take(Map(Filter(Naturals(), even), square), 3)
	fmt.Println("carrés des pairs [:3] :", slices.Collect(pipeline)) // [0 4 16]

	// 3. Arrêt anticipé : break stoppe toute la chaîne (yield renvoie false).
	fmt.Print("premier carré > 100 : ")
	for n := range Map(Naturals(), square) {
		if n > 100 {
			fmt.Println(n) // 121
			break
		}
	}

	// 4. Enumerate : itérateur à deux valeurs (index, valeur).
	fmt.Print("enumerate : ")
	for i, w := range Enumerate(slices.Values([]string{"go", "rust", "zig"})) {
		fmt.Printf("[%d]=%s ", i, w)
	}
	fmt.Println()

	// 5. Zip via iter.Pull : avancer deux sources en parallèle.
	names := slices.Values([]string{"Ada", "Alan", "Grace"})
	ages := slices.Values([]int{36, 41, 45})
	fmt.Print("zip : ")
	for name, age := range Zip(names, ages) {
		fmt.Printf("%s=%d ", name, age)
	}
	fmt.Println()

	// 6. Itérateurs de la bibliothèque standard (1.23).
	words := strings.Fields("banana apple cherry apple")
	fmt.Println("triés    :", slices.Sorted(slices.Values(words)))
}
