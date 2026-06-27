package main

import (
	"context"
	"time"
)

// rateLimited appelle f pour chaque élément, à raison d'un AU PLUS toutes les
// `every`. Le ticker délivre un « jeton » périodique : on attend un jeton avant
// chaque appel, et on s'interrompt si le contexte est annulé.
//
// Tester ce code en temps réel serait lent et instable. C'est le cas d'usage
// emblématique de testing/synctest (1.25) et de son horloge virtuelle : voir
// synctest_test.go, où 5 appels à 100 ms « prennent » 500 ms... instantanément.
func rateLimited(ctx context.Context, items []int, every time.Duration, f func(int)) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for _, it := range items {
		select {
		case <-tick.C:
			f(it)
		case <-ctx.Done():
			return
		}
	}
}
