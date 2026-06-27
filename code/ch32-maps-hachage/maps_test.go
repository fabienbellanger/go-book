package main

import (
	"sync"
	"testing"
)

func TestWordCount(t *testing.T) {
	wc := WordCount([]string{"a", "b", "a", "c", "a", "b"})
	want := map[string]int{"a": 3, "b": 2, "c": 1}
	if len(wc) != len(want) {
		t.Fatalf("len = %d ; attendu %d", len(wc), len(want))
	}
	for k, v := range want {
		if wc[k] != v {
			t.Errorf("wc[%q] = %d ; attendu %d", k, wc[k], v)
		}
	}
	if _, ok := wc["absent"]; ok {
		t.Error("clé absente signalée présente")
	}
}

// L'itération visite TOUTES les clés, quel que soit l'ordre.
func TestIterationVisitsAll(t *testing.T) {
	m := map[int]int{}
	for i := range 20 {
		m[i] = i * i
	}
	seen := map[int]bool{}
	for k, v := range m {
		if v != k*k {
			t.Errorf("m[%d] = %d ; attendu %d", k, v, k*k)
		}
		seen[k] = true
	}
	if len(seen) != 20 {
		t.Errorf("visité %d clés ; attendu 20", len(seen))
	}
}

// L'ordre d'itération est randomisé : sur plusieurs parcours, ils ne sont pas tous
// identiques (probabilité de coïncidence négligeable avec 16 clés).
func TestIterationRandomized(t *testing.T) {
	m := map[int]int{}
	for i := range 16 {
		m[i] = i
	}
	orders := IterationOrders(m, 10)
	allSame := true
	for i := 1; i < len(orders); i++ {
		if orders[i] != orders[0] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("10 parcours identiques : la randomisation d'itération semble absente")
	}
}

// La préallocation make(map, n) réduit le nombre d'allocations à la construction.
func TestPreallocReducesAllocs(t *testing.T) {
	const n = 1000
	noPre := testing.AllocsPerRun(5, func() {
		m := make(map[int]int)
		for i := range n {
			m[i] = i
		}
	})
	pre := testing.AllocsPerRun(5, func() {
		m := make(map[int]int, n)
		for i := range n {
			m[i] = i
		}
	})
	if pre >= noPre {
		t.Errorf("prealloc=%.0f devrait être < sans prealloc=%.0f", pre, noPre)
	}
	t.Logf("allocs/op : sans prealloc=%.0f, avec make(,%d)=%.0f", noPre, n, pre)
}

// Accès concurrent correct grâce au Mutex : le total est exact et -race est propre.
func TestSafeCounterConcurrent(t *testing.T) {
	c := NewSafeCounter()
	var wg sync.WaitGroup
	const goroutines, perG = 50, 200
	for range goroutines {
		wg.Go(func() {
			for range perG {
				c.Inc("k")
			}
		})
	}
	wg.Wait()
	if got := c.Get("k"); got != goroutines*perG {
		t.Errorf("compteur = %d ; attendu %d", got, goroutines*perG)
	}
}
