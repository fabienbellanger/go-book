package main

import (
	"expvar"
	"runtime"
	"sync"
	"testing"
)

// Un instantané « au repos » a des valeurs cohérentes.
func TestSnapshotSane(t *testing.T) {
	s := ReadSnapshot()
	if s.Goroutines < 1 {
		t.Errorf("Goroutines = %d ; attendu >= 1", s.Goroutines)
	}
	// La métrique /sched compte AUSSI les goroutines système : elle est >= NumGoroutine.
	if s.GoroutinesAll < uint64(s.Goroutines) {
		t.Errorf("GoroutinesAll (%d) < Goroutines (%d) : incohérent", s.GoroutinesAll, s.Goroutines)
	}
	if s.GOMAXPROCS < 1 {
		t.Errorf("GOMAXPROCS = %d ; attendu >= 1", s.GOMAXPROCS)
	}
	if s.GoVersion == "" {
		t.Error("GoVersion vide ; ReadBuildInfo a échoué")
	}
}

// Lancer des goroutines fait monter le compteur ; les libérer le fait redescendre.
func TestGoroutinesCounted(t *testing.T) {
	base := ReadSnapshot().Goroutines

	block := make(chan struct{})
	var wg sync.WaitGroup
	const n = 50
	for range n {
		wg.Go(func() { <-block })
	}
	// Attendre qu'elles soient toutes démarrées.
	for runtime.NumGoroutine() < base+n {
		runtime.Gosched()
	}

	during := ReadSnapshot()
	if during.Goroutines < base+n {
		t.Errorf("pendant : %d goroutines ; attendu >= %d", during.Goroutines, base+n)
	}
	if during.GoroutinesCreated == 0 {
		t.Error("GoroutinesCreated devrait être un cumul non nul")
	}

	close(block)
	wg.Wait()
}

// Le compteur expvar est réellement publié sur /debug/vars (accessible par Get).
func TestExpvarPublished(t *testing.T) {
	start := RequestsServed()
	RecordRequest()
	RecordRequest()
	if got := RequestsServed(); got != start+2 {
		t.Errorf("RequestsServed = %d ; attendu %d", got, start+2)
	}
	// expvar.Get retrouve la variable par son nom.
	if v := expvar.Get("requests_served"); v == nil {
		t.Error("requests_served n'est pas publié dans expvar")
	}
	if v := expvar.Get("goroutines_live"); v == nil {
		t.Error("goroutines_live (jauge) n'est pas publié dans expvar")
	}
}

// L'ancienne API ReadMemStats reste cohérente avec runtime/metrics (même ordre
// de grandeur pour le tas vivant).
func TestLegacyHeapAllocPositive(t *testing.T) {
	if LegacyHeapAlloc() == 0 {
		t.Error("HeapAlloc = 0 ; un programme vivant a forcément du tas")
	}
}
