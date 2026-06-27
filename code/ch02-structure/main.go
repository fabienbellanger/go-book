// Programme illustrant packages, import et points d'initialisation (Ch. 2).
// Lancement : depuis code/, `go run ./ch02-structure`
package main

import (
	"fmt"

	"example.com/gobook/ch02-structure/greeting" // import d'un package local au module
)

// version est une variable de package, initialisée avant tout init() du package main.
var version = "1.0"

func init() {
	// Cet init s'exécute APRÈS l'init du package importé (greeting),
	// car un package importé est entièrement initialisé avant celui qui l'importe.
	fmt.Println("[init main] version", version)
}

func main() {
	fmt.Println(greeting.Greet("fr", "Go"))
	fmt.Println(greeting.Greet("en", "Gopher"))
	fmt.Println(greeting.Greet("xx", "Inconnu")) // langue inconnue -> repli sur fr
}
