package cli

import (
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
)

// defaultWorkers détermine le nombre de workers par défaut.
//
// Ordre de priorité (du plus faible au plus fort) :
//  1. valeur par défaut = GOMAXPROCS (nombre de cœurs utilisables) ;
//  2. variable d'environnement TXTKIT_WORKERS si elle est un entier > 0.
//
// Le flag -j de chaque commande prime ensuite sur cette valeur : c'est le
// schéma classique « défaut < environnement < ligne de commande ».
func defaultWorkers() int {
	n := runtime.GOMAXPROCS(0)
	if v, ok := os.LookupEnv("TXTKIT_WORKERS"); ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		}
	}
	return n
}

// source nomme une entrée à traiter : soit un fichier sur disque, soit stdin.
type source struct {
	name string // nom affiché ("-" pour stdin)
	open func() (io.ReadCloser, error)
}

// sourcesFrom construit la liste des sources à partir des arguments.
// Sans fichier, on lit stdin ; le nom affiché est alors "-".
func sourcesFrom(files []string, stdin io.Reader) []source {
	if len(files) == 0 {
		return []source{{
			name: "-",
			open: func() (io.ReadCloser, error) { return io.NopCloser(stdin), nil },
		}}
	}
	srcs := make([]source, len(files))
	for i, f := range files {
		srcs[i] = source{
			name: f,
			open: func() (io.ReadCloser, error) { return os.Open(f) },
		}
	}
	return srcs
}

// mapBounded applique f à chaque élément avec au plus n goroutines simultanées,
// en préservant l'ordre des résultats (out[i] correspond à items[i]).
//
// Le canal `sem` joue le rôle de sémaphore comptant : on y dépose un jeton
// avant de lancer un worker (ce qui bloque quand n jetons sont déjà pris) et on
// le retire à la fin. C'est le pattern « worker borné » : on profite du
// parallélisme des I/O sans lancer une goroutine par fichier sans limite.
func mapBounded[T, R any](items []T, n int, f func(T) R) []R {
	if n < 1 {
		n = 1
	}
	out := make([]R, len(items))
	sem := make(chan struct{}, n)
	var wg sync.WaitGroup
	for i, it := range items { // Go 1.22+ : i et it sont propres à l'itération
		sem <- struct{}{} // bloque si n workers tournent déjà
		wg.Go(func() {    // WaitGroup.Go (Go 1.25) : lance + compte en un appel
			defer func() { <-sem }()
			out[i] = f(it)
		})
	}
	wg.Wait()
	return out
}
