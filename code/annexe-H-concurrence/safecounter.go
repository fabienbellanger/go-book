// Package main rassemble les patterns SÛRS de l'Annexe H : compteurs protégés,
// transfert sans interblocage (ordre de verrouillage global) et cession de
// propriété par canal. Tout le paquet passe « go test -race » sans avertissement.
package main

import (
	"sync"
	"sync/atomic"
)

// Counter est un compteur sûr en accès concurrent : le mutex garantit qu'un seul
// incrément s'exécute à la fois (exclusion mutuelle). Sans ce verrou, n++ est une
// data race (lecture + écriture concurrentes non synchronisées).
type Counter struct {
	mu sync.Mutex
	n  int
}

// Inc incrémente le compteur de façon atomique vis-à-vis des autres goroutines.
func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock() // defer : libère le verrou même en cas de panique ou de retour anticipé
	c.n++
}

// Value lit la valeur courante. La LECTURE doit elle aussi être verrouillée :
// lire pendant qu'une autre goroutine écrit est déjà une course.
func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// AtomicCounter fait la même chose sans verrou, pour le cas simple d'un entier :
// les opérations atomiques sont indivisibles par construction (🔁 Ch. 21).
type AtomicCounter struct {
	n atomic.Int64
}

// Inc ajoute 1 atomiquement.
func (c *AtomicCounter) Inc() { c.n.Add(1) }

// Value lit la valeur atomiquement.
func (c *AtomicCounter) Value() int64 { return c.n.Load() }
