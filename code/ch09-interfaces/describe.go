package main

import "fmt"

// totalArea ACCEPTE UNE INTERFACE : il fonctionne pour n'importe quel Shape, sans
// rien connaître du type concret. « Accept interfaces » est l'idiome Go.
func totalArea(shapes []Shape) float64 {
	var sum float64
	for _, s := range shapes {
		sum += s.Area()
	}
	return sum
}

// biggest renvoie la forme de plus grande aire (et false si la liste est vide).
func biggest(shapes []Shape) (Shape, bool) {
	if len(shapes) == 0 {
		return nil, false
	}
	best := shapes[0]
	for _, s := range shapes[1:] {
		if s.Area() > best.Area() {
			best = s
		}
	}
	return best, true
}

// classify illustre le TYPE SWITCH : on inspecte le type dynamique d'une valeur
// d'interface. Les cas concrets (int, string) précèdent les cas interface
// (Shape, error), car le premier cas qui correspond gagne.
func classify(x any) string {
	switch v := x.(type) {
	case nil:
		return "nil"
	case int:
		return fmt.Sprintf("int:%d", v)
	case string:
		return fmt.Sprintf("string:%q", v)
	case Shape:
		return fmt.Sprintf("shape:%.2f", v.Area())
	case error:
		return "error:" + v.Error()
	default:
		return fmt.Sprintf("autre:%T", v)
	}
}
