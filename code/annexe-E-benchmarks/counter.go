package main

import (
	"sync"
	"sync/atomic"
)

// Démonstration « mutex vs atomic » pour un compteur partagé.
//
// Les deux protègent un entier contre les accès concurrents, mais à des coûts
// très différents sous contention (voir Ch. 21).

// mutexCounter protège son entier par un verrou d'exclusion mutuelle.
type mutexCounter struct {
	mu sync.Mutex
	n  int64
}

func (c *mutexCounter) Inc() {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
}

func (c *mutexCounter) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// atomicCounter utilise une instruction atomique du processeur : pas de verrou,
// pas de mise en sommeil de la goroutine.
type atomicCounter struct {
	n atomic.Int64
}

func (c *atomicCounter) Inc()         { c.n.Add(1) }
func (c *atomicCounter) Value() int64 { return c.n.Load() }
