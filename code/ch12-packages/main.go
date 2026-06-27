// Démonstration du chapitre 12 : organisation en packages et dépendance vers un
// package internal/ du même projet. Lancement : depuis code/, `go run ./ch12-packages`
package main

import (
	"fmt"

	"example.com/gobook/ch12-packages/internal/money"
)

func main() {
	price := money.Euros(19, 99)
	shipping := money.Euros(4, 50)
	total := price.Add(shipping)

	// money.Amount implémente fmt.Stringer : %s (et Println) l'affichent formaté.
	fmt.Printf("prix    : %s\n", price)
	fmt.Printf("port    : %s\n", shipping)
	fmt.Printf("total   : %s\n", total)
}
