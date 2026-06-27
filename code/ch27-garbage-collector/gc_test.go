package main

import (
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

// Une référence faible n'empêche pas la collecte : dès que la dernière référence
// FORTE disparaît, le GC récupère l'objet et Get renvoie nil.
func TestWeakCacheCollects(t *testing.T) {
	c := NewCache()

	// Portée isolée : après ce bloc, plus aucune référence forte vers r.
	func() {
		r := &Resource{ID: 1}
		c.Put(r)
		if c.Get(1) == nil {
			t.Fatal("Get devrait trouver la ressource tant que r est vivant")
		}
		runtime.KeepAlive(r)
	}()

	runtime.GC()
	runtime.GC()
	if c.Get(1) != nil {
		t.Error("la ressource aurait dû être collectée (plus de référence forte)")
	}
}

// runtime.AddCleanup exécute le nettoyage une fois l'objet inatteignable.
func TestAddCleanupRuns(t *testing.T) {
	done := make(chan int, 1)
	func() {
		r := &Resource{ID: 7}
		WithCleanup(r, done)
		runtime.KeepAlive(r)
	}()

	runtime.GC()
	select {
	case id := <-done:
		if id != 7 {
			t.Errorf("cleanup reçu id=%d ; attendu 7", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("le cleanup ne s'est pas exécuté dans le délai")
	}
}

// SetGCPercent renvoie l'ancienne valeur : on peut sauvegarder/restaurer le réglage.
func TestWithGCPercentRestores(t *testing.T) {
	before := debug.SetGCPercent(-1) // lit la valeur (et désactive un instant)
	debug.SetGCPercent(before)       // restaure aussitôt

	ran := false
	WithGCPercent(42, func() { ran = true })
	if !ran {
		t.Fatal("la fonction n'a pas été exécutée")
	}
	if after := debug.SetGCPercent(before); after != before {
		t.Errorf("GOGC non restauré : %d ; attendu %d", after, before)
	}
}
