// Démonstrations du chapitre 21 : primitives de synchronisation.
// Lancement : depuis code/, `go run ./ch21-synchronisation`
package main

import (
	"sync"
	"sync/atomic"
)

// SafeCounter protège un entier par un mutex : un seul appelant à la fois entre
// dans la section critique (Lock..Unlock). defer Unlock garantit la libération,
// même si le corps panique.
type SafeCounter struct {
	mu sync.Mutex
	n  int64
}

func (c *SafeCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *SafeCounter) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// AtomicCounter fait la même chose SANS verrou : atomic.Int64 rend l'incrément
// indivisible au niveau matériel. Plus simple et plus rapide quand l'état tient
// en un seul mot machine.
type AtomicCounter struct {
	n atomic.Int64
}

func (c *AtomicCounter) Inc()         { c.n.Add(1) }
func (c *AtomicCounter) Value() int64 { return c.n.Load() }

// runConcurrently lance fn n fois en parallèle et attend la fin. WaitGroup.Go
// (Go 1.25) fusionne Add + go + Done : plus court, et impossible de mal placer
// le Add — `go vet` signale justement un Add appelé depuis la goroutine.
func runConcurrently(n int, fn func()) {
	var wg sync.WaitGroup
	for range n {
		wg.Go(fn)
	}
	wg.Wait()
}
