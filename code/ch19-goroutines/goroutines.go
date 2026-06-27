// Démonstrations du chapitre 19 : le modèle des goroutines.
// Lancement : depuis code/, `go run ./ch19-goroutines`
package main

import (
	"sync"
	"sync/atomic"
)

// parallelMap applique f à chaque élément dans SA PROPRE goroutine, puis attend
// que toutes aient fini. Chaque goroutine écrit à un index DISTINCT de out : il
// n'y a donc aucune course, et l'ordre du résultat est préservé.
//
// sync.WaitGroup (Ch. 21) est le compteur qui permet d'attendre la fin de N
// goroutines : Add(1) avant de lancer, Done() à la fin de chacune, Wait() bloque
// tant que le compteur n'est pas revenu à zéro.
func parallelMap[T, U any](items []T, f func(T) U) []U {
	out := make([]U, len(items))
	var wg sync.WaitGroup
	for i, item := range items { // Go 1.22+ : i et item sont propres à l'itération
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = f(item) // index distinct par goroutine : pas de course
		}()
	}
	wg.Wait() // bloque jusqu'à ce que les len(items) goroutines aient terminé
	return out
}

// tickUntilStop lance une goroutine qui incrémente un compteur tant qu'on ne lui
// demande pas de s'arrêter. C'est le patron de l'ARRÊT PROPRE : la goroutine
// surveille un canal d'arrêt et rend la main dès qu'on le ferme — elle ne fuit
// donc jamais. Elle ferme done en sortant pour signaler sa terminaison.
func tickUntilStop(stop <-chan struct{}) (count *atomic.Int64, done <-chan struct{}) {
	count = &atomic.Int64{}
	d := make(chan struct{})
	go func() {
		defer close(d) // signale « j'ai terminé » à coup sûr
		for {
			select {
			case <-stop:
				return // arrêt demandé : la goroutine se termine
			default:
				count.Add(1)
			}
		}
	}()
	return count, d
}
