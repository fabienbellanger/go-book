package main

import (
	"sync"
	"testing"
)

// Mutex : 1000 incréments concurrents donnent exactement 1000 (pas de course).
func TestSafeCounter(t *testing.T) {
	var c SafeCounter
	runConcurrently(1000, c.Inc)
	if got := c.Value(); got != 1000 {
		t.Errorf("SafeCounter = %d ; attendu 1000", got)
	}
}

// Atomic : même garantie, sans verrou.
func TestAtomicCounter(t *testing.T) {
	var c AtomicCounter
	runConcurrently(1000, c.Inc)
	if got := c.Value(); got != 1000 {
		t.Errorf("AtomicCounter = %d ; attendu 1000", got)
	}
}

// OnceValue : la fonction sous-jacente ne s'exécute qu'une fois, même sous
// 100 appels concurrents, et renvoie toujours la même valeur.
func TestOnceValueRunsOnce(t *testing.T) {
	runConcurrently(100, func() { _ = config() })
	if got := loadCount.Load(); got != 1 {
		t.Errorf("expensiveInit exécuté %d fois ; attendu 1", got)
	}
	if v, ok := config()["answer"]; !ok || v != 42 {
		t.Errorf("config()[answer] = %d, %v ; attendu 42, true", v, ok)
	}
}

// RWMutex : lectures et écritures concurrentes restent cohérentes et sans course.
func TestRegistryConcurrent(t *testing.T) {
	reg := NewRegistry()
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() { reg.Set("k", i) }) // écrivains
		wg.Go(func() { reg.Get("k") })    // lecteurs
	}
	wg.Wait()
	if _, ok := reg.Get("k"); !ok {
		t.Error("la clé devrait exister après les écritures")
	}
}

// atomic.Pointer : Store publie une nouvelle version, lue atomiquement.
func TestConfigPointerSwap(t *testing.T) {
	cfg := NewConfig(&Settings{Level: 1})
	runConcurrently(50, func() { _ = cfg.Load() }) // lecteurs sans verrou
	cfg.Store(&Settings{Level: 2, Verbose: true})
	if s := cfg.Load(); s.Level != 2 || !s.Verbose {
		t.Errorf("Settings = %+v ; attendu {Verbose:true Level:2}", *s)
	}
}

// sync.Pool : joinInts est correct et sûr en concurrence (buffers recyclés).
func TestJoinIntsConcurrent(t *testing.T) {
	if got := joinInts([]int{1, 2, 3}, "-"); got != "1-2-3" {
		t.Errorf("joinInts = %q ; attendu \"1-2-3\"", got)
	}
	runConcurrently(200, func() {
		if got := joinInts([]int{4, 5}, ","); got != "4,5" {
			t.Errorf("joinInts concurrent = %q ; attendu \"4,5\"", got)
		}
	})
}
