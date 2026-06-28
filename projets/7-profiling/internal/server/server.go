// Package server expose l'analyse de mots derrière une API HTTP — la même
// ossature que le Projet 2 (mux, middlewares, arrêt propre) — mais pensée pour
// être PROFILÉE : endpoints pprof montés, et un FlightRecorder qui capture une
// trace dès qu'une requête dépasse un seuil de latence.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime/trace"
	"strconv"
	"sync"
	"time"

	"example.com/wordstats/internal/analyze"
)

// Options configure le serveur.
type Options struct {
	Logger   *slog.Logger
	SlowReq  time.Duration // au-delà, on capture une trace (0 = désactivé)
	TraceDir string        // répertoire où écrire les traces du FlightRecorder
}

// Server enveloppe le routeur et le FlightRecorder.
type Server struct {
	log      *slog.Logger
	slowReq  time.Duration
	traceDir string

	fr     *trace.FlightRecorder // fenêtre glissante de trace (Go 1.25)
	frMu   sync.Mutex            // sérialise les WriteTo concurrents
	frSeq  int                   // numérote les fichiers de trace
	Routes http.Handler
}

// New construit le serveur, monte les routes (analyse + pprof) et, si un seuil
// de latence est fixé, démarre l'enregistreur de vol.
func New(opts Options) *Server {
	log := opts.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	s := &Server{log: log, slowReq: opts.SlowReq, traceDir: opts.TraceDir}

	if s.slowReq > 0 {
		// MinAge borne l'ancienneté des évènements gardés dans la fenêtre : on
		// veut juste de quoi reconstituer ce qui précède une requête lente.
		s.fr = trace.NewFlightRecorder(trace.FlightRecorderConfig{MinAge: 2 * time.Second})
		if err := s.fr.Start(); err != nil {
			s.log.Warn("FlightRecorder non démarré", "err", err)
			s.fr = nil
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /stats", s.handleStats)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	registerPprof(mux) // /debug/pprof/...
	s.Routes = mux
	return s
}

// handleStats lit le corps de la requête et renvoie les n mots les plus
// fréquents en JSON. Le paramètre impl choisit l'implémentation à profiler :
//
//	POST /stats?n=10&impl=v1   (naïve, regexp)   |   impl=v2 (optimisée, défaut)
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	n := 10
	if v := r.URL.Query().Get("n"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			n = parsed
		}
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<20))
	if err != nil {
		http.Error(w, "corps illisible", http.StatusBadRequest)
		return
	}

	var top []analyze.Count
	switch r.URL.Query().Get("impl") {
	case "v1":
		top = analyze.TopWordsRegexp(string(body), n)
	default:
		top = analyze.TopWordsScan(string(body), n)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(top); err != nil {
		s.log.Error("encodage JSON", "err", err)
	}

	// Si la requête a été lente, on fige la fenêtre de trace pour analyse a posteriori.
	if d := time.Since(start); s.slowReq > 0 && d >= s.slowReq {
		s.captureTrace(d)
	}
}

// captureTrace écrit la fenêtre courante du FlightRecorder dans un fichier, à
// analyser avec « go tool trace ». On sérialise les écritures et on numérote les
// fichiers. C'est LE motif du FlightRecorder : capturer le passé juste APRÈS un
// évènement rare (ici une requête lente), sans tracer en continu.
func (s *Server) captureTrace(d time.Duration) {
	if s.fr == nil {
		return
	}
	s.frMu.Lock()
	defer s.frMu.Unlock()
	s.frSeq++
	path := filepath.Join(s.traceDir, fmt.Sprintf("slow-%d.trace", s.frSeq))
	f, err := os.Create(path)
	if err != nil {
		s.log.Error("création de la trace", "err", err)
		return
	}
	defer f.Close()
	if _, err := s.fr.WriteTo(f); err != nil {
		s.log.Error("écriture de la trace", "err", err)
		return
	}
	s.log.Warn("requête lente : trace capturée", "durée", d.String(), "fichier", path)
}

// Serve lance le serveur HTTP et l'arrête proprement à l'annulation du contexte.
func (s *Server) Serve(ctx context.Context, srv *http.Server) error {
	srv.Handler = s.Routes
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if s.fr != nil {
			s.fr.Stop()
		}
		return srv.Shutdown(shutCtx)
	}
}

// registerPprof monte les handlers de net/http/pprof sur le mux fourni. En
// production réelle on les protégerait (auth, réseau interne) — ici, c'est l'outil
// de travail du capstone.
func registerPprof(mux *http.ServeMux) {
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile) // CPU
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
}
