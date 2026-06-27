package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// synctest (1.25) : tester le rate limiting SANS attente réelle. Dans la
// « bulle », l'horloge est virtuelle : 5 appels espacés de 100 ms « prennent »
// 500 ms de temps virtuel, mais le test s'exécute instantanément.
func TestRateLimitedVirtualTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var count atomic.Int64
		start := time.Now()
		rateLimited(context.Background(), []int{1, 2, 3, 4, 5}, 100*time.Millisecond,
			func(int) { count.Add(1) })

		if got := count.Load(); got != 5 {
			t.Fatalf("count = %d ; attendu 5", got)
		}
		if elapsed := time.Since(start); elapsed != 500*time.Millisecond {
			t.Errorf("temps virtuel = %v ; attendu 500ms", elapsed)
		}
	})
}

// synctest : tester un TIMEOUT déterministe, sans time.Sleep réel. Le contexte
// expire à t=1s virtuel, bien avant le time.After(2s) concurrent.
func TestTimeoutVirtual(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		select {
		case <-ctx.Done():
			// se déclenche à t=1s virtuel
		case <-time.After(2 * time.Second):
			t.Fatal("le timeout aurait dû se déclencher en premier")
		}
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Errorf("ctx.Err() = %v ; attendu DeadlineExceeded", ctx.Err())
		}
	})
}
