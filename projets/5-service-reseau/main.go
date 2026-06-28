// Commande kvd : un serveur clé-valeur en réseau, parlant un protocole binaire
// maison sur TCP.
//
//	kvd [-addr :7000] [-idle-timeout 60s] [-grace 5s]
//
// Le client de référence est le package internal/client (voir le README).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/kvd/internal/server"
)

// version est injectable à la compilation (cf. Makefile, -ldflags -X).
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

// run configure et lance le serveur. Renvoie le code de retour : 0 succès,
// 1 erreur d'exécution, 2 erreur d'usage.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("kvd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", ":7000", "adresse d'écoute (host:port)")
	idle := fs.Duration("idle-timeout", 60*time.Second, "délai d'inactivité par connexion")
	grace := fs.Duration("grace", 5*time.Second, "délai de grâce à l'arrêt")
	showVersion := fs.Bool("version", false, "afficher la version et quitter")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintf(stdout, "kvd %s\n", version)
		return 0
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintf(stderr, "kvd : écoute impossible : %v\n", err)
		return 1
	}

	srv := server.New(server.Options{
		Logger:       logger,
		IdleTimeout:  *idle,
		GraceTimeout: *grace,
	})
	if err := srv.Serve(ctx, ln); err != nil {
		fmt.Fprintf(stderr, "kvd : %v\n", err)
		return 1
	}
	return 0
}
