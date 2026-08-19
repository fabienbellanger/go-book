package set_test

import (
	"slices"
	"strconv"
	"testing"

	"example.com/gends/set"
)

func TestAddRemoveContains(t *testing.T) {
	s := set.New[string]()
	if got := s.Add("a", "b", "a"); got != 2 {
		t.Errorf("Add a,b,a = %d nouveaux, voulu 2", got)
	}
	if !s.Contains("a") || s.Contains("z") {
		t.Error("Contains incohérent")
	}
	if got := s.Remove("a", "z"); got != 1 {
		t.Errorf("Remove a,z = %d retirés, voulu 1", got)
	}
	if s.Len() != 1 {
		t.Errorf("Len = %d, voulu 1", s.Len())
	}
}

func TestSetOperations(t *testing.T) {
	a := set.Of(1, 2, 3, 4)
	b := set.Of(3, 4, 5)

	tests := []struct {
		name string
		got  []int
		want []int
	}{
		{"union", set.Sorted(a.Union(b)), []int{1, 2, 3, 4, 5}},
		{"intersect", set.Sorted(a.Intersect(b)), []int{3, 4}},
		{"difference", set.Sorted(a.Difference(b)), []int{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !slices.Equal(tt.got, tt.want) {
				t.Errorf("got %v, voulu %v", tt.got, tt.want)
			}
		})
	}

	// Les opérandes ne doivent pas être modifiés.
	if !slices.Equal(set.Sorted(a), []int{1, 2, 3, 4}) {
		t.Error("a a été modifié par une opération ensembliste")
	}
}

func TestEqualAndSubset(t *testing.T) {
	a := set.Of(1, 2, 3)
	if !a.Equal(set.Of(3, 2, 1)) {
		t.Error("Equal doit ignorer l'ordre")
	}
	if !set.Of(1, 2).IsSubsetOf(a) {
		t.Error("{1,2} doit être sous-ensemble de {1,2,3}")
	}
	if a.IsSubsetOf(set.Of(1, 2)) {
		t.Error("{1,2,3} ne doit pas être sous-ensemble de {1,2}")
	}
}

func TestCloneIsIndependent(t *testing.T) {
	a := set.Of(1, 2)
	c := a.Clone()
	c.Add(3)
	if a.Contains(3) {
		t.Error("Clone doit être indépendant de l'original")
	}
}

func TestAllIteration(t *testing.T) {
	a := set.Of(10, 20, 30)
	var got []int
	for v := range a.All() {
		got = append(got, v)
	}
	slices.Sort(got)
	if !slices.Equal(got, []int{10, 20, 30}) {
		t.Errorf("All a produit %v", got)
	}
}

func BenchmarkAdd(b *testing.B) {
	s := set.New[int]()
	for i := 0; b.Loop(); i++ {
		s.Add(i & 0xffff)
	}
}

func BenchmarkContains(b *testing.B) {
	s := set.New[int]()
	for i := range 1000 {
		s.Add(i)
	}
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		_ = s.Contains(i & 0x3ff)
	}
}

// --- Méthode générique (Go 1.27) ---

// TestMap couvre Set.Map, une méthode qui déclare son PROPRE paramètre de type.
func TestMap(t *testing.T) {
	s := set.Of(1, 2, 3, 4)

	// int -> string : le type de sortie change, d'où le besoin d'une méthode
	// générique. U est inféré depuis la fonction passée.
	noms := s.Map(strconv.Itoa)
	if got, want := set.Sorted(noms), []string{"1", "2", "3", "4"}; !slices.Equal(got, want) {
		t.Errorf("Map(strconv.Itoa) = %v ; attendu %v", got, want)
	}

	// U peut aussi être explicite.
	doubles := s.Map[int](func(i int) int { return i * 2 })
	if got, want := set.Sorted(doubles), []int{2, 4, 6, 8}; !slices.Equal(got, want) {
		t.Errorf("Map[int](x2) = %v ; attendu %v", got, want)
	}
}

// TestMapFusionne : Map peut RÉDUIRE la taille de l'ensemble, puisque deux
// antécédents distincts peuvent avoir la même image.
func TestMapFusionne(t *testing.T) {
	s := set.Of(-2, -1, 0, 1, 2)

	valeursAbsolues := s.Map(func(i int) int {
		if i < 0 {
			return -i
		}
		return i
	})

	if got, want := set.Sorted(valeursAbsolues), []int{0, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("Map(abs) = %v ; attendu %v (les opposés fusionnent)", got, want)
	}
	if valeursAbsolues.Len() >= s.Len() {
		t.Errorf("Len = %d ; attendu strictement moins que %d", valeursAbsolues.Len(), s.Len())
	}
}

// TestFilterEtChainage montre l'intérêt pratique de la méthode : le chaînage.
// Avec des fonctions libres, il faudrait écrire Map(Filter(s, pair), itoa).
func TestFilterEtChainage(t *testing.T) {
	s := set.Of(1, 2, 3, 4, 5, 6)

	got := s.
		Filter(func(i int) bool { return i%2 == 0 }).
		Map(strconv.Itoa)

	if want := []string{"2", "4", "6"}; !slices.Equal(set.Sorted(got), want) {
		t.Errorf("Filter puis Map = %v ; attendu %v", set.Sorted(got), want)
	}
}
