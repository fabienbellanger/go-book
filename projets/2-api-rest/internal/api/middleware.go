package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// ctxKey est un type privé pour les clés de context : il évite toute collision
// avec les clés d'autres packages (Ch. 22).
type ctxKey int

const requestIDKey ctxKey = iota

// requestID associe un identifiant unique à chaque requête. Il est repris de
// l'en-tête X-Request-Id s'il existe (utile derrière un proxy), sinon généré,
// puis renvoyé au client et déposé dans le context pour la journalisation.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requestIDFrom extrait l'identifiant de requête du context (vide si absent).
func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}

// statusRecorder enveloppe le ResponseWriter pour capturer le code de statut et
// le nombre d'octets écrits — invisibles autrement une fois la réponse partie.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK // Write implicite => 200
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// logging journalise chaque requête (méthode, chemin, statut, durée…) en une
// ligne structurée slog, corrélée par l'identifiant de requête.
func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		s.log.LogAttrs(r.Context(), slog.LevelInfo, "requête HTTP",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Int("bytes", rec.bytes),
			slog.Duration("dur", time.Since(start)),
			slog.String("request_id", requestIDFrom(r.Context())),
		)
	})
}

// recoverPanic transforme une panique d'un handler en réponse 500 propre, au
// lieu de tuer la connexion. Placée en tête de chaîne, elle protège tout le reste.
func recoverPanic(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.LogAttrs(r.Context(), slog.LevelError, "panique récupérée",
						slog.Any("panic", v),
						slog.String("request_id", requestIDFrom(r.Context())),
					)
					// Si l'en-tête est déjà parti, WriteHeader est un no-op
					// (la connexion sera coupée) ; sinon le client reçoit un 500 JSON.
					writeError(w, http.StatusInternalServerError, "erreur interne")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
