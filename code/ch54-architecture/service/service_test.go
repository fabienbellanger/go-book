package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"example.com/gobook/ch54-architecture/domain"
)

// fakeStore est une implémentation factice de NoteStore, entièrement en mémoire
// et sans I/O. Elle prouve le bénéfice des frontières par interfaces : le
// service se teste SANS base de données réelle, et on peut forcer une erreur.
type fakeStore struct {
	created []domain.Note
	getErr  error
}

func (f *fakeStore) Create(_ context.Context, title, body string) (domain.Note, error) {
	n := domain.Note{ID: "fake1", Title: title, Body: body}
	f.created = append(f.created, n)
	return n, nil
}

func (f *fakeStore) Get(_ context.Context, id string) (domain.Note, error) {
	if f.getErr != nil {
		return domain.Note{}, f.getErr
	}
	return domain.Note{ID: id, Title: "x"}, nil
}

func (f *fakeStore) List(_ context.Context) ([]domain.Note, error) {
	return f.created, nil
}

// newTestService branche un fake et un logger silencieux (io.Discard).
func newTestService(fake NoteStore) *Service {
	return New(fake, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestServiceCreateTrimsAndPersists(t *testing.T) {
	fake := &fakeStore{}
	svc := newTestService(fake)

	n, err := svc.Create(context.Background(), "  Courses  ", "lait")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n.Title != "Courses" {
		t.Errorf("titre non nettoyé : %q", n.Title)
	}
	if len(fake.created) != 1 {
		t.Fatalf("attendu 1 note persistée, obtenu %d", len(fake.created))
	}
}

func TestServiceCreateRejectsEmptyTitle(t *testing.T) {
	fake := &fakeStore{}
	svc := newTestService(fake)

	_, err := svc.Create(context.Background(), "   ", "corps")
	if !errors.Is(err, ErrEmptyTitle) {
		t.Fatalf("attendu ErrEmptyTitle, obtenu %v", err)
	}
	if len(fake.created) != 0 {
		t.Errorf("aucune note ne devait être persistée, obtenu %d", len(fake.created))
	}
}

func TestServiceGetPropagatesNotFound(t *testing.T) {
	fake := &fakeStore{getErr: domain.ErrNotFound}
	svc := newTestService(fake)

	_, err := svc.Get(context.Background(), "n1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("l'erreur du store doit rester reconnaissable (errors.Is), obtenu %v", err)
	}
}
