// Package cli implémente le dispatch des sous-commandes de txtkit.
//
// Le point d'entrée est Run : il reçoit les arguments (sans le nom du
// programme), les flux d'entrée/sortie, et renvoie le code de retour Unix
// attendu (0 = succès, 1 = erreur de traitement, 2 = erreur d'usage).
// Ce découplage rend chaque sous-commande testable sans toucher au système.
package cli

import (
	"fmt"
	"io"
)

// version est injectable à la compilation :
//
//	go build -ldflags "-X example.com/txtkit/internal/cli.version=v1.2.3"
var version = "dev"

// Run aiguille vers la sous-commande demandée et renvoie le code de retour.
//
//	stdin  : source par défaut quand aucune liste de fichiers n'est fournie ;
//	stdout : sortie normale (résultats) ;
//	stderr : messages d'erreur et d'usage.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "count":
		return runCount(rest, stdin, stdout, stderr)
	case "freq":
		return runFreq(rest, stdin, stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "txtkit %s\n", version)
		return 0
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "txtkit : sous-commande inconnue %q\n\n", cmd)
		usage(stderr)
		return 2
	}
}

// usage affiche l'aide générale.
func usage(w io.Writer) {
	fmt.Fprint(w, `txtkit — boîte à outils de traitement de fichiers texte

Usage :
  txtkit <commande> [options] [fichiers...]

Commandes :
  count    Compte lignes, mots, runes et octets (façon « wc »).
  freq     Affiche les mots les plus fréquents (top N).
  version  Affiche la version.
  help     Affiche cette aide.

Sans fichier, l'entrée standard est lue. Exemples :
  txtkit count *.go
  txtkit count < article.txt
  txtkit freq -n 5 -j 4 *.md

Détail des options d'une commande : « txtkit <commande> -h ».
`)
}
