// Programme « Hello » du chapitre 1.
// Lancement : depuis code/, `go run ./ch01-hello`
package main

import "fmt"

// greet construit le message de bienvenue pour name.
// On isole la logique dans une fonction pour pouvoir la tester (voir main_test.go).
func greet(name string) string {
	return fmt.Sprintf("Bonjour, %s ! 👋", name) // commentaire en français
}

func main() {
	fmt.Println(greet("Go")) // affiche : Bonjour, Go ! 👋
}
