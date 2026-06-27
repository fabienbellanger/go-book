// Démonstrations du chapitre 4 : if/for/range/switch et étiquettes.
// Lancement : depuis code/, `go run ./ch04-controlflow`
package main

import "fmt"

func main() {
	// --- if avec instruction d'initialisation : v n'existe que dans le if/else.
	if v := classify(85); v == "B" {
		fmt.Println("if-init   :", v)
	}

	// --- Les trois formes de for.
	sum := 0
	for i := 1; i <= 3; i++ { // 1. for classique
		sum += i
	}
	for sum < 10 { // 2. for-condition (= while)
		sum++
	}
	// 3. for sans condition (= boucle infinie) -> voir le break plus bas.
	fmt.Println("for sum   :", sum)

	// --- for range sur un entier (🆕 1.22).
	fmt.Print("range 3   : ")
	for i := range 3 {
		fmt.Print(i, " ")
	}
	fmt.Println()

	// --- Portée PAR ITÉRATION (🆕 1.22) : chaque closure capture sa propre copie.
	funcs := make([]func() int, 0, 3)
	for i := range 3 {
		funcs = append(funcs, func() int { return i })
	}
	fmt.Print("closures  : ")
	for _, f := range funcs {
		fmt.Print(f(), " ") // 0 1 2 (et non 3 3 3)
	}
	fmt.Println()

	// --- range sur une string : index OCTET + rune (point de code).
	fmt.Print("string    : ")
	for i, r := range "héllo" {
		fmt.Printf("[%d:%c] ", i, r) // 'é' occupe 2 octets -> l'index saute de 1 à 3
	}
	fmt.Println()

	// --- FizzBuzz et break étiqueté.
	fmt.Println("fizzbuzz  :", fizzbuzz(15))
	grid := [][]int{{1, 2, 3}, {4, 5, 6}}
	if i, j, ok := firstPair(grid, 5); ok {
		fmt.Printf("firstPair : trouvé 5 en (%d,%d)\n", i, j)
	}
}
