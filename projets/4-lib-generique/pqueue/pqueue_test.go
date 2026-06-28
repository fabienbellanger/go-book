package pqueue_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"example.com/gends/pqueue"
)

func TestOrderedPopsAscending(t *testing.T) {
	q := pqueue.NewOrdered[int]()
	for _, n := range []int{5, 1, 4, 2, 3, 2} {
		q.Push(n)
	}
	var got []int
	for q.Len() > 0 {
		v, ok := q.Pop()
		if !ok {
			t.Fatal("Pop a renvoyé ok=false sur une file non vide")
		}
		got = append(got, v)
	}
	if want := []int{1, 2, 2, 3, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("ordre de sortie = %v, voulu %v", got, want)
	}
}

func TestEmpty(t *testing.T) {
	q := pqueue.NewOrdered[int]()
	if _, ok := q.Pop(); ok {
		t.Error("Pop sur file vide doit renvoyer ok=false")
	}
	if _, ok := q.Peek(); ok {
		t.Error("Peek sur file vide doit renvoyer ok=false")
	}
}

func TestCustomLess(t *testing.T) {
	// Max-heap : less inversé => le plus grand sort en premier.
	q := pqueue.New(func(a, b int) bool { return a > b })
	for _, n := range []int{3, 1, 2} {
		q.Push(n)
	}
	v, _ := q.Peek()
	if v != 3 {
		t.Errorf("Peek = %d, voulu 3 (max-heap)", v)
	}
}

// TestHeapProperty vérifie, sur des entrées aléatoires, que Pop restitue
// toujours un ordre croissant — l'invariant du tas.
func TestHeapProperty(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	q := pqueue.NewOrdered[int]()
	for range 1000 {
		q.Push(r.IntN(10000))
	}
	prev := -1
	for q.Len() > 0 {
		v, _ := q.Pop()
		if v < prev {
			t.Fatalf("sortie non triée : %d après %d", v, prev)
		}
		prev = v
	}
}

func BenchmarkPushPop(b *testing.B) {
	q := pqueue.NewOrdered[int]()
	for i := 0; b.Loop(); i++ {
		q.Push(i & 0x3ff)
		if q.Len() > 512 {
			q.Pop()
		}
	}
}
