package main

// Stack est un TYPE générique : le paramètre T se reporte sur ses méthodes. Une
// Stack[int] et une Stack[string] sont des types distincts, vérifiés à la compilation.
type Stack[T any] struct {
	items []T
}

// Push empile une valeur.
func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

// Pop dépile la dernière valeur. Renvoie (zero, false) si la pile est vide.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

// Len renvoie le nombre d'éléments.
func (s *Stack[T]) Len() int { return len(s.items) }

// Note : une méthode ne peut PAS introduire son propre paramètre de type.
// `func (s *Stack[T]) Map[R any](...)` serait refusé par le compilateur.
// Pour cela, on écrit une fonction libre (ex. Map dans constraints.go).
