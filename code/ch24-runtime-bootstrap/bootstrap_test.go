package main

import (
	"runtime"
	"strings"
	"testing"
)

// L'ordre d'init suit les DÉPENDANCES (base avant derived), puis les init()
// dans l'ordre du source — jamais l'inverse.
func TestInitOrder(t *testing.T) {
	order := InitOrder()
	want := []string{"base", "derived(after base)", "init #1", "init #2"}
	if len(order) != len(want) {
		t.Fatalf("InitOrder() = %v ; attendu %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("étape %d = %q ; attendu %q", i, order[i], want[i])
		}
	}
}

// Quoi qu'il arrive, les variables s'initialisent AVANT les init().
func TestVariablesBeforeInitFuncs(t *testing.T) {
	order := strings.Join(InitOrder(), "|")
	base := strings.Index(order, "base")
	firstInit := strings.Index(order, "init #1")
	if base == -1 || firstInit == -1 || base > firstInit {
		t.Errorf("les variables devraient précéder les init() ; trace = %s", order)
	}
}

// CurrentRuntime reflète l'état réel au démarrage.
func TestCurrentRuntime(t *testing.T) {
	info := CurrentRuntime()
	if info.Version != runtime.Version() {
		t.Errorf("Version = %q ; attendu %q", info.Version, runtime.Version())
	}
	if info.NumCPU < 1 {
		t.Errorf("NumCPU = %d ; attendu >= 1", info.NumCPU)
	}
	if info.GOMAXPROCS < 1 {
		t.Errorf("GOMAXPROCS = %d ; attendu >= 1", info.GOMAXPROCS)
	}
}
