// Package main illustre l'ordonnanceur G-M-P côté observable : combien de P
// (GOMAXPROCS) bornent le parallélisme, comment répartir un calcul CPU sur
// plusieurs goroutines, et le fait que le RÉSULTAT ne dépend pas du parallélisme.
package main

import (
	"runtime"
	"sync"
)

// ActiveCPUs renvoie les cœurs logiques vus par le runtime (M potentiels) et le
// nombre de P actifs (le plafond de parallélisme réel).
func ActiveCPUs() (numCPU, gomaxprocs int) {
	return runtime.NumCPU(), runtime.GOMAXPROCS(0)
}

// WithGOMAXPROCS exécute f avec un nombre de P temporaire, puis restaure l'ancien.
func WithGOMAXPROCS(n int, f func()) {
	old := runtime.GOMAXPROCS(n)
	defer runtime.GOMAXPROCS(old)
	f()
}

// parallelSum répartit nums sur `workers` goroutines (fan-out). Chaque goroutine
// somme sa tranche dans une case DISTINCTE (aucune course), puis on combine.
// Quel que soit le nombre de workers ou de P, le total est le même.
func parallelSum(nums []int, workers int) int {
	if workers < 1 {
		workers = 1
	}
	if workers > len(nums) {
		workers = max(1, len(nums))
	}
	partials := make([]int, workers)
	chunk := (len(nums) + workers - 1) / workers

	var wg sync.WaitGroup
	for w := range workers {
		start := w * chunk
		end := min(start+chunk, len(nums))
		if start >= end {
			continue
		}
		wg.Go(func() {
			s := 0
			for _, v := range nums[start:end] {
				s += v
			}
			partials[w] = s // case distincte : pas de synchronisation nécessaire
		})
	}
	wg.Wait()

	total := 0
	for _, p := range partials {
		total += p
	}
	return total
}

// busyWork effectue un calcul purement CPU (sert à observer le parallélisme).
//
//go:noinline
func busyWork(iterations int) int {
	x := 0
	for i := range iterations {
		x += i % 7
		if i%1_000_000 == 0 {
			runtime.Gosched() // point de coopération : céder volontairement la main
		}
	}
	return x
}
