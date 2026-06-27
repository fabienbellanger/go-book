package main

import (
	"fmt"
	"runtime"
)

func square(n int) int { return n * n }

func main() {
	// Le runtime expose l'état de l'ordonnanceur (Ch. 28).
	fmt.Println("GOMAXPROCS              :", runtime.GOMAXPROCS(0))
	fmt.Println("goroutines au démarrage :", runtime.NumGoroutine()) // 1 : main

	// 1. parallelMap : une goroutine par élément, résultat dans l'ordre.
	out := parallelMap([]int{1, 2, 3, 4, 5}, square)
	fmt.Println("parallelMap carrés      :", out) // [1 4 9 16 25]

	// 2. Arrêt propre : la goroutine surveille un canal d'arrêt.
	stop := make(chan struct{})
	count, done := tickUntilStop(stop)
	for count.Load() == 0 { // laisser la goroutine démarrer (pour une démo non triviale)
	}
	close(stop) // on DEMANDE l'arrêt
	<-done      // on ATTEND la confirmation : pas de fuite
	fmt.Printf("la goroutine a tourné %d fois puis s'est arrêtée proprement\n", count.Load())
}
