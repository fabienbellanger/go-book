// Package store définit le modèle de données (Task) et l'abstraction de
// persistance (Store) de l'API.
//
// Deux implémentations cohabitent derrière la même interface :
//   - MemStore : en mémoire, sans aucune dépendance — backend par défaut et
//     support des tests ;
//   - SQLStore : adossé à database/sql + migrations, pour une vraie base
//     relationnelle.
//
// Les handlers HTTP ne dépendent que de l'interface Store : changer de backend
// ne touche pas une ligne du code web (inversion de dépendance, voir Ch. 9).
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNotFound est renvoyée quand aucune tâche ne correspond à l'identifiant.
// La couche HTTP la traduit en 404 (voir internal/api).
var ErrNotFound = errors.New("tâche introuvable")

// MaxTitleLen borne la longueur d'un titre (validation métier).
const MaxTitleLen = 200

// Task est une tâche à faire. Les tags JSON fixent le contrat de l'API.
type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskInput porte les champs qu'un client peut fournir à la création ou à la
// mise à jour. On le sépare de Task pour que l'API ne laisse jamais le client
// imposer un ID ou une date de création.
type TaskInput struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// Validate applique les règles métier sur une entrée. La couche HTTP la
// traduit en 422 (entité non traitable).
func (in TaskInput) Validate() error {
	title := strings.TrimSpace(in.Title)
	switch {
	case title == "":
		return errors.New("le titre est obligatoire")
	case len(title) > MaxTitleLen:
		return fmt.Errorf("le titre dépasse %d caractères", MaxTitleLen)
	}
	return nil
}

// ListFilter filtre et pagine la liste des tâches.
type ListFilter struct {
	Done   *bool // nil = toutes ; sinon ne garde que les tâches dans cet état
	Limit  int   // 0 = limite par défaut
	Offset int   // décalage (pagination)
}

// Store est le contrat de persistance. Chaque méthode reçoit un context.Context
// pour propager annulation et délai (Ch. 22) jusqu'au backend.
type Store interface {
	Create(ctx context.Context, in TaskInput) (Task, error)
	Get(ctx context.Context, id int64) (Task, error)
	List(ctx context.Context, f ListFilter) ([]Task, error)
	Update(ctx context.Context, id int64, in TaskInput) (Task, error)
	Delete(ctx context.Context, id int64) error
}
