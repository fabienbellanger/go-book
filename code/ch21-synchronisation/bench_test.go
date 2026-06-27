package main

import (
	"sync"
	"testing"
)

// Compteur partagé sous contention (RunParallel) : atomic vs mutex.
func BenchmarkAtomicInc(b *testing.B) {
	var c AtomicCounter
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkMutexInc(b *testing.B) {
	var c SafeCounter
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

// Lecture pure sous contention : RWMutex (lecteurs parallèles) vs Mutex (exclusif).
func BenchmarkRWMutexRead(b *testing.B) {
	var mu sync.RWMutex
	n := 42
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			_ = n
			mu.RUnlock()
		}
	})
}

func BenchmarkMutexRead(b *testing.B) {
	var mu sync.Mutex
	n := 42
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			_ = n
			mu.Unlock()
		}
	})
}
