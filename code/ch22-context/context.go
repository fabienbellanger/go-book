// Démonstrations du chapitre 22 : context (annulation, délais, valeurs).
// Lancement : depuis code/, `go run ./ch22-context`
package main

import (
	"context"
	"errors"
)

// sumUntilCancel additionne les entiers reçus sur in, jusqu'à épuisement de in
// OU annulation du contexte. C'est le MOTIF CENTRAL de context : surveiller
// ctx.Done() dans le MÊME select que le travail utile, pour rendre la main
// immédiatement quand l'appelant annule (ou qu'un délai expire).
func sumUntilCancel(ctx context.Context, in <-chan int) (int, error) {
	sum := 0
	for {
		select {
		case <-ctx.Done():
			// ctx.Err() vaut Canceled ou DeadlineExceeded ; Cause() peut donner
			// l'erreur métier précise passée à cancel(cause).
			return sum, context.Cause(ctx)
		case v, ok := <-in:
			if !ok {
				return sum, nil // entrée fermée : fin normale
			}
			sum += v
		}
	}
}

// ErrTooSlow illustre une CAUSE d'annulation métier, transmise via cancel(cause)
// et récupérable avec context.Cause — ce que ctx.Err() seul ne dirait pas.
var ErrTooSlow = errors.New("trop lent")
