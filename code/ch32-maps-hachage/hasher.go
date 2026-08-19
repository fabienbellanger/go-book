package main

import "hash/maphash"

// StringSet est un ensemble minimaliste bâti sur une table de hachage MAISON
// (adressage ouvert, sondage linéaire), pour montrer à quoi sert le contrat
// hash/maphash.Hasher[T] (🆕 Go 1.27).
//
// En vrai code, on utiliserait `map[string]struct{}` : la map native est déjà
// une Swiss Table, déjà ensemencée, déjà optimisée. L'intérêt de Hasher n'est
// PAS de réimplémenter la map, mais de brancher un type sur une structure que
// la stdlib ne fournit pas (filtre de Bloom, cache LRU, ensemble probabiliste).
type StringSet struct {
	hasher maphash.Hasher[string] // le contrat : Hash + Equal
	seed   maphash.Seed
	slots  []string
	occupé []bool
	nombre int
}

// NewStringSet construit un ensemble de capacité `taille` (puissance de 2).
//
// La graine vient de maphash.MakeSeed() : elle est ALÉATOIRE à chaque
// démarrage du processus, exactement comme celle des maps natives. C'est ce qui
// protège des attaques par collisions choisies — et ce qui interdit de
// persister ces empreintes où que ce soit.
func NewStringSet(taille int) *StringSet {
	n := 8
	for n < taille*2 {
		n *= 2
	}
	return &StringSet{
		hasher: maphash.ComparableHasher[string]{}, // implémentation fournie
		seed:   maphash.MakeSeed(),
		slots:  make([]string, n),
		occupé: make([]bool, n),
	}
}

// hash calcule l'empreinte d'une clé via le Hasher.
//
// Le contrat impose une seule chose, mais elle est essentielle :
// si Equal(a, b) est vrai, alors Hash doit produire le même état pour a et b.
func (s *StringSet) hash(v string) uint64 {
	var h maphash.Hash
	h.SetSeed(s.seed)
	s.hasher.Hash(&h, v)
	return h.Sum64()
}

// Add insère v et renvoie true si la valeur était absente.
func (s *StringSet) Add(v string) bool {
	if s.nombre*2 >= len(s.slots) {
		s.grow()
	}
	masque := uint64(len(s.slots) - 1)
	for i := s.hash(v) & masque; ; i = (i + 1) & masque {
		if !s.occupé[i] {
			s.slots[i], s.occupé[i] = v, true
			s.nombre++
			return true
		}
		// Equal, pas == : c'est le Hasher qui définit l'égalité.
		if s.hasher.Equal(s.slots[i], v) {
			return false
		}
	}
}

// Contains indique si v est présent.
func (s *StringSet) Contains(v string) bool {
	masque := uint64(len(s.slots) - 1)
	for i := s.hash(v) & masque; s.occupé[i]; i = (i + 1) & masque {
		if s.hasher.Equal(s.slots[i], v) {
			return true
		}
	}
	return false
}

// Len renvoie le nombre d'éléments.
func (s *StringSet) Len() int { return s.nombre }

// grow double la table et réinsère tout (les positions dépendent du hachage).
func (s *StringSet) grow() {
	anciens, occupé := s.slots, s.occupé
	s.slots = make([]string, len(anciens)*2)
	s.occupé = make([]bool, len(anciens)*2)
	s.nombre = 0
	for i, v := range anciens {
		if occupé[i] {
			s.Add(v)
		}
	}
}
