package main

import "testing"

func TestStack(t *testing.T) {
	var s Stack[int]
	if _, ok := s.Pop(); ok {
		t.Error("Pop sur pile vide doit renvoyer false")
	}
	s.Push(1)
	s.Push(2)
	s.Push(3)
	if s.Len() != 3 {
		t.Errorf("Len = %d, voulu 3", s.Len())
	}
	for _, want := range []int{3, 2, 1} { // ordre LIFO
		if got, ok := s.Pop(); !ok || got != want {
			t.Errorf("Pop = (%d, %v), voulu %d", got, ok, want)
		}
	}
}

func TestQueue(t *testing.T) {
	var q Queue[string]
	q.Enqueue("a")
	q.Enqueue("b")
	q.Enqueue("c")
	for _, want := range []string{"a", "b", "c"} { // ordre FIFO
		if got, ok := q.Dequeue(); !ok || got != want {
			t.Errorf("Dequeue = (%q, %v), voulu %q", got, ok, want)
		}
	}
	if _, ok := q.Dequeue(); ok {
		t.Error("Dequeue sur file vide doit renvoyer false")
	}
}

func TestUnionFind(t *testing.T) {
	uf := NewUnionFind(6)
	if uf.Connected(0, 1) {
		t.Error("0 et 1 ne doivent pas être connectés au départ")
	}
	uf.Union(0, 1)
	uf.Union(1, 2)
	uf.Union(3, 4)
	if !uf.Connected(0, 2) {
		t.Error("0 et 2 devraient être connectés (transitivité)")
	}
	if uf.Connected(0, 3) {
		t.Error("0 et 3 ne doivent pas être connectés")
	}
	if uf.Union(0, 2) {
		t.Error("Union d'éléments déjà unis doit renvoyer false")
	}
}
