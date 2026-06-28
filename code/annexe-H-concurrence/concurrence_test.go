package main

import (
	"sync"
	"testing"
)

// TestCounterRace lance de nombreux incréments concurrents. À exécuter avec
// « go test -race » : sans le mutex de Counter, le détecteur signalerait la course
// et le total final serait imprévisible.
func TestCounterRace(t *testing.T) {
	const goroutines, perG = 50, 200
	var c Counter
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() { // WaitGroup.Go (Go 1.25)
			for range perG {
				c.Inc()
			}
		})
	}
	wg.Wait()
	if got := c.Value(); got != goroutines*perG {
		t.Errorf("compteur = %d, voulu %d (course ?)", got, goroutines*perG)
	}
}

func TestAtomicCounterRace(t *testing.T) {
	const goroutines, perG = 50, 200
	var c AtomicCounter
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range perG {
				c.Inc()
			}
		})
	}
	wg.Wait()
	if got := c.Value(); got != int64(goroutines*perG) {
		t.Errorf("compteur = %d, voulu %d", got, goroutines*perG)
	}
}

// TestTransferNoDeadlock lance des virements CROISÉS en parallèle (A->B et B->A).
// Avec un ordre de verrouillage naïf (toujours from puis to), ce test pourrait
// se figer (deadlock AB-BA). L'ordre global par id l'en empêche ; on vérifie en
// prime que le total est CONSERVÉ (invariant comptable) — preuve d'absence de course.
func TestTransferNoDeadlock(t *testing.T) {
	a, b := NewAccount(1, 1000), NewAccount(2, 1000)
	const rounds = 1000
	var wg sync.WaitGroup
	wg.Go(func() {
		for range rounds {
			Transfer(a, b, 1)
		}
	})
	wg.Go(func() {
		for range rounds {
			Transfer(b, a, 1)
		}
	})
	wg.Wait()

	if total := a.Balance() + b.Balance(); total != 2000 {
		t.Errorf("total = %d, voulu 2000 (course sur le solde ?)", total)
	}
}

// TestTransferSameAccount vérifie le garde-fou « from == to » : pas de double
// verrouillage du même mutex (qui serait un auto-interblocage), et solde inchangé.
func TestTransferSameAccount(t *testing.T) {
	a := NewAccount(7, 500)
	Transfer(a, a, 100)
	if got := a.Balance(); got != 500 {
		t.Errorf("solde = %d, voulu 500 (un virement vers soi-même est un no-op)", got)
	}
}

// TestOwnership vérifie la cession de propriété par canal : aucune mémoire
// partagée, le range s'arrête à la fermeture.
func TestOwnership(t *testing.T) {
	if got := consume(produce(100)); got != 4950 { // somme 0..99
		t.Errorf("somme = %d, voulu 4950", got)
	}
}
