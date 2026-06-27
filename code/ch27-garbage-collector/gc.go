// Package main illustre le GC côté API utilisateur : un cache à références
// FAIBLES (weak.Pointer, 1.24) qui n'empêche pas la collecte, un nettoyage de
// ressource via runtime.AddCleanup (1.24), et le réglage du GC (GOGC/GOMEMLIMIT).
package main

import (
	"runtime"
	"runtime/debug"
	"sync"
	"weak"
)

// Resource est un objet « lourd » qu'on aimerait mettre en cache SANS empêcher
// le GC de le récupérer quand plus personne d'autre ne le tient.
type Resource struct {
	ID   int
	data [4096]byte
}

// Cache associe un ID à une référence FAIBLE vers la ressource. Une référence
// faible ne compte pas comme « vivante » : si l'objet n'est plus tenu fortement
// ailleurs, le GC le collecte et wp.Value() renvoie nil.
type Cache struct {
	mu sync.Mutex
	m  map[int]weak.Pointer[Resource]
}

// NewCache crée un cache vide.
func NewCache() *Cache {
	return &Cache{m: make(map[int]weak.Pointer[Resource])}
}

// Put enregistre une référence faible vers r.
func (c *Cache) Put(r *Resource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[r.ID] = weak.Make(r)
}

// Get renvoie la ressource si elle est ENCORE vivante, sinon nil (collectée).
func (c *Cache) Get(id int) *Resource {
	c.mu.Lock()
	defer c.mu.Unlock()
	if wp, ok := c.m[id]; ok {
		return wp.Value() // promotion faible -> forte ; nil si déjà collecté
	}
	return nil
}

// WithCleanup attache un nettoyage exécuté APRÈS que r soit devenu inatteignable.
// Contrairement à un finalizer, AddCleanup ne ressuscite pas l'objet et reçoit
// une valeur séparée (ici un canal pour signaler l'exécution).
func WithCleanup(r *Resource, done chan<- int) {
	runtime.AddCleanup(r, func(id int) { done <- id }, r.ID)
}

// WithGCPercent exécute f avec un GOGC temporaire, puis restaure l'ancien.
// GOGC = % de croissance du tas vivant avant de redéclencher un GC (défaut 100).
func WithGCPercent(pct int, f func()) {
	old := debug.SetGCPercent(pct)
	defer debug.SetGCPercent(old)
	f()
}

// CurrentMemoryLimit lit la limite mémoire douce (GOMEMLIMIT) sans la modifier.
// math.MaxInt64 signifie « aucune limite ».
func CurrentMemoryLimit() int64 {
	return debug.SetMemoryLimit(-1) // -1 = lecture seule
}
