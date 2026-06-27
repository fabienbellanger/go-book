package main

import (
	"fmt"
	"strings"
	"sync"
)

// WordCount compte les occurrences de chaque mot. La préallocation make(map, n)
// dimensionne la table d'emblée et évite les croissances/évacuations successives.
func WordCount(words []string) map[string]int {
	counts := make(map[string]int, len(words))
	for _, w := range words {
		counts[w]++ // lecture-incrément : 0 par défaut si absent
	}
	return counts
}

// IterationOrders renvoie n parcours de la MÊME map, chacun sérialisé en chaîne. Go
// randomise le point de départ de chaque `range` : ces parcours diffèrent en général,
// ce qui empêche tout code de dépendre d'un ordre (et constitue un durcissement).
func IterationOrders(m map[int]int, n int) []string {
	orders := make([]string, n)
	for i := range n {
		var b strings.Builder
		for k := range m {
			fmt.Fprintf(&b, "%d,", k)
		}
		orders[i] = b.String()
	}
	return orders
}

// SafeCounter protège une map par un Mutex. Les maps Go ne sont PAS sûres en accès
// concurrent : deux écritures simultanées déclenchent « fatal error: concurrent map
// writes » (non rattrapable). Le verrou sérialise les accès.
type SafeCounter struct {
	mu sync.Mutex
	m  map[string]int
}

func NewSafeCounter() *SafeCounter {
	return &SafeCounter{m: make(map[string]int)}
}

func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	c.m[key]++
	c.mu.Unlock()
}

func (c *SafeCounter) Get(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[key]
}

func (c *SafeCounter) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.m)
}
