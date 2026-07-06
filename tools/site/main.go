// Commande gobook-site : générateur de site HTML statique du livre.
//
// Elle lit SOMMAIRE.md et les fichiers Markdown référencés, les convertit en
// HTML (goldmark + chroma), construit un index de recherche client et écrit un
// site statique navigable dans le dossier de sortie.
//
// Usage :
//
//	go run ./tools/site [flags]
//
// Voir les flags avec `go run ./tools/site -h`.
package main

import (
	"embed"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"example.com/gobook-site/internal/site"
	"example.com/gobook-site/internal/sommaire"
)

//go:embed assets
var assetsFS embed.FS

func main() {
	var (
		src     = flag.String("src", ".", "racine du livre à lire")
		out     = flag.String("out", "public", "dossier de sortie")
		serve   = flag.Bool("serve", false, "sert le résultat en HTTP local après génération")
		addr    = flag.String("addr", ":8080", "adresse du serveur de prévisualisation")
		clean   = flag.Bool("clean", false, "vide le dossier de sortie avant génération")
		title   = flag.String("title", "Comprendre et maîtriser Go 1.26", "titre du livre")
		version = flag.String("version", "v1.0.0", "version affichée dans le pied de page")
		verbose = flag.Bool("v", false, "logs verbeux (niveau debug)")
	)
	flag.Parse()

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	if err := run(logger, *src, *out, *title, *version, *clean, *serve, *addr); err != nil {
		logger.Error("échec de la génération", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, src, out, title, version string, clean, serve bool, addr string) error {
	if clean {
		logger.Debug("nettoyage du dossier de sortie", "out", out)
		if err := os.RemoveAll(out); err != nil {
			return err
		}
	}

	// 1. Parse de la navigation.
	sommairePath := filepath.Join(src, "SOMMAIRE.md")
	f, err := os.Open(sommairePath)
	if err != nil {
		return err
	}
	book, err := sommaire.Parse(f, title)
	f.Close()
	if err != nil {
		return err
	}
	sommaire.ResolveDirLinks(book, src) // README.md des projets → pages rendues
	book.Version = version
	book.Year = time.Now().Year()
	logger.Debug("sommaire parsé", "parties", len(book.Parts), "pages", len(book.Pages))

	// 2. Assemblage du site.
	assets, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return err
	}
	builder, err := site.New(src, out, assets, logger)
	if err != nil {
		return err
	}
	if err := builder.Build(book); err != nil {
		return err
	}

	// 3. Prévisualisation HTTP optionnelle.
	if serve {
		logger.Info("serveur de prévisualisation", "url", "http://localhost"+addr)
		return http.ListenAndServe(addr, http.FileServer(http.Dir(out)))
	}
	return nil
}
