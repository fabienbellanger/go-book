// Package pqueue fournit une file de priorité générique, implémentée comme un
// tas binaire (min-heap) stocké dans une tranche.
//
// La priorité est définie par une fonction de comparaison « less » : si
// less(a, b) est vrai, a est prioritaire sur b et sortira en premier. Pour des
// types ordonnés, [NewOrdered] fournit directement un min-heap.
//
// Le type zéro [Queue] n'est pas prêt à l'emploi : construire avec [New] ou
// [NewOrdered]. Queue n'est pas sûr pour un usage concurrent.
package pqueue

import "cmp"

// Queue est une file de priorité d'éléments de type T.
type Queue[T any] struct {
	items []T
	less  func(a, b T) bool
}

// New crée une file de priorité dont l'ordre est donné par less.
// less(a, b) doit renvoyer vrai lorsque a est prioritaire sur b.
func New[T any](less func(a, b T) bool) *Queue[T] {
	if less == nil {
		panic("pqueue: less ne doit pas être nil")
	}
	return &Queue[T]{less: less}
}

// NewOrdered crée un min-heap pour un type ordonné : le plus petit élément sort
// en premier. Pratique pour les nombres, chaînes, etc.
func NewOrdered[T cmp.Ordered]() *Queue[T] {
	return New(func(a, b T) bool { return a < b })
}

// Len renvoie le nombre d'éléments dans la file.
func (q *Queue[T]) Len() int { return len(q.items) }

// Push insère un élément.
func (q *Queue[T]) Push(v T) {
	q.items = append(q.items, v)
	q.up(len(q.items) - 1)
}

// Peek renvoie l'élément prioritaire sans le retirer. ok vaut false si la file
// est vide.
func (q *Queue[T]) Peek() (v T, ok bool) {
	if len(q.items) == 0 {
		return v, false
	}
	return q.items[0], true
}

// Pop retire et renvoie l'élément prioritaire. ok vaut false si la file est vide.
func (q *Queue[T]) Pop() (v T, ok bool) {
	n := len(q.items)
	if n == 0 {
		return v, false
	}
	top := q.items[0]
	last := n - 1
	q.items[0] = q.items[last]

	var zero T
	q.items[last] = zero // libère la référence (utile si T contient des pointeurs)
	q.items = q.items[:last]

	if len(q.items) > 0 {
		q.down(0)
	}
	return top, true
}

// up rétablit la propriété de tas en remontant l'élément i vers la racine.
func (q *Queue[T]) up(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !q.less(q.items[i], q.items[parent]) {
			break
		}
		q.items[i], q.items[parent] = q.items[parent], q.items[i]
		i = parent
	}
}

// down rétablit la propriété de tas en descendant l'élément i vers les feuilles.
func (q *Queue[T]) down(i int) {
	n := len(q.items)
	for {
		left, right := 2*i+1, 2*i+2
		best := i
		if left < n && q.less(q.items[left], q.items[best]) {
			best = left
		}
		if right < n && q.less(q.items[right], q.items[best]) {
			best = right
		}
		if best == i {
			return
		}
		q.items[i], q.items[best] = q.items[best], q.items[i]
		i = best
	}
}
