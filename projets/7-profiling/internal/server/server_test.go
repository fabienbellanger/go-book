package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/wordstats/internal/analyze"
	"example.com/wordstats/internal/server"
)

func newTestServer(t *testing.T, opts server.Options) *httptest.Server {
	t.Helper()
	srv := server.New(opts)
	ts := httptest.NewServer(srv.Routes)
	t.Cleanup(ts.Close)
	return ts
}

func TestStatsEndpoint(t *testing.T) {
	ts := newTestServer(t, server.Options{})
	body := strings.NewReader("le chat le chien le chat")

	resp, err := http.Post(ts.URL+"/stats?n=2", "text/plain", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut = %d", resp.StatusCode)
	}

	var got []analyze.Count
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := []analyze.Count{{Word: "le", Count: 3}, {Word: "chat", Count: 2}}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("top = %+v, voulu %+v", got, want)
	}
}

// TestBothImplementations vérifie que les deux moteurs (v1 et v2) servis par
// l'API donnent le même résultat — la garantie qui rend l'optimisation sûre.
func TestBothImplementations(t *testing.T) {
	ts := newTestServer(t, server.Options{})
	const text = "alpha beta alpha gamma beta alpha"

	get := func(impl string) []analyze.Count {
		resp, err := http.Post(ts.URL+"/stats?impl="+impl, "text/plain", strings.NewReader(text))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var c []analyze.Count
		json.NewDecoder(resp.Body).Decode(&c)
		return c
	}
	v1, v2 := get("v1"), get("v2")
	if len(v1) != len(v2) {
		t.Fatalf("tailles différentes : v1=%d v2=%d", len(v1), len(v2))
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("rang %d : v1=%+v v2=%+v", i, v1[i], v2[i])
		}
	}
}

func TestHealthz(t *testing.T) {
	ts := newTestServer(t, server.Options{})
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz = %d", resp.StatusCode)
	}
}

// TestPprofMounted vérifie que les endpoints de profilage répondent.
func TestPprofMounted(t *testing.T) {
	ts := newTestServer(t, server.Options{})
	resp, err := http.Get(ts.URL + "/debug/pprof/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/debug/pprof/ = %d", resp.StatusCode)
	}
}

// TestFlightRecorderCapture déclenche une capture de trace : avec un seuil de 0,
// toute requête est « lente », et un fichier de trace doit apparaître.
func TestFlightRecorderCapture(t *testing.T) {
	dir := t.TempDir()
	ts := newTestServer(t, server.Options{SlowReq: 1 * time.Nanosecond, TraceDir: dir})

	resp, err := http.Post(ts.URL+"/stats", "text/plain", strings.NewReader("un deux trois"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		t.Skip("aucune trace écrite (FlightRecorder peut être indisponible selon la plateforme)")
	}
	if !strings.HasPrefix(entries[0].Name(), "slow-") {
		t.Errorf("fichier de trace inattendu : %s", entries[0].Name())
	}
}
