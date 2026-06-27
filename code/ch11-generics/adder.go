package main

// Adder est une contrainte AUTO-RÉFÉRENTIELLE (🆕 1.26) : elle se nomme elle-même.
// Elle exige une méthode Add prenant et renvoyant le PROPRE type qui la satisfait —
// le motif « F-bounded ». Avant 1.26, écrire `type Adder[A Adder[A]]` était refusé.
type Adder[A Adder[A]] interface {
	Add(A) A
}

// Vec2 satisfait Adder[Vec2] : son Add prend et renvoie un Vec2.
type Vec2 struct{ X, Y int }

func (v Vec2) Add(o Vec2) Vec2 { return Vec2{X: v.X + o.X, Y: v.Y + o.Y} }

// SumAll additionne n'importe quel type additionnable « sur lui-même ». Le compilateur
// garantit que acc.Add(x) est typé : pas d'interface, pas de dispatch dynamique.
func SumAll[A Adder[A]](xs []A) A {
	var acc A // zero value de A (Vec2{0, 0} ici)
	for _, x := range xs {
		acc = acc.Add(x)
	}
	return acc
}
