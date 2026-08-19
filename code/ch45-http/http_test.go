// code/ch45-http/http_test.go
//
// Tous les tests s'appuient sur net/http/httptest : pas de port réel, pas d'I/O
// réseau externe. httptest.NewServer démarre un vrai serveur sur un port
// éphémère (127.0.0.1:0) que l'on referme en fin de test ; httptest.NewRecorder
// capture une réponse sans même passer par le réseau.
//
// 🆕 Go 1.27 ajoute httptest.NewTestServer(t, h) : même API de Server, mais sur
// un réseau EN MÉMOIRE (ni socket, ni port), fermé automatiquement par le
// testing.TB. Voir TestInMemoryServer plus bas.
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestRouteWildcard vérifie le routage enrichi 1.22 : la méthode et le wildcard
// {id}. Ici on teste le Handler en isolation avec un ResponseRecorder.
func TestRouteWildcard(t *testing.T) {
	mux := newMux()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, voulu %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "item 42" {
		t.Errorf("corps = %q, voulu %q", got, "item 42")
	}
}

// TestMethodMismatch : un POST sur une route déclarée GET doit donner 405
// (Method Not Allowed) — le ServeMux le gère automatiquement depuis 1.22.
func TestMethodMismatch(t *testing.T) {
	mux := newMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/items/42", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("code = %d, voulu %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestMiddlewareLogging vérifie que le middleware appelle bien le handler suivant
// ET journalise. On capture les logs dans une closure plutôt qu'avec un vrai log.
func TestMiddlewareLogging(t *testing.T) {
	var logged []string
	capture := func(format string, args ...any) {
		logged = append(logged, format)
	}
	handler := logging(capture)(newMux())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/items/7", nil)
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "item 7" {
		t.Errorf("le handler enveloppé n'a pas été appelé: %q", rec.Body.String())
	}
	if len(logged) != 1 {
		t.Errorf("entrées de log = %d, voulu 1", len(logged))
	}
}

// TestClientAgainstServer : un serveur httptest réel + un client à timeout.
// C'est le test d'intégration HTTP idiomatique, sans port fixe.
func TestClientAgainstServer(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close() // referme le listener éphémère

	client := &http.Client{Timeout: 2 * time.Second}
	got, err := fetch(client, srv.URL+"/items/99")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got != "item 99" {
		t.Errorf("réponse = %q, voulu %q", got, "item 99")
	}
}

// TestRoundTripper vérifie que headerRoundTripper injecte bien l'en-tête dans
// la requête sortante. Le serveur renvoie la valeur reçue, on la compare.
func TestRoundTripper(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.Header.Get("X-Trace")))
	}))
	defer srv.Close()

	client := &http.Client{
		Timeout:   2 * time.Second,
		Transport: headerRoundTripper{key: "X-Trace", value: "abc-123"},
	}
	got, err := fetch(client, srv.URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got != "abc-123" {
		t.Errorf("en-tête propagé = %q, voulu %q", got, "abc-123")
	}
}

// TestContextCancellation : une requête annulée via son contexte échoue
// immédiatement, sans attendre la réponse d'un serveur lent.
func TestContextCancellation(t *testing.T) {
	// Serveur volontairement lent.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(time.Second):
			w.Write([]byte("trop tard"))
		case <-r.Context().Done(): // le client a abandonné
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // déjà annulé : la requête doit échouer tout de suite

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = http.DefaultClient.Do(req)
	if err == nil {
		t.Fatal("erreur attendue (contexte annulé), obtenu nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("erreur = %v, attendu une annulation de contexte", err)
	}
}

// --- httptest.NewTestServer et url.Clone (🆕 Go 1.27) ---

// TestInMemoryServer : la variante 1.27 de TestClientAgainstServer.
//
// Trois différences avec httptest.NewServer :
//   - aucun socket n'est ouvert, donc aucun port à réserver — les tests
//     parallèles ne peuvent plus entrer en collision ni buter sur un quota de
//     descripteurs de fichiers ;
//   - pas de defer Close() : le serveur est lié au testing.TB ;
//   - le transport étant purement en mémoire, il vit DANS une bulle
//     testing/synctest — ce qu'un vrai socket interdisait.
//
// ⚠️ srv.URL vaut "http://example.com", SANS port : il n'y a pas d'adresse
// réelle. Seul le client renvoyé par srv.Client() sait joindre cet hôte fictif ;
// un http.Client construit à la main échouera.
func TestInMemoryServer(t *testing.T) {
	srv := httptest.NewTestServer(t, newMux())

	got, err := fetch(srv.Client(), srv.URL+"/items/99")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got != "item 99" {
		t.Errorf("réponse = %q, voulu %q", got, "item 99")
	}
	if strings.Contains(srv.URL, "127.0.0.1") {
		t.Errorf("URL = %q ; un serveur en mémoire n'a pas d'adresse de loopback", srv.URL)
	}
}

// TestURLClone couvre url.URL.Clone() et url.Values.Clone() (🆕 1.27).
//
// Le besoin est réel dans un RoundTripper ou un middleware : muter l'URL de la
// requête reçue est un effet de bord partagé avec l'appelant. Avant 1.27, une
// « copie » par affectation restait superficielle — le champ User et la map de
// Values continuaient d'être partagés.
func TestURLClone(t *testing.T) {
	u, err := url.Parse("https://api.example.com/v1/items?page=1&tri=nom")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}

	c := u.Clone()
	c.Host = "staging.example.com"
	c.Path = "/v2/items"

	if u.Host != "api.example.com" || u.Path != "/v1/items" {
		t.Errorf("l'original a été muté : %s", u)
	}

	// Values.Clone : la map sous-jacente est bien dupliquée.
	q := u.Query()
	q2 := q.Clone()
	q2.Set("page", "2")
	q2.Add("filtre", "actif")

	if q.Get("page") != "1" {
		t.Errorf("q[page] = %q ; l'original ne devait pas bouger", q.Get("page"))
	}
	if q.Has("filtre") {
		t.Error("la clé ajoutée au clone est apparue dans l'original")
	}
	if q2.Get("page") != "2" || q2.Get("filtre") != "actif" {
		t.Errorf("le clone n'a pas reçu les modifications : %v", q2)
	}
}
