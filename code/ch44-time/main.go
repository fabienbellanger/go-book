// Command ch44-time illustre les usages courants du package time :
// mesure de durée (horloge monotone), timers/tickers correctement arrêtés,
// fuseaux horaires et deadlines via context.
package main

import (
	"context"
	"fmt"
	"time"
)

// measure renvoie le temps écoulé pendant l'exécution de work.
//
// time.Now() embarque une lecture d'horloge MONOTONE : la soustraction de deux
// instants (via time.Since, qui appelle Sub) reste correcte même si l'horloge
// murale est ajustée (NTP, changement d'heure) pendant la mesure.
func measure(work func()) time.Duration {
	start := time.Now()
	work()
	return time.Since(start) // == time.Now().Sub(start), composante monotone
}

// drainTimer arrête un *time.Timer et draine son canal si le timer avait déjà
// expiré, pour ne pas laisser une valeur en attente (utile avant un Reset).
func drainTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

// countTicks compte les tops émis par un ticker pendant la durée window, puis
// arrête proprement le ticker. Le ticker est TOUJOURS stoppé via defer.
func countTicks(interval, window time.Duration) int {
	ticker := time.NewTicker(interval)
	defer ticker.Stop() // sans Stop, le ticker continuerait à émettre

	deadline := time.NewTimer(window)
	defer drainTimer(deadline)

	count := 0
	for {
		select {
		case <-ticker.C:
			count++
		case <-deadline.C:
			return count
		}
	}
}

// slowDouble double n après un délai, en respectant l'annulation du contexte.
// Si le contexte expire avant le délai, il renvoie ctx.Err().
func slowDouble(ctx context.Context, n int, delay time.Duration) (int, error) {
	timer := time.NewTimer(delay)
	defer drainTimer(timer)

	select {
	case <-timer.C:
		return n * 2, nil
	case <-ctx.Done():
		return 0, ctx.Err() // context.DeadlineExceeded ou Canceled
	}
}

func main() {
	// Mesure de durée (horloge monotone).
	d := measure(func() {
		sum := 0
		for i := range 1_000_000 {
			sum += i
		}
		_ = sum
	})
	fmt.Printf("calcul effectué en %v\n", d.Round(time.Microsecond))

	// Fuseaux : stocker en UTC, afficher en local.
	utc := time.Date(2025, time.June, 28, 14, 30, 0, 0, time.UTC)
	if paris, err := time.LoadLocation("Europe/Paris"); err == nil {
		fmt.Printf("UTC=%s  Paris=%s\n",
			utc.Format(time.TimeOnly), utc.In(paris).Format(time.TimeOnly))
	}

	// Ticker borné par un timer.
	fmt.Printf("tops en 55ms à 10ms : %d\n", countTicks(10*time.Millisecond, 55*time.Millisecond))

	// Deadline via context : le délai (200ms) dépasse la deadline (20ms).
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := slowDouble(ctx, 21, 200*time.Millisecond); err != nil {
		fmt.Printf("slowDouble annulé : %v\n", err)
	}
}
