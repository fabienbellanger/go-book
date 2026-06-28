package main

// Démonstration « interface vs générique ».
//
// La même méthode (doubler un entier) est appelée de deux façons :
//   - via une interface : l'appel passe par la table de méthodes (dispatch
//     dynamique), opaque au compilateur ;
//   - via un type paramétré instancié sur un type concret : le compilateur
//     monomorphise et peut dévirtualiser/inliner l'appel.
//
// Voir Ch. 11 (génériques) et Ch. 33 (interfaces en profondeur).

// doubler est l'interface commune.
type doubler interface {
	Double(int) int
}

// intDoubler est une implémentation concrète, sans état.
type intDoubler struct{}

func (intDoubler) Double(n int) int { return n * 2 }

// viaInterface somme le double de chaque élément via le dispatch dynamique.
func viaInterface(d doubler, xs []int) int {
	total := 0
	for _, x := range xs {
		total += d.Double(x)
	}
	return total
}

// viaGeneric fait la même chose, mais d est un paramètre de type contraint par
// doubler : instancié sur intDoubler, l'appel devient statique.
func viaGeneric[T doubler](d T, xs []int) int {
	total := 0
	for _, x := range xs {
		total += d.Double(x)
	}
	return total
}
