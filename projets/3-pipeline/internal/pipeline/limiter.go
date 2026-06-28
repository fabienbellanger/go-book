package pipeline

import (
	"context"
	"time"
)

// RateLimiter limite le débit à N opérations par seconde, partagé entre tous
// les workers. Il s'appuie sur un time.Ticker : chaque Wait consomme un tic.
//
// C'est un seau à jetons *sans rafale* (le ticker ne cumule pas les tics
// manqués). Simple et suffisant pour de la régulation ; pour des bursts ou des
// politiques fines, préférer golang.org/x/time/rate (voir README).
//
// Sous testing/synctest, time.Ticker utilise l'horloge virtuelle : les tests de
// débit s'exécutent instantanément tout en respectant les délais simulés.
type RateLimiter struct {
	ticker *time.Ticker
}

// NewRateLimiter crée un limiteur à perSecond opérations par seconde.
// Penser à appeler Stop pour libérer le ticker.
func NewRateLimiter(perSecond int) *RateLimiter {
	if perSecond < 1 {
		perSecond = 1
	}
	return &RateLimiter{ticker: time.NewTicker(time.Second / time.Duration(perSecond))}
}

// Wait bloque jusqu'au prochain jeton, ou rend l'erreur du contexte s'il est
// annulé d'ici là.
func (r *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-r.ticker.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop arrête le ticker sous-jacent.
func (r *RateLimiter) Stop() { r.ticker.Stop() }
