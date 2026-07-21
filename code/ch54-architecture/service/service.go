// Package service porte la logique applicative. Il ne connaît NI la base de
// données, NI le protocole d'entrée : il dépend seulement d'une petite interface
// (NoteStore) qu'il définit lui-même, et de types domain.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"example.com/gobook/ch54-architecture/domain"
)

// ErrEmptyTitle est renvoyée quand on tente de créer une note sans titre.
var ErrEmptyTitle = errors.New("le titre est vide")

// NoteStore est l'interface définie CÔTÉ CONSOMMATEUR : le service déclare
// exactement les opérations dont il a besoin, ni plus ni moins. Un fake de test
// comme une implémentation SQL la satisfont sans rien déclarer (🔁 Ch. 09).
type NoteStore interface {
	Create(ctx context.Context, title, body string) (domain.Note, error)
	Get(ctx context.Context, id string) (domain.Note, error)
	List(ctx context.Context) ([]domain.Note, error)
}

// Service applique les règles métier puis délègue la persistance au store.
type Service struct {
	store NoteStore
	log   *slog.Logger
}

// New construit un Service. Les dépendances (store, logger) sont INJECTÉES par
// le constructeur : aucun accès à un état global, tout est explicite et donc
// remplaçable en test. On renvoie *Service (un struct), on accepte NoteStore
// (une interface).
func New(store NoteStore, log *slog.Logger) *Service {
	return &Service{store: store, log: log}
}

// Create valide l'entrée puis persiste la note. La génération de l'identifiant
// est une préoccupation de persistance : elle revient au store.
func (s *Service) Create(ctx context.Context, title, body string) (domain.Note, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Note{}, ErrEmptyTitle // règle métier, pas une erreur du store
	}
	n, err := s.store.Create(ctx, title, body)
	if err != nil {
		// On ENVELOPPE l'erreur du store et on la renvoie. On ne la logge PAS
		// ici : logger ET renvoyer, c'est la journaliser deux fois. L'appelant
		// qui a le contexte complet décide (🔁 Ch. 10).
		return domain.Note{}, fmt.Errorf("service.Create: %w", err)
	}
	s.log.InfoContext(ctx, "note créée", "id", n.ID)
	return n, nil
}

// Get relit une note et propage telle quelle domain.ErrNotFound (via %w), pour
// que l'appelant puisse la reconnaître avec errors.Is.
func (s *Service) Get(ctx context.Context, id string) (domain.Note, error) {
	n, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Note{}, fmt.Errorf("service.Get %q: %w", id, err)
	}
	return n, nil
}

// List renvoie toutes les notes, en déléguant la lecture au store. Avec Create et
// Get, le service utilise ainsi les TROIS opérations qu'il déclare dans NoteStore :
// l'interface reste « ni plus, ni moins » que ce dont le service a besoin.
func (s *Service) List(ctx context.Context) ([]domain.Note, error) {
	notes, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("service.List: %w", err)
	}
	return notes, nil
}
