// Démonstrations du chapitre 20 : canaux et select.
// Lancement : depuis code/, `go run ./ch20-channels-select`
package main

import (
	"sync"
	"time"
)

// gen envoie chaque valeur de nums sur un canal, puis le FERME. Le type de
// retour <-chan int (réception seule) interdit à l'appelant d'envoyer ou de
// fermer : la DIRECTION documente et fait respecter le contrat à la compilation.
func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // « plus rien ne viendra » : permet au range de finir
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

// fanIn fusionne plusieurs canaux d'entrée en un seul (patron fan-in). Chaque
// entrée est drainée par sa propre goroutine ; quand TOUTES ont fini, on ferme
// la sortie. C'est le rôle de la goroutine « wg.Wait puis close ».
func fanIn(inputs ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, in := range inputs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range in { // range sur canal : s'arrête quand in est fermé
				out <- v
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out) // fermer une seule fois, après que toutes les entrées sont taries
	}()
	return out
}

// trySend tente un envoi NON BLOQUANT : la branche default de select s'exécute
// si — et seulement si — aucune autre branche n'est prête. Sans elle, l'envoi
// bloquerait jusqu'à ce qu'un récepteur soit disponible.
func trySend(ch chan<- int, v int) bool {
	select {
	case ch <- v:
		return true
	default:
		return false // l'envoi bloquerait : on renonce
	}
}

// recvWithTimeout attend une valeur sur ch, mais abandonne après d. time.After
// renvoie un canal qui délivre une valeur une fois le délai écoulé : la première
// branche prête de select gagne.
func recvWithTimeout(ch <-chan int, d time.Duration) (int, bool) {
	select {
	case v := <-ch:
		return v, true
	case <-time.After(d):
		return 0, false // délai dépassé : rien n'est arrivé à temps
	}
}

// selectFairness exécute n fois un select dont les DEUX cas sont TOUJOURS prêts
// (chaque branche remet aussitôt une valeur après l'avoir consommée) et compte
// combien de fois chacune est choisie. Démontre la règle du langage : quand
// plusieurs cas sont prêts, select en choisit un par sélection ALÉATOIRE UNIFORME
// — aucune branche n'est favorisée par son ordre d'écriture dans le code.
func selectFairness(n int) (a, b int) {
	chA := make(chan struct{}, 1)
	chB := make(chan struct{}, 1)
	chA <- struct{}{}
	chB <- struct{}{}
	for range n {
		select {
		case <-chA:
			a++
			chA <- struct{}{} // remis aussitôt : les deux cas restent prêts au tour suivant
		case <-chB:
			b++
			chB <- struct{}{}
		}
	}
	return a, b
}
