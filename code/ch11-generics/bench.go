package main

// Ce fichier sert uniquement à la comparaison de performance « générique vs interface »
// du benchmark (cf. ⚡ Performance du chapitre). La version générique réutilise Sum.

// Valuer abstrait « quelque chose qui a une valeur entière » via une interface.
type Valuer interface{ Value() int }

// myInt satisfait Valuer. Stocké dans un []Valuer, chaque élément passe par le
// dispatch dynamique de l'interface.
type myInt int

func (i myInt) Value() int { return int(i) }

// sumIface additionne via l'interface : un appel indirect (dispatch) par élément.
func sumIface(xs []Valuer) int {
	acc := 0
	for _, x := range xs {
		acc += x.Value()
	}
	return acc
}
