// Commande enumgen : génère des méthodes String() pour les énumérations Go
// annotées d'une directive //enumgen:stringer.
//
// Usage typique, via go generate :
//
//	//enumgen:stringer trimprefix=Color
//	type Color int
//
//	//go:generate enumgen
//
//	$ go generate ./...
//
// enumgen analyse le paquet courant (drapeau -dir), repère les types annotés,
// évalue leurs constantes et écrit « <paquet>_enum.go » à côté.
package main

import (
	"os"

	"example.com/enumgen/internal/cli"
)

func main() {
	// Tout le travail est délégué à cli.Run, testable car il reçoit ses flux
	// en paramètres et renvoie le code de retour du processus.
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
