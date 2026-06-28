// Package api assemble le serveur HTTP : routage par méthode (Go 1.22), chaîne
// de middlewares, handlers JSON et point d'entrée Run (avec arrêt propre).
package api

import (
	"log/slog"
	"net/http"

	"example.com/tasksapi/internal/store"
)

// Server porte les dépendances des handlers (le Store et le logger) et expose
// le routeur déjà enveloppé de ses middlewares. Il implémente http.Handler.
type Server struct {
	store   store.Store
	log     *slog.Logger
	handler http.Handler
}

// NewServer câble les routes et la chaîne de middlewares.
//
// cop (protection CSRF, Go 1.25) est optionnel : passer nil la désactive, ce
// qui est pratique en test. Quand il est fourni, il enveloppe le routeur au plus
// près, juste après que la journalisation a relevé la requête.
func NewServer(st store.Store, log *slog.Logger, cop *http.CrossOriginProtection) *Server {
	s := &Server{store: st, log: log}

	mux := http.NewServeMux()
	// Routage Go 1.22 : méthode + motif. {id} est un wildcard lu via PathValue.
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/tasks", s.handleList)
	mux.HandleFunc("POST /api/tasks", s.handleCreate)
	mux.HandleFunc("GET /api/tasks/{id}", s.handleGet)
	mux.HandleFunc("PUT /api/tasks/{id}", s.handleUpdate)
	mux.HandleFunc("DELETE /api/tasks/{id}", s.handleDelete)

	// Chaîne de middlewares, écrite de l'intérieur vers l'extérieur. À
	// l'exécution, une requête les traverse dans l'ordre inverse :
	//   recoverPanic → requestID → logging → CSRF → mux.
	var h http.Handler = mux
	if cop != nil {
		h = cop.Handler(h)
	}
	h = s.logging(h)
	h = requestID(h)
	h = recoverPanic(s.log)(h)

	s.handler = h
	return s
}

// ServeHTTP rend *Server utilisable comme handler (et donc testable via httptest).
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}
