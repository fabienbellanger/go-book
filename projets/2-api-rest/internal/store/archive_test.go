package store

import (
	"context"
	"testing"
)

// TestArchiveTaskCommit vérifie le cas nominal : la tâche quitte « tasks » et
// atterrit dans l'archive, sans rien laisser derrière.
func TestArchiveTaskCommit(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()
	created, err := s.Create(ctx, TaskInput{Title: "à archiver"})
	if err != nil {
		t.Fatalf("Create : %v", err)
	}

	if err := s.ArchiveTask(ctx, created.ID); err != nil {
		t.Fatalf("ArchiveTask : %v", err)
	}

	// Plus dans la table active…
	if _, err := s.Get(ctx, created.ID); err != ErrNotFound {
		t.Errorf("Get après archivage = %v, voulu ErrNotFound", err)
	}
	// …mais bien dans l'archive.
	if _, ok := s.archived[created.ID]; !ok {
		t.Errorf("tâche absente de l'archive")
	}
}

// TestArchiveTaskAtomicOnFailure vérifie que l'échec ne laisse aucun état
// intermédiaire : archiver un ID inconnu (ou déjà archivé) renvoie ErrNotFound
// et ne modifie ni la table active ni l'archive. C'est la propriété « tout ou
// rien » : soit l'opération aboutit entièrement, soit rien ne bouge.
func TestArchiveTaskAtomicOnFailure(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()
	keep, _ := s.Create(ctx, TaskInput{Title: "à garder"})
	move, _ := s.Create(ctx, TaskInput{Title: "à déplacer"})

	// Premier archivage : succès.
	if err := s.ArchiveTask(ctx, move.ID); err != nil {
		t.Fatalf("ArchiveTask : %v", err)
	}

	// Ré-archiver le même ID : il n'est plus dans « tasks » -> ErrNotFound.
	if err := s.ArchiveTask(ctx, move.ID); err != ErrNotFound {
		t.Errorf("second ArchiveTask = %v, voulu ErrNotFound", err)
	}
	// Archiver un ID jamais créé -> ErrNotFound.
	if err := s.ArchiveTask(ctx, 9999); err != ErrNotFound {
		t.Errorf("ArchiveTask(inconnu) = %v, voulu ErrNotFound", err)
	}

	// État inchangé par les échecs : la tâche gardée est toujours là, l'archive
	// contient exactement une entrée (pas de doublon, pas de suppression fantôme).
	if _, err := s.Get(ctx, keep.ID); err != nil {
		t.Errorf("tâche conservée introuvable : %v", err)
	}
	if n := len(s.archived); n != 1 {
		t.Errorf("archive contient %d entrées, voulu 1", n)
	}
	if _, ok := s.tasks[move.ID]; ok {
		t.Errorf("la tâche archivée est réapparue dans la table active")
	}
}

// TestArchiveTaskContextCancelled vérifie que l'annulation est respectée avant
// toute mutation (cohérent avec les autres méthodes du MemStore).
func TestArchiveTaskContextCancelled(t *testing.T) {
	s := NewMemStore()
	created, _ := s.Create(context.Background(), TaskInput{Title: "x"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.ArchiveTask(ctx, created.ID); err != context.Canceled {
		t.Fatalf("ArchiveTask sur context annulé = %v, voulu context.Canceled", err)
	}
	// La tâche n'a pas bougé.
	if _, err := s.Get(context.Background(), created.ID); err != nil {
		t.Errorf("tâche modifiée malgré l'annulation : %v", err)
	}
	if len(s.archived) != 0 {
		t.Errorf("archive non vide malgré l'annulation")
	}
}
