package main

import (
	"fmt"
	"sync"
)

func main() {
	wc := WordCount([]string{"go", "fast", "go", "safe", "go"})
	fmt.Printf("WordCount : %v (go=%d)\n", wc, wc["go"])

	// comma-ok : distinguer « absent » de « présent à zéro ».
	if v, ok := wc["rust"]; !ok {
		fmt.Printf("\"rust\" absent (v=%d, ok=%v)\n", v, ok)
	}

	// Ordre d'itération randomisé : deux parcours diffèrent.
	m := map[int]int{}
	for i := range 12 {
		m[i] = i
	}
	orders := IterationOrders(m, 2)
	fmt.Printf("parcours 1 : %s\nparcours 2 : %s\ndifférents ? %v\n",
		orders[0], orders[1], orders[0] != orders[1])

	// Compteur concurrent protégé par Mutex (sinon : crash).
	c := NewSafeCounter()
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			for range 100 {
				c.Inc("hits")
			}
		})
	}
	wg.Wait()
	fmt.Printf("SafeCounter après 100x100 Inc concurrents : hits=%d\n", c.Get("hits"))
}
