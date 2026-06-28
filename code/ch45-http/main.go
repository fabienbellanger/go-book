// code/ch45-http/main.go
//
// Démonstrations de net/http côté serveur ET client :
//   - un ServeMux avec routage enrichi 1.22 (méthode + wildcard {id})
//   - un middleware de logging (func(http.Handler) http.Handler)
//   - un client à timeout et un RoundTripper qui ajoute un en-tête.
//
// Tout est exécutable et testé via net/http/httptest (voir http_test.go) :
// aucun port réel n'est ouvert.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// newMux construit le routeur du service. Le routage enrichi (Go 1.22) permet
// d'indiquer la MÉTHODE et un WILDCARD nommé directement dans le motif.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	// « GET /items/{id} » : ne matche que GET, et capture {id}.
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id") // valeur du wildcard nommé
		// ⚠️ écrire l'en-tête AVANT le corps : tout WriteHeader après un Write
		// est ignoré (le code 200 est déjà parti).
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "item %s", id)
	})

	// « POST /items » : création. Le corps est lu depuis r.Body.
	mux.HandleFunc("POST /items", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "corps illisible", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "créé: %s", body)
	})

	return mux
}

// logging est un middleware : il enveloppe un Handler et journalise chaque
// requête. La signature func(http.Handler) http.Handler se compose à l'infini.
func logging(logf func(string, ...any)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logf("%s %s en %s", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

// headerRoundTripper est un RoundTripper qui injecte un en-tête dans CHAQUE
// requête sortante avant de déléguer au transport sous-jacent. C'est le
// « middleware côté client » : authentification, traçage, retry, etc.
type headerRoundTripper struct {
	key, value string
	next       http.RoundTripper // nil -> http.DefaultTransport
}

func (t headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// On clone la requête : un RoundTripper ne doit PAS muter celle reçue.
	clone := req.Clone(req.Context())
	clone.Header.Set(t.key, t.value)
	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(clone)
}

// fetch effectue un GET avec un client à timeout. defer resp.Body.Close() est
// OBLIGATOIRE, et drainer le corps permet la réutilisation de la connexion.
func fetch(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func main() {
	mux := newMux()
	handler := logging(log.Printf)(mux) // mux enveloppé par le middleware

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second, // ⚠️ indispensable en prod (anti Slowloris)
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Println("écoute sur", srv.Addr, "(Ctrl-C pour arrêter)")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
