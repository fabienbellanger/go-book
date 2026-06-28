// Commande wordstats : un service HTTP d'analyse de fréquence de mots, conçu
// comme support du Projet 7 (profiling de bout en bout).
//
// Il reprend l'ossature du Projet 2 (mux, slog, arrêt propre) mais ajoute tout
// l'outillage d'observation : endpoints net/http/pprof et capture de trace par
// FlightRecorder sur requête lente.
//
//	POST /stats?n=10[&impl=v1|v2]   analyse le corps, renvoie le top-N en JSON
//	GET  /healthz                   sonde de vivacité
//	GET  /debug/pprof/...           profils CPU/heap/goroutine/...
//
// Exemple :
//
//	wordstats -addr :8080 -slow 50ms -tracedir /tmp &
//	curl -s --data-binary @testdata/corpus.txt 'localhost:8080/stats?impl=v1' | head
//	go tool pprof 'http://localhost:8080/debug/pprof/profile?seconds=5'
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/wordstats/internal/server"
)

// version est injectable à la compilation (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	addr := flag.String("addr", ":8080", "adresse d'écoute")
	slow := flag.Duration("slow", 0, "seuil de latence déclenchant une capture de trace (0 = désactivé)")
	traceDir := flag.String("tracedir", ".", "répertoire des traces capturées")
	showVersion := flag.Bool("version", false, "afficher la version puis quitter")
	flag.Parse()

	if *showVersion {
		os.Stdout.WriteString("wordstats " + version + "\n")
		return
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(server.Options{Logger: log, SlowReq: *slow, TraceDir: *traceDir})
	httpSrv := &http.Server{
		Addr:              *addr,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("wordstats démarre", "addr", *addr, "version", version)
	if err := srv.Serve(ctx, httpSrv); err != nil && err != http.ErrServerClosed {
		log.Error("serveur arrêté sur erreur", "err", err)
		os.Exit(1)
	}
	log.Info("arrêt propre terminé")
}
