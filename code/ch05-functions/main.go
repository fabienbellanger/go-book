// Démonstrations du chapitre 5 : retours multiples, variadiques, fonctions-valeurs,
// passage par valeur vs pointeur, et option pattern.
// Lancement : depuis code/, `go run ./ch05-functions`
package main

import "fmt"

func main() {
	// --- Retours multiples + affectation simultanée.
	q, r := divmod(17, 5)
	fmt.Printf("divmod(17,5)   : q=%d r=%d\n", q, r)

	// --- Convention « erreur en dernier ».
	if _, err := safeDivide(1, 0); err != nil {
		fmt.Println("safeDivide(1,0):", err)
	}

	// --- Variadique : 0, 1 ou N arguments, ou un slice « éclaté » avec ...
	fmt.Println("sum(1,2,3)     :", sum(1, 2, 3))
	xs := []int{4, 5, 6}
	fmt.Println("sum(xs...)     :", sum(xs...))

	// --- Retours nommés.
	lo, hi := minMax(3, 9, 1, 7)
	fmt.Printf("minMax         : lo=%d hi=%d\n", lo, hi)

	// --- Fonction passée en paramètre (fonctions = valeurs de 1re classe).
	squared := apply([]int{1, 2, 3, 4}, func(n int) int { return n * n })
	fmt.Println("apply(carré)   :", squared)

	// --- Récursivité.
	fmt.Println("factorial(5)   :", factorial(5))

	// --- Passage PAR VALEUR vs POINTEUR.
	c := counter{n: 0}
	incVal(c) // copie -> aucun effet
	fmt.Printf("après incVal   : c.n=%d (inchangé)\n", c.n)
	incPtr(&c) // pointeur -> effet
	fmt.Printf("après incPtr   : c.n=%d (modifié)\n", c.n)

	// --- Slice : header copié, mais tableau sous-jacent partagé.
	nums := []int{1, 2, 3}
	scale(nums, 10)
	fmt.Println("après scale    :", nums)

	// --- Functional options.
	s := NewServer(WithPort(9000), WithTLS())
	fmt.Printf("server         : %+v\n", *s)
}
