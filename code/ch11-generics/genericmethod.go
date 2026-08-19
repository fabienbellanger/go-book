package main

import "strconv"

// List est un type générique ordinaire : E vient de sa déclaration.
type List[E any] []E

// Apply est une MÉTHODE GÉNÉRIQUE (🆕 Go 1.27) : en plus de E (fourni par le
// récepteur), elle déclare son PROPRE paramètre de type F.
//
// Avant 1.27, une méthode ne pouvait utiliser que les paramètres de type de son
// récepteur. Un « map » changeant de type de retour était donc impossible en
// méthode : il fallait une fonction libre, `Map(l, f)`, qui se lit à l'envers et
// ne se chaîne pas.
func (l List[E]) Apply[F any](f func(E) F) List[F] {
	r := make(List[F], len(l))
	for i, v := range l {
		r[i] = f(v)
	}
	return r
}

// Filter n'introduit aucun type nouveau : elle n'a pas besoin d'être générique.
// La méthode générique n'est justifiée que par un type de retour DIFFÉRENT.
func (l List[E]) Filter(keep func(E) bool) List[E] {
	r := make(List[E], 0, len(l))
	for _, v := range l {
		if keep(v) {
			r = append(r, v)
		}
	}
	return r
}

// itoa sert d'exemple de conversion E -> F (int -> string).
func itoa(i int) string { return strconv.Itoa(i) }

// ⚠️ Limite structurante : une interface NE PEUT PAS déclarer de méthode à
// paramètres de type, et une méthode générique ne peut pas en satisfaire une.
// L'itab d'un type (Ch. 33) est une table de pointeurs de fonctions figée à la
// compilation ; une méthode générique est une FAMILLE de fonctions, sans adresse
// unique à y ranger. Décommenter ce bloc donne :
//
//	interface method must have no type parameters
//
// type Mapper interface {
//	Apply[F any](f func(int) F) List[F]
// }
