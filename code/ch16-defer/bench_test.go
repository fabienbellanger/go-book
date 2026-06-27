package main

import "testing"

//go:noinline
func work() { sink++ }

// withDeferFixed : un seul defer à position fixe -> open-coded (quasi gratuit).
func withDeferFixed() {
	defer work()
}

// withoutDefer : appel direct, pour comparer.
func withoutDefer() {
	work()
}

// deferLoop : defers empilés dans une boucle -> mécanisme runtime (plus lent).
func deferLoop(n int) {
	for range n {
		defer work()
	}
}

var sink int

func BenchmarkWithDefer(b *testing.B) {
	for b.Loop() {
		withDeferFixed()
	}
}

func BenchmarkWithoutDefer(b *testing.B) {
	for b.Loop() {
		withoutDefer()
	}
}

func BenchmarkDeferInLoop(b *testing.B) {
	for b.Loop() {
		deferLoop(8)
	}
}
