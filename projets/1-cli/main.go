// Commande txtkit : une boîte à outils de traitement de fichiers texte.
//
// Sous-commandes :
//
//	txtkit count [-j N] [-total] [fichiers...]   compte lignes, mots, runes, octets
//	txtkit freq  [-j N] [-n N] [fichiers...]      fréquence des mots (top N)
//	txtkit version
//	txtkit help
//
// Sans fichier, l'entrée est lue sur stdin (« txtkit count < data.txt »).
package main

import (
	"os"

	"example.com/txtkit/internal/cli"
)

func main() {
	// Tout le travail est délégué à cli.Run, qui reçoit ses entrées/sorties en
	// paramètres (testable) et renvoie le code de retour du processus.
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
