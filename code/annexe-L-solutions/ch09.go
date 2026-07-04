package main

import (
	"fmt"
	"math"
)

// ch09Shape : un seul verbe. N'importe quel type qui définit Area() le satisfait
// IMPLICITEMENT — d'où l'ajout gratuit de Triangle (exercice 2).
type ch09Shape interface{ Area() float64 }

type ch09Rectangle struct{ W, H float64 }
type ch09Triangle struct{ Base, Height float64 }

func (r ch09Rectangle) Area() float64 { return r.W * r.H }
func (t ch09Triangle) Area() float64  { return t.Base * t.Height / 2 }

// ch09TotalArea accepte n'importe quelle Shape, y compris Triangle sans que la
// fonction ait à le connaître.
func ch09TotalArea(shapes []ch09Shape) float64 {
	var sum float64
	for _, s := range shapes {
		sum += s.Area()
	}
	return sum
}

// ch09Describe distingue un fmt.Stringer des autres types via une assertion de
// type (exercice 3).
func ch09Describe(x any) string {
	if s, ok := x.(fmt.Stringer); ok {
		return "Stringer: " + s.String()
	}
	return fmt.Sprintf("%T: %v", x, x)
}

// ch09Round arrondit à la décimale (utilitaire pour comparer des float64 en test
// sans dépendre d'une égalité binaire fragile).
func ch09Round(f float64) float64 { return math.Round(f*100) / 100 }
