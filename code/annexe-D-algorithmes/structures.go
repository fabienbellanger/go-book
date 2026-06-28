package main

// Stack est une pile LIFO générique sur tranche.
type Stack[T any] struct {
	items []T
}

// Push empile v.
func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

// Pop dépile le dernier élément. Renvoie false si la pile est vide.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items[len(s.items)-1] = zero // libère la référence (sinon fuite mémoire potentielle)
	s.items = s.items[:len(s.items)-1]
	return v, true
}

// Len renvoie le nombre d'éléments.
func (s *Stack[T]) Len() int { return len(s.items) }

// Queue est une file FIFO générique sur tranche.
type Queue[T any] struct {
	items []T
}

// Enqueue ajoute v en fin de file.
func (q *Queue[T]) Enqueue(v T) { q.items = append(q.items, v) }

// Dequeue retire l'élément de tête. Renvoie false si la file est vide.
func (q *Queue[T]) Dequeue() (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	v := q.items[0]
	q.items[0] = zero
	q.items = q.items[1:]
	return v, true
}

// Len renvoie le nombre d'éléments.
func (q *Queue[T]) Len() int { return len(q.items) }

// UnionFind (union-find / ensembles disjoints) avec COMPRESSION DE CHEMIN et
// UNION PAR RANG : Find et Union s'exécutent en temps quasi constant (amorti,
// inverse de la fonction d'Ackermann). Classique pour les composantes connexes.
type UnionFind struct {
	parent []int
	rank   []int
}

// NewUnionFind crée n éléments, chacun dans son propre ensemble.
func NewUnionFind(n int) *UnionFind {
	uf := &UnionFind{parent: make([]int, n), rank: make([]int, n)}
	for i := range uf.parent {
		uf.parent[i] = i // chaque élément est d'abord son propre représentant
	}
	return uf
}

// Find renvoie le représentant de l'ensemble de x (avec compression de chemin).
func (uf *UnionFind) Find(x int) int {
	for uf.parent[x] != x {
		uf.parent[x] = uf.parent[uf.parent[x]] // compression « par moitié »
		x = uf.parent[x]
	}
	return x
}

// Union fusionne les ensembles de x et y. Renvoie false s'ils étaient déjà unis.
func (uf *UnionFind) Union(x, y int) bool {
	rx, ry := uf.Find(x), uf.Find(y)
	if rx == ry {
		return false
	}
	if uf.rank[rx] < uf.rank[ry] { // rattache toujours le plus petit arbre
		rx, ry = ry, rx
	}
	uf.parent[ry] = rx
	if uf.rank[rx] == uf.rank[ry] {
		uf.rank[rx]++
	}
	return true
}

// Connected indique si x et y sont dans le même ensemble.
func (uf *UnionFind) Connected(x, y int) bool { return uf.Find(x) == uf.Find(y) }
