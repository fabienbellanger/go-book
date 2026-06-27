package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	numCPU, gmp := ActiveCPUs()
	fmt.Printf("NumCPU=%d  GOMAXPROCS=%d (plafond de parallélisme)\n", numCPU, gmp)

	// Même calcul, même résultat, que l'on utilise 1 ou N goroutines.
	nums := make([]int, 1_000_000)
	for i := range nums {
		nums[i] = i
	}
	fmt.Printf("parallelSum(1 worker)  = %d\n", parallelSum(nums, 1))
	fmt.Printf("parallelSum(%d workers) = %d\n", gmp, parallelSum(nums, gmp))

	// Démonstration du parallélisme : 8 tâches CPU, GOMAXPROCS=1 vs défaut.
	run := func() {
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() { _ = busyWork(20_000_000) })
		}
		wg.Wait()
	}
	var d1, dn time.Duration
	WithGOMAXPROCS(1, func() { t := time.Now(); run(); d1 = time.Since(t) })
	t := time.Now()
	run()
	dn = time.Since(t)
	fmt.Printf("8 tâches CPU : GOMAXPROCS=1 -> %v ; GOMAXPROCS=%d -> %v\n", d1.Round(time.Millisecond), gmp, dn.Round(time.Millisecond))

	// Pour observer l'ordonnanceur en direct :
	//   GODEBUG=schedtrace=200 go run ./ch28-ordonnanceur-gmp
}
