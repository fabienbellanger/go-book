package main

import (
	"fmt"
	"strings"
)

func main() {
	// Au moment où main démarre, toute l'initialisation est DÉJÀ terminée :
	// packages importés, variables, puis init(). main est le dernier maillon.
	fmt.Println("Ordre d'initialisation observé :")
	fmt.Println("  " + strings.Join(InitOrder(), " -> "))

	info := CurrentRuntime()
	fmt.Printf("\nRuntime : %s, %s/%s\n", info.Version, info.GOOS, info.GOARCH)
	fmt.Printf("NumCPU=%d  GOMAXPROCS=%d\n", info.NumCPU, info.GOMAXPROCS)

	// Astuce : pour voir l'init de TOUS les packages (stdlib comprise), lancer
	//   GODEBUG=inittrace=1 go run ./ch24-runtime-bootstrap
}
