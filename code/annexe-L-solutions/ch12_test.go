package main

import "fmt"

// ExampleCh12Amount_String verrouille le format de sortie : c'est un test
// exécutable (comparaison de la sortie à // Output:). Casser le format le fait
// échouer (exercice 2).
func Example_ch12Amount() {
	fmt.Println(ch12Amount(1250)) // 12 € et 50 c
	fmt.Println(ch12Amount(-99))  // montant négatif
	// Output:
	// 12.50 €
	// -0.99 €
}
