// Démonstrations du chapitre 18 : itérateurs par fonction (range-over-func).
// Lancement : depuis code/, `go run ./ch18-iterators`
package main

import "iter"

// Count produit 0, 1, ..., n-1. C'est un itérateur PUSH (iter.Seq[int]) : il
// POUSSE chaque valeur dans yield. yield renvoie false quand le consommateur
// veut s'arrêter (un break dans le range) ; on respecte ce signal en sortant.
func Count(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			if !yield(i) {
				return
			}
		}
	}
}

// Naturals est un itérateur INFINI (0, 1, 2, ...). Il est sûr car le consommateur
// peut l'arrêter à tout moment (break, ou un combinateur comme Take).
func Naturals() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; ; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

// Map transforme chaque élément PARESSEUSEMENT : f n'est appelée que pour les
// éléments réellement consommés, et aucune slice intermédiaire n'est créée.
func Map[A, B any](seq iter.Seq[A], f func(A) B) iter.Seq[B] {
	return func(yield func(B) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// Filter ne laisse passer que les éléments retenus par keep.
func Filter[V any](seq iter.Seq[V], keep func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if keep(v) && !yield(v) {
				return
			}
		}
	}
}

// Take limite la séquence à ses n premiers éléments. C'est lui qui rend une
// source infinie exploitable.
func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	return func(yield func(V) bool) {
		if n <= 0 {
			return
		}
		count := 0
		for v := range seq {
			if !yield(v) {
				return
			}
			if count++; count >= n {
				return // assez d'éléments : on arrête (et on stoppe la source)
			}
		}
	}
}

// Enumerate associe à chaque valeur son index : un itérateur à DEUX valeurs
// (iter.Seq2[int, V]), comme slices.All.
func Enumerate[V any](seq iter.Seq[V]) iter.Seq2[int, V] {
	return func(yield func(int, V) bool) {
		i := 0
		for v := range seq {
			if !yield(i, v) {
				return
			}
			i++
		}
	}
}

// BrokenAfterStop est un itérateur volontairement BOGUÉ : il ignore le booléen
// renvoyé par yield et continue d'en appeler même après une demande d'arrêt.
// Il sert à illustrer le piège documenté au chapitre : le runtime ne laisse
// pas faire, voir TestYieldAfterStopPanics dans iterators_test.go.
func BrokenAfterStop() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < 3; i++ {
			yield(i) // BUG : la valeur de retour n'est jamais vérifiée
		}
	}
}
