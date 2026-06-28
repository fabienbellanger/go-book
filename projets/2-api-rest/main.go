// Commande tasksd : une petite API REST de gestion de tâches.
//
//	tasksd [-addr :8080] [-audit fichier.log] [-origins https://app.exemple]
//
// L'API expose des tâches (CRUD JSON) sur /api/tasks, avec routage par méthode
// (Go 1.22), journalisation structurée (slog), protection CSRF (Go 1.25) et
// arrêt propre sur SIGINT/SIGTERM.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"example.com/tasksapi/internal/api"
)

func main() {
	// signal.NotifyContext annule ctx à la première réception de SIGINT/SIGTERM :
	// c'est le déclencheur de l'arrêt propre, propagé jusqu'à api.Run.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Comme pour le Projet 1, toute la logique vit dans Run (testable), et main
	// se contente de traduire le code de retour en code de sortie du processus.
	os.Exit(api.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
