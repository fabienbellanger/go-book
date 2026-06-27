package main

import (
	"sync"
	"sync/atomic"
)

// --- Initialisation paresseuse, une seule fois (sync.OnceValue, Go 1.21) ---

// loadCount compte les exécutions RÉELLES de expensiveInit, pour prouver qu'il
// n'a lieu qu'une fois quel que soit le nombre d'appelants concurrents.
var loadCount atomic.Int64

func expensiveInit() map[string]int {
	loadCount.Add(1)
	return map[string]int{"answer": 42}
}

// config renvoie toujours la MÊME valeur, calculée à la première demande
// seulement. OnceValue remplace l'ancien trio var globale + sync.Once + test nil.
var config = sync.OnceValue(expensiveInit)

// --- Cache lecture-intensif protégé par RWMutex ---

// Registry est lu bien plus souvent qu'écrit. RWMutex laisse les lecteurs entrer
// en PARALLÈLE (RLock) et n'exclut tout le monde que pour une écriture (Lock).
type Registry struct {
	mu sync.RWMutex
	m  map[string]int
}

func NewRegistry() *Registry { return &Registry{m: map[string]int{}} }

func (r *Registry) Get(key string) (int, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.m[key]
	return v, ok
}

func (r *Registry) Set(key string, val int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[key] = val
}

// --- Échange d'état SANS verrou (atomic.Pointer[T]) ---

// Settings est une configuration immuable qu'on remplace en bloc.
type Settings struct {
	Verbose bool
	Level   int
}

// Config publie une *Settings via atomic.Pointer : les lecteurs obtiennent le
// pointeur courant sans jamais bloquer, et un écrivain publie une nouvelle
// version d'un seul Store atomique. Idéal pour une config rechargée à chaud.
type Config struct {
	current atomic.Pointer[Settings]
}

func NewConfig(s *Settings) *Config {
	c := &Config{}
	c.current.Store(s)
	return c
}

func (c *Config) Load() *Settings   { return c.current.Load() }
func (c *Config) Store(s *Settings) { c.current.Store(s) }
