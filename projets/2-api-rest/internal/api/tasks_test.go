package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/tasksapi/internal/store"
)

// newTestServer monte un serveur sur un MemStore, sans protection CSRF (nil) et
// avec un logger muet (io.Discard) pour ne pas polluer la sortie des tests.
func newTestServer() *Server {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(store.NewMemStore(), log, nil)
}

// do exécute une requête contre le serveur et renvoie la réponse enregistrée.
func do(t *testing.T, srv *Server, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, r)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestCreateAndGet(t *testing.T) {
	srv := newTestServer()

	rec := do(t, srv, http.MethodPost, "/api/tasks", `{"title":"acheter du café"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST code = %d, voulu 201 (corps : %s)", rec.Code, rec.Body)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/tasks/1" {
		t.Errorf("Location = %q, voulu /api/tasks/1", loc)
	}

	var created store.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("réponse JSON invalide : %v", err)
	}
	if created.ID != 1 || created.Title != "acheter du café" || created.Done {
		t.Errorf("tâche créée inattendue : %+v", created)
	}

	rec = do(t, srv, http.MethodGet, "/api/tasks/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET code = %d, voulu 200", rec.Code)
	}
}

func TestListWithFilter(t *testing.T) {
	srv := newTestServer()
	do(t, srv, http.MethodPost, "/api/tasks", `{"title":"a","done":true}`)
	do(t, srv, http.MethodPost, "/api/tasks", `{"title":"b","done":false}`)

	rec := do(t, srv, http.MethodGet, "/api/tasks?done=true", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, voulu 200", rec.Code)
	}
	var resp listResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 || len(resp.Tasks) != 1 || resp.Tasks[0].Title != "a" {
		t.Errorf("liste filtrée inattendue : %+v", resp)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	srv := newTestServer()
	do(t, srv, http.MethodPost, "/api/tasks", `{"title":"brouillon"}`)

	rec := do(t, srv, http.MethodPut, "/api/tasks/1", `{"title":"final","done":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT code = %d, voulu 200 (corps : %s)", rec.Code, rec.Body)
	}
	var updated store.Task
	json.Unmarshal(rec.Body.Bytes(), &updated)
	if !updated.Done || updated.Title != "final" {
		t.Errorf("mise à jour non appliquée : %+v", updated)
	}

	rec = do(t, srv, http.MethodDelete, "/api/tasks/1", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE code = %d, voulu 204", rec.Code)
	}
	rec = do(t, srv, http.MethodGet, "/api/tasks/1", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET après suppression = %d, voulu 404", rec.Code)
	}
}

func TestErrorCases(t *testing.T) {
	srv := newTestServer()
	tests := []struct {
		name           string
		method, target string
		body           string
		wantCode       int
	}{
		{"titre vide => 422", http.MethodPost, "/api/tasks", `{"title":"  "}`, http.StatusUnprocessableEntity},
		{"JSON invalide => 400", http.MethodPost, "/api/tasks", `{`, http.StatusBadRequest},
		{"champ inconnu => 400", http.MethodPost, "/api/tasks", `{"title":"x","bidon":1}`, http.StatusBadRequest},
		{"id non entier => 400", http.MethodGet, "/api/tasks/abc", "", http.StatusBadRequest},
		{"tâche absente => 404", http.MethodGet, "/api/tasks/999", "", http.StatusNotFound},
		{"mauvaise méthode => 405", http.MethodDelete, "/api/tasks", "", http.StatusMethodNotAllowed},
		{"route inconnue => 404", http.MethodGet, "/nope", "", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, srv, tt.method, tt.target, tt.body)
			if rec.Code != tt.wantCode {
				t.Errorf("code = %d, voulu %d (corps : %s)", rec.Code, tt.wantCode, rec.Body)
			}
		})
	}
}

// TestCSRFProtection vérifie que la protection cross-origin (Go 1.25) bloque un
// POST déclaré « cross-site » par le navigateur, mais laisse passer un GET.
func TestCSRFProtection(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(store.NewMemStore(), log, http.NewCrossOriginProtection())

	// POST cross-site : rejeté (403) avant d'atteindre le handler.
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST cross-site = %d, voulu 403", rec.Code)
	}

	// GET (méthode sûre) : toujours autorisé, même cross-site.
	req = httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET cross-site = %d, voulu 200", rec.Code)
	}
}
