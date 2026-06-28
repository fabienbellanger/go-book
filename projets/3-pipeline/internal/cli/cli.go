// Package cli câble la commande « pipe » sur la bibliothèque pipeline : elle
// hache en SHA-256, en parallèle, les fichiers passés en arguments ou listés
// sur stdin.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"runtime"
	"slices"

	"example.com/pipeline/internal/pipeline"
)

// version est injectable à la compilation (cf. Makefile, -ldflags -X).
var version = "dev"

// Run exécute la commande et renvoie le code de retour : 0 succès, 1 erreur de
// traitement, 2 erreur d'usage. ctx porte l'annulation (signal d'arrêt).
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("pipe", flag.ContinueOnError)
	fs.SetOutput(stderr)
	workers := fs.Int("j", runtime.GOMAXPROCS(0), "nombre de workers concurrents (fan-out)")
	buffer := fs.Int("buffer", 0, "taille du canal de sorties (pression arrière)")
	rate := fs.Int("rate", 0, "limite de débit en fichiers/seconde (0 = illimité)")
	showVersion := fs.Bool("version", false, "afficher la version et quitter")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage : pipe [-j N] [-buffer N] [-rate N] [fichiers...]")
		fmt.Fprintln(stderr, "Calcule le SHA-256 des fichiers (arguments ou lignes de stdin), en parallèle.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintf(stdout, "pipe %s\n", version)
		return 0
	}

	cfg := pipeline.Config{Workers: *workers, Buffer: *buffer}
	if *rate > 0 {
		lim := pipeline.NewRateLimiter(*rate)
		defer lim.Stop()
		cfg.Limiter = lim
	}

	// Lance le pipeline : la source est paresseuse, les sorties arrivent dans le
	// désordre (concurrence).
	paths := pathsFrom(fs.Args(), stdin)
	out, metrics, wait := pipeline.Process(ctx, paths, hashFile, cfg)

	// Fan-in : on draine TOUTES les sorties avant d'interroger wait().
	results := make([]hashResult, 0)
	for r := range out {
		results = append(results, r)
	}
	err := wait()

	// Tri déterministe par chemin, puis affichage « somme  chemin » (façon shasum).
	slices.SortFunc(results, func(a, b hashResult) int {
		if a.path < b.path {
			return -1
		}
		if a.path > b.path {
			return 1
		}
		return 0
	})
	for _, r := range results {
		fmt.Fprintf(stdout, "%s  %s\n", r.sum, r.path)
	}

	snap := metrics.Snapshot()
	fmt.Fprintf(stderr, "pipe : traités=%d échecs=%d pic_concurrence=%d\n",
		snap.Processed, snap.Failed, snap.MaxInFlight)

	if err != nil {
		fmt.Fprintf(stderr, "pipe : %v\n", err)
		return 1
	}
	return 0
}
