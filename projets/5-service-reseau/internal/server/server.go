// Package server implémente le serveur clé-valeur TCP : une boucle d'acceptation
// qui délègue chaque connexion à sa propre goroutine, avec deadlines, arrêt
// propre et magasin concurrent.
package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"example.com/kvd/internal/protocol"
)

// Options configure un serveur. Les zéros sont remplacés par des valeurs sûres.
type Options struct {
	Logger       *slog.Logger
	IdleTimeout  time.Duration // délai d'inactivité avant fermeture d'une connexion
	WriteTimeout time.Duration // délai max pour écrire une réponse
	GraceTimeout time.Duration // délai laissé aux connexions en cours lors de l'arrêt
}

// Server sert le protocole clé-valeur sur un net.Listener.
type Server struct {
	store        *kvStore
	log          *slog.Logger
	idleTimeout  time.Duration
	writeTimeout time.Duration
	grace        time.Duration

	mu    sync.Mutex            // protège conns
	conns map[net.Conn]struct{} // connexions actives, pour les fermer à l'arrêt
	wg    sync.WaitGroup        // suit les goroutines de connexion
}

// New crée un serveur prêt à l'emploi.
func New(opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if opts.IdleTimeout <= 0 {
		opts.IdleTimeout = 60 * time.Second
	}
	if opts.WriteTimeout <= 0 {
		opts.WriteTimeout = 10 * time.Second
	}
	if opts.GraceTimeout < 0 {
		opts.GraceTimeout = 0
	}
	return &Server{
		store:        newKVStore(),
		log:          opts.Logger,
		idleTimeout:  opts.IdleTimeout,
		writeTimeout: opts.WriteTimeout,
		grace:        opts.GraceTimeout,
		conns:        make(map[net.Conn]struct{}),
	}
}

// Serve accepte les connexions jusqu'à l'annulation de ctx, puis effectue un
// arrêt propre : il cesse d'accepter, laisse aux connexions en cours un délai de
// grâce, puis attend leur fin. Renvoie nil sur arrêt demandé.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	s.log.Info("écoute", "addr", ln.Addr().String())

	// Surveillant d'arrêt : à l'annulation, on ferme le listener (ce qui
	// débloque Accept) puis, après le délai de grâce, toutes les connexions
	// encore ouvertes (ce qui débloque leurs lectures).
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
		case <-stopped:
			return
		}
		ln.Close()
		if s.grace > 0 {
			time.AfterFunc(s.grace, s.closeAllConns)
		} else {
			s.closeAllConns()
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			close(stopped)
			if ctx.Err() != nil {
				s.wg.Wait() // attend les connexions en cours
				s.log.Info("arrêt propre terminé")
				return nil
			}
			return err
		}

		s.trackConn(conn)
		s.wg.Go(func() { // WaitGroup.Go (Go 1.25) : Add + go + Done en un appel
			defer s.untrackConn(conn)
			s.handleConn(conn)
		})
	}
}

// handleConn traite les requêtes d'une connexion, en série, jusqu'à fermeture,
// inactivité ou erreur. Une connexion = une goroutine.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	br := bufio.NewReader(conn) // évite un syscall par champ lu

	for {
		// Deadline de lecture = horloge anti-inactivité, réarmée à chaque requête.
		conn.SetReadDeadline(time.Now().Add(s.idleTimeout))
		req, err := protocol.ReadRequest(br)
		if err != nil {
			// EOF (client parti) et connexion fermée à l'arrêt sont normaux.
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				s.log.Debug("fin de connexion", "remote", remote, "err", err)
			}
			return
		}

		resp := s.dispatch(req)

		conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
		if err := protocol.WriteResponse(conn, resp); err != nil {
			return
		}
	}
}

// dispatch exécute une requête contre le magasin et construit la réponse.
func (s *Server) dispatch(req protocol.Request) protocol.Response {
	switch req.Op {
	case protocol.OpGet:
		if v, ok := s.store.Get(req.Key); ok {
			return protocol.Response{Status: protocol.StatusOK, Value: v}
		}
		return protocol.Response{Status: protocol.StatusNotFound}
	case protocol.OpSet:
		s.store.Set(req.Key, req.Value)
		return protocol.Response{Status: protocol.StatusOK}
	case protocol.OpDel:
		if s.store.Delete(req.Key) {
			return protocol.Response{Status: protocol.StatusOK}
		}
		return protocol.Response{Status: protocol.StatusNotFound}
	case protocol.OpPing:
		return protocol.Response{Status: protocol.StatusOK, Value: []byte("pong")}
	default:
		return protocol.Response{Status: protocol.StatusError, Err: "opération inconnue"}
	}
}

func (s *Server) trackConn(c net.Conn) {
	s.mu.Lock()
	s.conns[c] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) untrackConn(c net.Conn) {
	s.mu.Lock()
	delete(s.conns, c)
	s.mu.Unlock()
}

func (s *Server) closeAllConns() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.conns {
		c.Close()
	}
}
