package pipeline

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"
)

// seq adapte une tranche en iter.Seq pour alimenter le pipeline.
func seq[T any](items []T) func(func(T) bool) {
	return func(yield func(T) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
}

// collect draine un canal en tranche.
func collect[T any](ch <-chan T) []T {
	out := []T{}
	for v := range ch {
		out = append(out, v)
	}
	return out
}

func TestProcessHappyPath(t *testing.T) {
	ctx := context.Background()
	in := []int{1, 2, 3, 4, 5}
	double := func(_ context.Context, n int) (int, error) { return n * 2, nil }

	out, m, wait := Process(ctx, seq(in), double, Config{Workers: 3, Buffer: 2})
	got := collect(out)
	if err := wait(); err != nil {
		t.Fatalf("wait : %v", err)
	}

	slices.Sort(got) // ordre non déterministe (3 workers) : on trie pour comparer
	if want := []int{2, 4, 6, 8, 10}; !slices.Equal(got, want) {
		t.Errorf("got %v, voulu %v", got, want)
	}
	if snap := m.Snapshot(); snap.Processed != 5 || snap.Failed != 0 {
		t.Errorf("métriques = %+v, voulu 5 traités / 0 échec", snap)
	}
}

// TestProcessFirstError vérifie que la première erreur annule tout le pipeline
// et est restituée par wait(). synctest garantit en prime qu'aucune goroutine
// (feeder, workers, closer) ne fuit après l'annulation.
func TestProcessFirstError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		errBoom := errors.New("boum")
		fn := func(_ context.Context, n int) (int, error) {
			if n == 3 {
				return 0, errBoom
			}
			return n, nil
		}
		// Beaucoup d'éléments : sans annulation correcte, le feeder resterait
		// bloqué sur `in <- it` et synctest signalerait une fuite/un blocage.
		in := make([]int, 100)
		for i := range in {
			in[i] = i
		}

		out, _, wait := Process(context.Background(), seq(in), fn, Config{Workers: 4, Buffer: 1})
		collect(out)
		if err := wait(); !errors.Is(err, errBoom) {
			t.Errorf("wait = %v, voulu errBoom", err)
		}
	})
}

// TestProcessCancel vérifie l'arrêt propre quand le contexte appelant est annulé.
func TestProcessCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Étape lente mais annulable.
		slow := func(ctx context.Context, n int) (int, error) {
			select {
			case <-time.After(time.Hour):
				return n, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
		in := make([]int, 50)

		out, _, wait := Process(ctx, seq(in), slow, Config{Workers: 4, Buffer: 1})

		// Laisse les workers démarrer, puis annule.
		synctest.Wait()
		cancel()

		collect(out)
		if err := wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("wait = %v, voulu context.Canceled", err)
		}
	})
}

// TestProcessMaxInFlight vérifie que la concurrence ne dépasse jamais le nombre
// de workers configuré.
func TestProcessMaxInFlight(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const workers = 3
		// Chaque étape attend que le contexte avance : ainsi plusieurs items
		// sont « en vol » simultanément, ce qui sollicite le pic de concurrence.
		fn := func(_ context.Context, n int) (int, error) {
			time.Sleep(10 * time.Millisecond)
			return n, nil
		}
		in := make([]int, 20)

		out, m, wait := Process(context.Background(), seq(in), fn, Config{Workers: workers, Buffer: 5})
		collect(out)
		if err := wait(); err != nil {
			t.Fatal(err)
		}
		if snap := m.Snapshot(); snap.MaxInFlight > workers {
			t.Errorf("pic de concurrence = %d, ne doit pas dépasser %d", snap.MaxInFlight, workers)
		}
	})
}
