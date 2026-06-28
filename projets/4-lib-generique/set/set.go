// Package set fournit un ensemble générique d'éléments distincts, bâti sur une
// map.
//
// Le type zéro [Set] n'est pas prêt à l'emploi : construire un ensemble avec
// [New] ou [Of]. Set n'est pas sûr pour un usage concurrent ; protéger les
// accès avec un verrou si plusieurs goroutines y touchent.
//
// L'ordre d'itération n'est pas spécifié (comme pour une map) : utiliser
// [Sorted] pour un résultat déterministe.
package set

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// Set est un ensemble d'éléments distincts de type T. La contrainte comparable
// suffit : c'est exactement ce qu'exige une clé de map.
type Set[T comparable] struct {
	m map[T]struct{}
}

// New crée un ensemble vide.
func New[T comparable]() *Set[T] {
	return &Set[T]{m: make(map[T]struct{})}
}

// Of crée un ensemble contenant les éléments donnés (doublons fusionnés).
func Of[T comparable](items ...T) *Set[T] {
	s := &Set[T]{m: make(map[T]struct{}, len(items))}
	s.Add(items...)
	return s
}

// Add insère des éléments et renvoie le nombre d'éléments réellement nouveaux.
func (s *Set[T]) Add(items ...T) int {
	added := 0
	for _, it := range items {
		if _, ok := s.m[it]; !ok {
			s.m[it] = struct{}{}
			added++
		}
	}
	return added
}

// Remove retire des éléments et renvoie le nombre réellement retirés.
func (s *Set[T]) Remove(items ...T) int {
	removed := 0
	for _, it := range items {
		if _, ok := s.m[it]; ok {
			delete(s.m, it)
			removed++
		}
	}
	return removed
}

// Contains indique si v appartient à l'ensemble.
func (s *Set[T]) Contains(v T) bool {
	_, ok := s.m[v]
	return ok
}

// Len renvoie le cardinal de l'ensemble.
func (s *Set[T]) Len() int { return len(s.m) }

// All renvoie un itérateur sur les éléments (ordre non spécifié).
//
// L'itérateur est une [iter.Seq] (Go 1.23) : « for v := range s.All() ».
func (s *Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for k := range s.m {
			if !yield(k) {
				return
			}
		}
	}
}

// Clone renvoie une copie indépendante de l'ensemble.
func (s *Set[T]) Clone() *Set[T] {
	return &Set[T]{m: maps.Clone(s.m)}
}

// Union renvoie un nouvel ensemble contenant les éléments de s ou de other.
// Les opérandes ne sont pas modifiés.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	out := s.Clone()
	for k := range other.m {
		out.m[k] = struct{}{}
	}
	return out
}

// Intersect renvoie un nouvel ensemble contenant les éléments présents dans s
// et dans other.
func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	// On itère sur le plus petit ensemble : moins de tests d'appartenance.
	small, big := s, other
	if big.Len() < small.Len() {
		small, big = big, small
	}
	out := New[T]()
	for k := range small.m {
		if _, ok := big.m[k]; ok {
			out.m[k] = struct{}{}
		}
	}
	return out
}

// Difference renvoie un nouvel ensemble contenant les éléments de s absents de
// other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	out := New[T]()
	for k := range s.m {
		if _, ok := other.m[k]; !ok {
			out.m[k] = struct{}{}
		}
	}
	return out
}

// Equal indique si s et other contiennent exactement les mêmes éléments.
func (s *Set[T]) Equal(other *Set[T]) bool {
	if s.Len() != other.Len() {
		return false
	}
	for k := range s.m {
		if _, ok := other.m[k]; !ok {
			return false
		}
	}
	return true
}

// IsSubsetOf indique si tous les éléments de s appartiennent à other.
func (s *Set[T]) IsSubsetOf(other *Set[T]) bool {
	if s.Len() > other.Len() {
		return false
	}
	for k := range s.m {
		if _, ok := other.m[k]; !ok {
			return false
		}
	}
	return true
}

// Sorted renvoie les éléments de s triés en ordre croissant.
//
// C'est une fonction libre (et non une méthode) car elle exige une contrainte
// plus forte que [Set] : [cmp.Ordered]. Un Set[T] n'a besoin que de comparable,
// mais le tri suppose un ordre total — d'où la contrainte ajoutée ici.
func Sorted[T cmp.Ordered](s *Set[T]) []T {
	out := make([]T, 0, s.Len())
	for k := range s.m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
