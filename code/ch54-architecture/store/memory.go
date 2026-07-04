// Package store fournit une implémentation concrète de la persistance. C'est un
// ADAPTATEUR de périphérie : il dépend du cœur (domain), jamais l'inverse. On
// pourrait ajouter à côté un store SQL ou fichier sans toucher au service.
package store

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"

	"example.com/gobook/ch54-architecture/domain"
)

// Mem est un store en mémoire, sûr pour un usage concurrent (🔁 Ch. 21). Le
// contexte est accepté pour respecter le contrat de l'interface (une vraie base
// l'utiliserait pour l'annulation), même si cette implémentation l'ignore.
type Mem struct {
	mu    sync.Mutex
	seq   int
	notes map[string]domain.Note
}

// NewMem renvoie le TYPE CONCRET *Mem, pas une interface : « accepter des
// interfaces, renvoyer des structs » (🔁 Ch. 09). C'est l'appelant (le service,
// via son interface NoteStore) qui décide de ne voir qu'une vue restreinte.
func NewMem() *Mem {
	return &Mem{notes: make(map[string]domain.Note)}
}

func (m *Mem) Create(_ context.Context, title, body string) (domain.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	n := domain.Note{ID: fmt.Sprintf("n%d", m.seq), Title: title, Body: body}
	m.notes[n.ID] = n
	return n, nil
}

func (m *Mem) Get(_ context.Context, id string) (domain.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.notes[id]
	if !ok {
		return domain.Note{}, domain.ErrNotFound
	}
	return n, nil
}

func (m *Mem) List(_ context.Context) ([]domain.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Note, 0, len(m.notes))
	for _, n := range m.notes {
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b domain.Note) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}
