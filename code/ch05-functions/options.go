package main

// Server est configuré via le « functional options pattern », un idiome Go très
// répandu : une fonction de construction variadique qui accepte des options.
type Server struct {
	host string
	port int
	tls  bool
}

// Option est une fonction qui modifie un Server en cours de construction.
// (Le fait qu'une Option « capture » son argument relève des closures : Ch. 15.)
type Option func(*Server)

// WithHost, WithPort, WithTLS produisent chacune une Option.
func WithHost(h string) Option { return func(s *Server) { s.host = h } }
func WithPort(p int) Option    { return func(s *Server) { s.port = p } }
func WithTLS() Option          { return func(s *Server) { s.tls = true } }

// NewServer part de valeurs par défaut, puis applique chaque option dans l'ordre.
// L'appelant n'écrit que ce qu'il veut changer : NewServer(WithPort(9000)).
func NewServer(opts ...Option) *Server {
	s := &Server{host: "localhost", port: 8080} // défauts
	for _, opt := range opts {
		opt(s)
	}
	return s
}
