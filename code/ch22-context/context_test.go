package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Fin normale : le canal se ferme, on renvoie la somme et nil.
func TestSumNormalCompletion(t *testing.T) {
	in := make(chan int)
	go func() {
		defer close(in)
		for _, n := range []int{1, 2, 3, 4} {
			in <- n
		}
	}()
	sum, err := sumUntilCancel(context.Background(), in)
	if sum != 10 || err != nil {
		t.Errorf("sumUntilCancel = (%d, %v) ; attendu (10, nil)", sum, err)
	}
}

// Annulation avec cause : ctx.Err() == Canceled, mais Cause() rend l'erreur métier.
func TestSumCanceledWithCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(ErrTooSlow) // annulation immédiate avec cause -> select déterministe
	sum, err := sumUntilCancel(ctx, make(chan int))
	if sum != 0 {
		t.Errorf("somme partielle = %d ; attendu 0", sum)
	}
	if !errors.Is(err, ErrTooSlow) {
		t.Errorf("err = %v ; attendu ErrTooSlow", err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("ctx.Err() = %v ; attendu Canceled", ctx.Err())
	}
}

// Délai dépassé : err == DeadlineExceeded (le canal n'est jamais alimenté).
func TestSumTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, err := sumUntilCancel(ctx, make(chan int))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v ; attendu DeadlineExceeded", err)
	}
}

// Valeur de contexte : aller-retour et absence.
func TestRequestIDRoundTrip(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-42")
	if id, ok := RequestID(ctx); !ok || id != "req-42" {
		t.Errorf("RequestID = (%q, %v) ; attendu (\"req-42\", true)", id, ok)
	}
	if id, ok := RequestID(context.Background()); ok {
		t.Errorf("RequestID(vide) = (%q, %v) ; attendu (\"\", false)", id, ok)
	}
}
