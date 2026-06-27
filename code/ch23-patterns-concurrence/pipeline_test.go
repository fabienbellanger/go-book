package main

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// Pipeline en flux : source -> x2 -> +1. Tout est concurrent, le résultat
// reste ordonné car un pipeline préserve l'ordre.
func TestPipeline(t *testing.T) {
	ctx := context.Background()
	out := stage(ctx, stage(ctx, source(ctx, 1, 2, 3, 4),
		func(n int) int { return n * 2 }),
		func(n int) int { return n + 1 })
	if got := collect(out); !slices.Equal(got, []int{3, 5, 7, 9}) {
		t.Errorf("pipeline = %v ; attendu [3 5 7 9]", got)
	}
}

// Parallélisme borné : 3 workers, 8 tâches, ordre préservé. -race confirme
// l'absence de course (chaque worker écrit à un index distinct).
func TestWorkerPoolBounded(t *testing.T) {
	got := workerPool(3, []int{1, 2, 3, 4, 5, 6, 7, 8}, func(n int) int { return n * n })
	if want := []int{1, 4, 9, 16, 25, 36, 49, 64}; !slices.Equal(got, want) {
		t.Errorf("workerPool = %v ; attendu %v", got, want)
	}
}

// errgroup maison : la première erreur est renvoyée, et le contexte partagé est
// annulé (la 3e tâche, qui attend gctx.Done(), se débloque donc).
func TestGroupFirstError(t *testing.T) {
	boom := errors.New("boom")
	g, gctx := NewGroup(context.Background())
	g.Go(func() error { return nil })
	g.Go(func() error { return boom })
	released := make(chan struct{})
	g.Go(func() error {
		<-gctx.Done() // ne se débloque QUE si l'annulation se propage
		close(released)
		return gctx.Err()
	})
	if err := g.Wait(); !errors.Is(err, boom) {
		t.Errorf("group.Wait = %v ; attendu boom", err)
	}
	select {
	case <-released:
	default:
		t.Error("le contexte du groupe n'a pas été annulé")
	}
}

// errgroup maison : aucune erreur -> Wait renvoie nil.
func TestGroupAllSucceed(t *testing.T) {
	g, _ := NewGroup(context.Background())
	for range 5 {
		g.Go(func() error { return nil })
	}
	if err := g.Wait(); err != nil {
		t.Errorf("group.Wait = %v ; attendu nil", err)
	}
}
