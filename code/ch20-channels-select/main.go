package main

import (
	"fmt"
	"slices"
	"time"
)

func main() {
	// 1. Générateur + range : le range s'arrête quand le canal est fermé.
	fmt.Print("gen + range : ")
	for v := range gen(1, 2, 3) {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// 2. Fan-in : fusionner deux sources en un seul canal (ordre non garanti).
	merged := fanIn(gen(1, 2, 3), gen(10, 20, 30))
	var all []int
	for v := range merged {
		all = append(all, v)
	}
	slices.Sort(all) // l'ordre d'arrivée est indéterminé : on trie pour l'affichage
	fmt.Println("fan-in (trié) :", all)

	// 3. Envoi non bloquant via select+default.
	buf := make(chan int, 1)
	fmt.Println("trySend (vide) :", trySend(buf, 1)) // true : il reste de la place
	fmt.Println("trySend (plein):", trySend(buf, 2)) // false : le tampon est plein

	// 4. Réception avec délai via select + time.After.
	empty := make(chan int)
	if _, ok := recvWithTimeout(empty, 20*time.Millisecond); !ok {
		fmt.Println("recvWithTimeout : délai dépassé (rien reçu)")
	}
	ready := gen(42)
	if v, ok := recvWithTimeout(ready, time.Second); ok {
		fmt.Println("recvWithTimeout : reçu", v)
	}
}
