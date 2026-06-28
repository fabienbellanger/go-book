package store

import (
	"context"
	"errors"
	"testing"
)

func TestMemStoreCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()

	created, err := s.Create(ctx, TaskInput{Title: "  écrire le chapitre  "})
	if err != nil {
		t.Fatalf("Create : %v", err)
	}
	if created.ID != 1 {
		t.Errorf("ID = %d, voulu 1", created.ID)
	}
	if created.Title != "écrire le chapitre" {
		t.Errorf("Title = %q, le titre doit être nettoyé (TrimSpace)", created.Title)
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt ne doit pas être nul")
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get : %v", err)
	}
	if got != created {
		t.Errorf("Get = %+v, voulu %+v", got, created)
	}

	updated, err := s.Update(ctx, created.ID, TaskInput{Title: "relire", Done: true})
	if err != nil {
		t.Fatalf("Update : %v", err)
	}
	if !updated.Done || updated.Title != "relire" {
		t.Errorf("Update = %+v, modification non appliquée", updated)
	}
	if updated.ID != created.ID || !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Error("Update ne doit changer ni l'ID ni la date de création")
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete : %v", err)
	}
	if _, err := s.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("après suppression, Get = %v, voulu ErrNotFound", err)
	}
}

func TestMemStoreNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()

	if _, err := s.Get(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get inexistant = %v, voulu ErrNotFound", err)
	}
	if _, err := s.Update(ctx, 99, TaskInput{Title: "x"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update inexistant = %v, voulu ErrNotFound", err)
	}
	if err := s.Delete(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete inexistant = %v, voulu ErrNotFound", err)
	}
}

func TestMemStoreListFilterAndPaginate(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()
	// 5 tâches : les paires marquées « done ».
	for i := range 5 {
		in := TaskInput{Title: "t", Done: i%2 == 0}
		if _, err := s.Create(ctx, in); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name    string
		filter  ListFilter
		wantIDs []int64
	}{
		{"toutes", ListFilter{}, []int64{1, 2, 3, 4, 5}},
		{"done=true", ListFilter{Done: new(true)}, []int64{1, 3, 5}},
		{"done=false", ListFilter{Done: new(false)}, []int64{2, 4}},
		{"pagination", ListFilter{Limit: 2, Offset: 1}, []int64{2, 3}},
		{"offset hors borne", ListFilter{Offset: 100}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.List(ctx, tt.filter)
			if err != nil {
				t.Fatalf("List : %v", err)
			}
			var ids []int64
			for _, task := range got {
				ids = append(ids, task.ID)
			}
			if !slicesEqual(ids, tt.wantIDs) {
				t.Errorf("IDs = %v, voulu %v", ids, tt.wantIDs)
			}
		})
	}
}

func TestMemStoreContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // contexte déjà annulé

	s := NewMemStore()
	if _, err := s.Create(ctx, TaskInput{Title: "x"}); !errors.Is(err, context.Canceled) {
		t.Errorf("Create = %v, voulu context.Canceled", err)
	}
}

func slicesEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
