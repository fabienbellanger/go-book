package server

import (
	"bytes"
	"sync"
)

// kvStore est un magasin clé-valeur en mémoire, sûr pour un usage concurrent.
//
// Un RWMutex sépare lectures (Get) et écritures (Set/Delete) : plusieurs Get
// peuvent s'exécuter en parallèle, seules les écritures sont exclusives.
type kvStore struct {
	mu sync.RWMutex
	m  map[string][]byte
}

func newKVStore() *kvStore {
	return &kvStore{m: make(map[string][]byte)}
}

// Get renvoie une *copie* de la valeur : sans cela, l'appelant partagerait la
// tranche stockée et une mutation ultérieure provoquerait une course.
func (s *kvStore) Get(key string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key]
	if !ok {
		return nil, false
	}
	return bytes.Clone(v), true
}

// Set stocke une copie de la valeur fournie.
func (s *kvStore) Set(key string, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = bytes.Clone(value)
}

// Delete retire une clé et indique si elle existait.
func (s *kvStore) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[key]; !ok {
		return false
	}
	delete(s.m, key)
	return true
}
