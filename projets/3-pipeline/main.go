// Commande pipe : calcule le SHA-256 d'un ensemble de fichiers en parallèle,
// au moyen d'un pipeline concurrent borné.
//
//	pipe [-j N] [-buffer N] [-rate N] [fichiers...]
//
// Sans argument, la liste des fichiers est lue sur stdin (un chemin par ligne).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"example.com/pipeline/internal/cli"
)

func main() {
	// L'annulation par signal (Ctrl-C) se propage au pipeline : feeder et
	// workers s'arrêtent proprement, sans laisser de goroutine en vie.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(cli.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
