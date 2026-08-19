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

// synctest.Sleep (🆕 1.27) : équivaut à time.Sleep(d) suivi de synctest.Wait().
//
// POURQUOI ÇA COMPTE. Quand le test ET le code testé dorment EXACTEMENT la même
// durée virtuelle, les deux goroutines redeviennent exécutables au même instant
// et l'ordre de reprise n'est pas défini : le test observe tantôt l'état
// d'avant, tantôt celui d'après. synctest.Sleep supprime l'ambiguïté en
// attendant en plus que la bulle se STABILISE — toutes les autres goroutines
// durablement bloquées — avant de rendre la main.
func TestSynctestSleepAttendLaStabilisation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var prêt atomic.Bool

		// Le code « sous test » bascule un drapeau après exactement 100 ms.
		go func() {
			time.Sleep(100 * time.Millisecond)
			prêt.Store(true)
		}()

		// Même durée que la goroutine ci-dessus : c'est le cas ambigu.
		// synctest.Sleep garantit qu'au retour, la goroutine a fini.
		synctest.Sleep(100 * time.Millisecond)

		if !prêt.Load() {
			t.Error("après synctest.Sleep, la goroutine aurait dû avoir terminé")
		}
	})
}

// Variante explicite : le duo time.Sleep + synctest.Wait, que synctest.Sleep
// condense en un seul appel. Les deux formes sont exactement équivalentes.
func TestSleepPuisWaitEquivalent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var compteur atomic.Int64
		for range 3 {
			go func() {
				time.Sleep(50 * time.Millisecond)
				compteur.Add(1)
			}()
		}

		time.Sleep(50 * time.Millisecond)
		synctest.Wait() // ce que synctest.Sleep ferait tout seul

		if got := compteur.Load(); got != 3 {
			t.Errorf("compteur = %d ; attendu 3", got)
		}
	})
}
