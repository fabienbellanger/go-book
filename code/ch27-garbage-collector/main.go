package main

import (
	"fmt"
	"runtime"
)

func main() {
	c := NewCache()

	// Tant que r est tenu fortement, le cache le retrouve.
	r := &Resource{ID: 1}
	c.Put(r)
	fmt.Println("avant GC, Get(1) != nil :", c.Get(1) != nil)

	// On lâche la référence forte, puis on force un GC : la référence FAIBLE
	// tombe à nil — l'objet a été collecté.
	runtime.KeepAlive(r) // garde r vivant jusqu'ici
	r = nil
	runtime.GC()
	fmt.Println("après GC, Get(1) == nil :", c.Get(1) == nil)

	// Réglages du GC.
	fmt.Printf("GOMEMLIMIT actuel : %d (MaxInt64 = aucune limite)\n", CurrentMemoryLimit())
	WithGCPercent(50, func() {
		fmt.Println("dans WithGCPercent(50) : GC plus agressif, moins de mémoire")
	})

	// Pour voir chaque cycle de GC :
	//   GODEBUG=gctrace=1 go run ./ch27-garbage-collector
}
