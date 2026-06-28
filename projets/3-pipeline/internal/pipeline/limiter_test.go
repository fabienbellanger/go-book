package pipeline

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

// TestRateLimiter démontre l'horloge virtuelle de synctest : 5 éléments à
// 10/seconde avec un seul worker prennent exactement 500 ms simulées — le test
// s'exécute pourtant instantanément.
func TestRateLimiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := NewRateLimiter(10) // un jeton toutes les 100 ms
		defer lim.Stop()

		start := time.Now()
		identity := func(_ context.Context, n int) (int, error) { return n, nil }

		out, _, wait := Process(context.Background(), seq([]int{0, 1, 2, 3, 4}), identity,
			Config{Workers: 1, Buffer: 0, Limiter: lim})
		collect(out)
		if err := wait(); err != nil {
			t.Fatal(err)
		}

		if elapsed := time.Since(start); elapsed != 500*time.Millisecond {
			t.Errorf("durée simulée = %v, voulu 500ms", elapsed)
		}
	})
}

func TestRateLimiterCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := NewRateLimiter(1) // un jeton par seconde
		defer lim.Stop()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := lim.Wait(ctx); err == nil {
			t.Error("Wait sur contexte annulé doit renvoyer une erreur")
		}
	})
}
