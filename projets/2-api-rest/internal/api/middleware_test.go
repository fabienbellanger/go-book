package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRecoverPanic vérifie qu'une panique d'un handler devient une réponse 500
// au lieu de faire tomber le serveur.
func TestRecoverPanic(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boum")
	})
	h := recoverPanic(log)(boom)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, voulu 500", rec.Code)
	}
}

// TestRequestID vérifie qu'un identifiant est généré et renvoyé, et qu'un
// identifiant fourni par le client est conservé.
func TestRequestID(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = requestIDFrom(r.Context())
	})
	h := requestID(next)

	t.Run("généré", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if seen == "" {
			t.Error("aucun identifiant déposé dans le context")
		}
		if rec.Header().Get("X-Request-Id") != seen {
			t.Error("l'en-tête X-Request-Id doit refléter l'identifiant du context")
		}
	})

	t.Run("repris du client", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-Id", "abc-123")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if seen != "abc-123" {
			t.Errorf("identifiant = %q, voulu abc-123", seen)
		}
	})
}
