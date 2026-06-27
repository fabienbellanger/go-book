package main

import (
	"sync"
	"testing"
)

// La valeur reçue par canal est entièrement publiée : aucun champ « à moitié écrit ».
// Lancé avec -race, ce test reste vert (le canal établit le happens-before).
func TestPublishViaChannel(t *testing.T) {
	for range 100 {
		c := PublishViaChannel()
		if c.Addr != "localhost" || c.Port != 8080 || !c.Ready {
			t.Fatalf("config partiellement visible : %+v", c)
		}
	}
}

// Mille goroutines appellent GetConfig en même temps : toutes obtiennent LE MÊME
// pointeur, et le constructeur n'a tourné qu'une fois. -race confirme l'absence
// de course (sync.Once synchronise lecteurs et écrivain).
func TestGetConfigConcurrent(t *testing.T) {
	const n = 1000
	got := make([]*Config, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() { got[i] = GetConfig() })
	}
	wg.Wait()

	first := got[0]
	if first == nil {
		t.Fatal("GetConfig a renvoyé nil")
	}
	for i, c := range got {
		if c != first {
			t.Fatalf("goroutine %d a obtenu un pointeur différent", i)
		}
	}
}

// atomic.Pointer : un Store « happens-before » le Load qui l'observe. Des lecteurs
// concurrents ne voient jamais un état déchiré. -race reste vert.
func TestAtomicPublish(t *testing.T) {
	SwapConfig(buildConfig())
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			if c := LoadConfig(); c == nil || c.Addr != "localhost" {
				t.Errorf("Load a renvoyé un état invalide : %+v", c)
			}
		})
	}
	wg.Wait()
}
