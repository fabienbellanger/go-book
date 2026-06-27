package main

import (
	"fmt"
	"sync"
)

func main() {
	fmt.Println("lifoOrder      :", lifoOrder()) // [2 1 0]

	snap, live := evalContrast()
	fmt.Printf("evalContrast   : snapshot=%d (figé) live=%d (lu au retour)\n", snap, live)

	fmt.Println("doubleViaDefer :", doubleViaDefer()) // 42

	fmt.Println("processScoped  :", processScoped([]string{"a", "b"}))
	fmt.Println("processInLoop  :", processDeferInLoop([]string{"a", "b"}))

	// Trace d'entrée/sortie via defer.
	var log []string
	func() {
		defer trace("work", &log)() // entrée maintenant, sortie au retour
		log = append(log, "...travail...")
	}()
	fmt.Println("trace          :", log)

	// withLock : section critique protégée, unlock garanti par defer.
	var mu sync.Mutex
	withLock(&mu, func() { fmt.Println("section critique tenue sous verrou") })
}
