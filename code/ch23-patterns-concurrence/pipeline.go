// Démonstrations du chapitre 23 : patterns de concurrence.
// Lancement : depuis code/, `go run ./ch23-patterns-concurrence`
package main

import (
	"context"
	"sync"
)

// source émet les valeurs de nums sur un canal, en s'arrêtant si le contexte est
// annulé. C'est le premier maillon d'un pipeline ; la surveillance de ctx.Done()
// évite que la goroutine fuie quand l'aval abandonne.
func source(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return // annulation : on cesse d'émettre
			}
		}
	}()
	return out
}

// stage est un maillon générique de pipeline : il applique f à chaque valeur
// reçue et la pousse sur un nouveau canal. Chaque stage tourne dans sa propre
// goroutine, donc les maillons s'exécutent EN PARALLÈLE, en flux — rien n'est
// matérialisé entre eux.
func stage[A, B any](ctx context.Context, in <-chan A, f func(A) B) <-chan B {
	out := make(chan B)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- f(v):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// workerPool applique f à chaque élément avec AU PLUS n workers en parallèle :
// le patron du parallélisme BORNÉ. n goroutines tirent des indices d'un même
// canal ; chacune écrit à un index distinct de out (donc pas de course) et
// l'ordre du résultat est préservé. À préférer à « une goroutine par tâche »
// quand les tâches sont nombreuses.
func workerPool[T, U any](n int, items []T, f func(T) U) []U {
	out := make([]U, len(items))
	idx := make(chan int)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			for i := range idx { // chaque worker prend le prochain index libre
				out[i] = f(items[i])
			}
		})
	}
	for i := range items {
		idx <- i
	}
	close(idx) // plus d'indices : les workers sortent de leur range
	wg.Wait()
	return out
}

// collect draine un canal jusqu'à sa fermeture et renvoie toutes ses valeurs.
func collect[T any](ch <-chan T) []T {
	var out []T
	for v := range ch {
		out = append(out, v)
	}
	return out
}
