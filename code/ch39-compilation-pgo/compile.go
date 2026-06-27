package main

// add est minuscule : le compilateur l'inline (coût bien sous le budget ~80).
// Vérifiable avec `go build -gcflags=-m`.
func add(a, b int) int { return a + b }

// AddTwice expose un appel inliné : `inlining call to add`.
func AddTwice(n int) int { return add(n, n) }

// SumRange parcourt par range : le compilateur PROUVE que l'accès est sûr et
// élimine le contrôle de borne (bounds check elimination, BCE).
func SumRange(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}

// SumGather indexe par des valeurs externes (idx) : le compilateur ne peut PAS
// prouver que xs[i] est dans les bornes -> il conserve le contrôle.
// `go build -gcflags=-d=ssa/check_bce` affiche « Found IsInBounds » sur xs[i].
func SumGather(xs, idx []int) int {
	total := 0
	for _, i := range idx {
		total += xs[i]
	}
	return total
}

// SumHinted : un accès « témoin » en tête prouve la sûreté des suivants. Après
// `_ = xs[3]`, le compilateur élimine les contrôles de xs[0..3].
func SumHinted(xs []int) int {
	if len(xs) < 4 {
		return 0
	}
	_ = xs[3] // témoin : borne haute prouvée
	return xs[0] + xs[1] + xs[2] + xs[3]
}

// --- Cible PGO : un site d'appel POLYMORPHE ---

// Shape a deux implémentations : le site s.Area() ne peut donc PAS être
// dévirtualisé statiquement. Avec un profil (PGO) montrant qu'un type domine,
// le compilateur le dévirtualise et l'inline.
type Shape interface{ Area() float64 }

type Square struct{ Side float64 }

func (s Square) Area() float64 { return s.Side * s.Side }

type Circle struct{ R float64 }

func (c Circle) Area() float64 { return 3.14159 * c.R * c.R }

// TotalArea somme les aires : appel d'interface dans la boucle.
//
//go:noinline
func TotalArea(shapes []Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area() // site polymorphe -> candidat à la dévirtualisation PGO
	}
	return total
}
