// Package cli expose le point d'entrée d'enumgen sous forme testable.
//
// Run reçoit ses arguments et ses flux, et renvoie le code de retour Unix
// (0 = succès, 1 = erreur de génération, 2 = erreur d'usage). enumgen est conçu
// pour être appelé par « go generate » :
//
//	//go:generate enumgen -type=Color
//
// Le répertoire courant lors de l'appel est celui du fichier source ; c'est
// pourquoi, par défaut, on analyse « . » et on écrit à côté.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"example.com/enumgen/internal/generator"
)

// version est injectable à la compilation :
//
//	go build -ldflags "-X example.com/enumgen/internal/cli.version=v1.2.3"
var version = "dev"

// Run analyse les drapeaux, lance la génération et écrit le fichier de sortie.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("enumgen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { usage(stderr, fs) }

	dir := fs.String("dir", ".", "répertoire du paquet à analyser")
	out := fs.String("out", "", "fichier de sortie (défaut : <paquet>_enum.go dans -dir)")
	dryRun := fs.Bool("n", false, "afficher le code généré sur stdout sans écrire de fichier")
	showVersion := fs.Bool("version", false, "afficher la version puis quitter")

	if err := fs.Parse(args); err != nil {
		return 2 // flag a déjà écrit le message et l'usage
	}
	if *showVersion {
		fmt.Fprintf(stdout, "enumgen %s\n", version)
		return 0
	}

	// La commande reconstituée sert d'en-tête « Code généré par … ».
	command := "enumgen " + strings.Join(args, " ")
	code, err := generator.Generate(*dir, strings.TrimSpace(command))
	if err != nil {
		fmt.Fprintf(stderr, "enumgen : %v\n", err)
		return 1
	}
	if code == nil {
		fmt.Fprintln(stderr, "enumgen : aucun type annoté //enumgen:stringer trouvé")
		return 1
	}

	if *dryRun {
		stdout.Write(code)
		return 0
	}

	outPath := *out
	if outPath == "" {
		pkg := generator.PackageName(code)
		outPath = filepath.Join(*dir, pkg+"_enum.go")
	}
	if err := os.WriteFile(outPath, code, 0o644); err != nil {
		fmt.Fprintf(stderr, "enumgen : écriture de %s : %v\n", outPath, err)
		return 1
	}
	fmt.Fprintf(stdout, "enumgen : %s écrit\n", outPath)
	return 0
}

func usage(w io.Writer, fs *flag.FlagSet) {
	fmt.Fprint(w, `enumgen — génère des méthodes String() pour les énumérations annotées.

Annoter le type avec une directive (Go 1.26 : ast.ParseDirective la décode) :

  //enumgen:stringer trimprefix=Color
  type Color int

Puis, dans le même paquet :

  //go:generate enumgen
  go generate ./...

Options :
`)
	fs.PrintDefaults()
}
