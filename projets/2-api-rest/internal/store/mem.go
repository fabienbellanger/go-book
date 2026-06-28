package store

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"
)

// defaultLimit et maxLimit encadrent la pagination par défaut.
const (
	defaultLimit = 20
	maxLimit     = 100
)

// MemStore est un Store en mémoire, sûr pour un usage concurrent.
//
// Toutes les opérations sont protégées par un RWMutex : les lectures (Get,
// List) prennent un verrou partagé, les écritures un verrou exclusif. C'est le
// schéma classique « beaucoup de lecteurs, peu d'écrivains » (Ch. 21).
type MemStore struct {
	mu     sync.RWMutex
	tasks  map[int64]Task
	nextID int64
	now    func() time.Time // injectable : les tests figent l'horloge
}

// NewMemStore crée un MemStore vide.
func NewMemStore() *MemStore {
	return &MemStore{
		tasks: make(map[int64]Task),
		now:   time.Now,
	}
}

func (s *MemStore) Create(ctx context.Context, in TaskInput) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	t := Task{
		ID:        s.nextID,
		Title:     strings.TrimSpace(in.Title),
		Done:      in.Done,
		CreatedAt: s.now().UTC(),
	}
	s.tasks[t.ID] = t
	return t, nil
}

func (s *MemStore) Get(ctx context.Context, id int64) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

func (s *MemStore) List(ctx context.Context, f ListFilter) ([]Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	// L'itération d'une map est randomisée (Ch. 32) : on collecte puis on trie
	// par ID pour une sortie déterministe — indispensable à une pagination stable.
	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		if f.Done != nil && t.Done != *f.Done {
			continue
		}
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b Task) int { return int(a.ID - b.ID) })

	return paginate(out, f.Limit, f.Offset), nil
}

func (s *MemStore) Update(ctx context.Context, id int64, in TaskInput) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	// On ne touche ni à l'ID ni à la date de création : ce sont des invariants.
	t.Title = strings.TrimSpace(in.Title)
	t.Done = in.Done
	s.tasks[id] = t
	return t, nil
}

func (s *MemStore) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}

// paginate applique offset puis limit à une tranche déjà triée. La limite est
// normalisée (0 => défaut, > maxLimit => maxLimit) et partagée par les backends.
func paginate[T any](items []T, limit, offset int) []T {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []T{}
	}
	end := min(offset+limit, len(items))
	return items[offset:end]
}
