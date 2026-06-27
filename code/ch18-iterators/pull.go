package main

import "iter"

// Zip combine deux séquences en paires, jusqu'à épuisement de la plus courte.
//
// Impossible avec deux range imbriqués : il faut AVANCER les deux sources en
// parallèle. On convertit donc chaque itérateur PUSH en itérateur PULL avec
// iter.Pull, ce qui rend la main au consommateur (next/stop). On TIRE une valeur
// de chaque à chaque tour.
func Zip[A, B any](sa iter.Seq[A], sb iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		nextA, stopA := iter.Pull(sa)
		defer stopA() // libère la goroutine sous-jacente de Pull
		nextB, stopB := iter.Pull(sb)
		defer stopB()
		for {
			a, okA := nextA()
			b, okB := nextB()
			if !okA || !okB {
				return // une source épuisée -> fin
			}
			if !yield(a, b) {
				return
			}
		}
	}
}
