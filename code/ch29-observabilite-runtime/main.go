package main

import (
	"fmt"
	"sync"
)

func main() {
	before := ReadSnapshot()
	fmt.Printf("Go %s, GOMAXPROCS=%d\n", before.GoVersion, before.GOMAXPROCS)
	fmt.Printf("au repos : goroutines utilisateur=%d, toutes=%d (écart = système)\n",
		before.Goroutines, before.GoroutinesAll)

	// Lançons 50 goroutines bloquées et reprenons un instantané.
	block := make(chan struct{})
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { <-block })
	}
	during := ReadSnapshot()
	fmt.Printf("50 lancées : goroutines=%d, toutes=%d, créées (cumul)=%d\n",
		during.Goroutines, during.GoroutinesAll, during.GoroutinesCreated)
	close(block)
	wg.Wait()

	// Quelques requêtes applicatives (compteur expvar).
	for range 3 {
		RecordRequest()
	}
	fmt.Printf("requests_served=%d  heap=%d o  objets=%d  GC=%d\n",
		RequestsServed(), during.HeapAllocBytes, during.HeapObjects, during.NumGC)

	fmt.Println("\nExposez le tout en HTTP : import _ \"net/http/pprof\" + expvar")
	fmt.Println("  -> http://localhost:PORT/debug/vars  et  /debug/pprof/")
}
