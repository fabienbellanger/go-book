// Command annexe-L-solutions regroupe les solutions des exercices « À tester
// soi-même » de la Partie I (chapitres 2 à 13). Chaque solution vit dans un
// fichier chNN_*.go ; les tests correspondants (chNN_test.go) les vérifient.
//
// Le main lui-même se contente d'un rappel : l'essentiel est dans les tests.
//
//	cd code && go test ./annexe-L-solutions/
package main

import "fmt"

func main() {
	fmt.Println("Solutions des exercices — Partie I (voir les tests : go test ./annexe-L-solutions/)")
	fmt.Println(ch02Greet("de", "Go")) // Hallo, Go !
}
