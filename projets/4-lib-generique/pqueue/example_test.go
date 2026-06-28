package pqueue_test

import (
	"fmt"

	"example.com/gends/pqueue"
)

func ExampleQueue() {
	q := pqueue.NewOrdered[int]()
	for _, n := range []int{5, 1, 3, 2, 4} {
		q.Push(n)
	}
	var out []int
	for q.Len() > 0 {
		v, _ := q.Pop()
		out = append(out, v)
	}
	fmt.Println(out)
	// Output: [1 2 3 4 5]
}

func ExampleNew() {
	// Une file de tâches : priorité par échéance croissante.
	type task struct {
		name string
		at   int
	}
	q := pqueue.New(func(a, b task) bool { return a.at < b.at })
	q.Push(task{"déjeuner", 12})
	q.Push(task{"réveil", 7})
	q.Push(task{"réunion", 9})

	next, _ := q.Peek()
	fmt.Println(next.name)
	// Output: réveil
}
