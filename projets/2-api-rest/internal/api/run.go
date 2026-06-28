package api

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"example.com/tasksapi/internal/store"
)

// version est injectable à la compilation (cf. Makefile, -ldflags -X).
var version = "dev"

// Run configure puis lance le serveur jusqu'à l'annulation de ctx (signal
// d'arrêt) ou une erreur fatale. Il renvoie le code de retour du processus :
// 0 succès (arrêt propre), 1 erreur d'exécution, 2 erreur d'usage.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tasksd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", ":8080", "adresse d'écoute (host:port)")
	auditPath := fs.String("audit", "", "fichier de log JSON d'audit (vide = désactivé)")
	origins := fs.String("origins", "", "origines de confiance CSRF, séparées par des virgules")
	shutdownTimeout := fs.Duration("shutdown-timeout", 10*time.Second, "délai max d'arrêt propre")
	showVersion := fs.Bool("version", false, "afficher la version et quitter")
	if err := fs.Parse(args); err != nil {
		return 2 // flag a déjà écrit le message ; -h passe aussi par ici
	}
	if *showVersion {
		fmt.Fprintf(stdout, "tasksd %s\n", version)
		return 0
	}

	// Journalisation : texte lisible sur stderr + (option) JSON d'audit en fichier.
	logger, closeLog, err := newLogger(stderr, *auditPath)
	if err != nil {
		fmt.Fprintf(stderr, "tasksd : %v\n", err)
		return 1
	}
	defer closeLog()

	// Protection CSRF (Go 1.25) : rejette les requêtes cross-origin non sûres,
	// en autorisant les origines déclarées de confiance.
	cop := http.NewCrossOriginProtection()
	for _, o := range splitList(*origins) {
		if err := cop.AddTrustedOrigin(o); err != nil {
			fmt.Fprintf(stderr, "tasksd : origine invalide %q : %v\n", o, err)
			return 2
		}
	}

	srv := NewServer(store.NewMemStore(), logger, cop)
	httpSrv := &http.Server{
		Addr:    *addr,
		Handler: srv,
		// Délais défensifs : sans eux, une connexion lente immobilise une
		// goroutine indéfiniment (Ch. 23, robustesse réseau).
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Le serveur écoute dans sa propre goroutine ; une erreur de démarrage
	// (port occupé…) remonte par errCh.
	errCh := make(chan error, 1)
	go func() {
		logger.Info("démarrage du serveur", "addr", *addr, "version", version)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		fmt.Fprintf(stderr, "tasksd : %v\n", err)
		return 1
	case <-ctx.Done():
		logger.Info("signal d'arrêt reçu, arrêt en cours")
	}

	// Arrêt propre : on laisse les requêtes en cours se terminer, dans la limite
	// du délai. context.Background (et non ctx, déjà annulé) borne cette phase.
	shutCtx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(shutCtx); err != nil {
		logger.Error("arrêt non propre", "err", err)
		return 1
	}
	logger.Info("arrêt propre terminé")
	return 0
}

// newLogger construit le logger. Il combine, via slog.NewMultiHandler (Go 1.25),
// un handler texte (lisible, sur stderr) et — si un fichier d'audit est demandé
// — un handler JSON. Le même enregistrement part alors vers les deux sorties.
func newLogger(stderr io.Writer, auditPath string) (*slog.Logger, func(), error) {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	handlers := []slog.Handler{slog.NewTextHandler(stderr, opts)}
	closeFn := func() {}

	if auditPath != "" {
		f, err := os.OpenFile(auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("ouverture du fichier d'audit : %w", err)
		}
		handlers = append(handlers, slog.NewJSONHandler(f, opts))
		closeFn = func() { f.Close() }
	}

	return slog.New(slog.NewMultiHandler(handlers...)), closeFn, nil
}

// splitList découpe une liste « a, b ,c » en éléments nettoyés, en ignorant les
// entrées vides.
func splitList(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}
