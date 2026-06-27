package main

import "time"

// --- Functional options : des closures configurent un objet à la construction. ---

// Server est l'objet à configurer. Ses champs sont privés : on ne les règle que
// via des Options, ce qui garde l'API stable même si on ajoute des champs.
type Server struct {
	host    string
	port    int
	timeout time.Duration
}

// Option est une closure qui modifie un Server en cours de construction.
type Option func(*Server)

// WithPort capture p et l'applique au Server quand l'option est exécutée.
func WithPort(p int) Option { return func(s *Server) { s.port = p } }

// WithTimeout capture d de la même façon.
func WithTimeout(d time.Duration) Option { return func(s *Server) { s.timeout = d } }

// NewServer part de valeurs par défaut, puis applique chaque option reçue.
func NewServer(host string, opts ...Option) *Server {
	s := &Server{host: host, port: 8080, timeout: 30 * time.Second}
	for _, opt := range opts {
		opt(s) // chaque closure mute s
	}
	return s
}
