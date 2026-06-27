package main

import (
	"fmt"
	"math"
	"reflect"
)

// Shape : interface non vide -> valeur représentée par un iface (*itab, *data).
type Shape interface{ Area() float64 }

type Circle struct{ R float64 }

func (c Circle) Area() float64 { return math.Pi * c.R * c.R }

type Rectangle struct{ W, H float64 }

func (r Rectangle) Area() float64 { return r.W * r.H }

// TotalArea additionne les aires via DISPATCH DYNAMIQUE : pour chaque Shape, la
// méthode réellement appelée est celle du type concret, trouvée dans l'itab.
func TotalArea(shapes []Shape) float64 {
	var sum float64
	for _, s := range shapes {
		sum += s.Area()
	}
	return sum
}

// Describe inspecte le type concret par un type switch — sans allocation.
func Describe(s Shape) string {
	switch v := s.(type) {
	case Circle:
		return fmt.Sprintf("cercle r=%g", v.R)
	case Rectangle:
		return fmt.Sprintf("rectangle %gx%g", v.W, v.H)
	default:
		return "forme inconnue"
	}
}

// --- Piège de l'interface nil non-nil ---

type myError struct{ msg string }

func (e *myError) Error() string { return e.msg }

// FailBuggy renvoie un *myError (éventuellement nil) dans une interface error. Même
// quand le pointeur est nil, l'interface renvoyée porte un TYPE (*myError) : elle
// n'est donc PAS égale à nil. C'est le piège classique « interface nil non-nil ».
func FailBuggy(ok bool) error {
	var e *myError // pointeur nil
	if !ok {
		e = &myError{"échec"}
	}
	return e // si ok==true, e est nil MAIS l'interface != nil
}

// FailCorrect renvoie explicitement l'interface nil : aucun type parasite.
func FailCorrect(ok bool) error {
	if ok {
		return nil
	}
	return &myError{"échec"}
}

// --- Boxing ---

var sink any

// BoxValue place une valeur dans une interface vide (boxing). Les petits entiers
// (0..255) sont mis en cache par le runtime (0 alloc) ; les autres allouent.
func BoxValue(v any) { sink = v }

// --- reflect.TypeAssert (Go 1.25) ---

// AsCircle extrait un Circle d'une Shape via reflect.TypeAssert, qui rend le type
// concret SANS passer par Value.Interface() (lequel re-boxe la valeur).
func AsCircle(s Shape) (Circle, bool) {
	return reflect.TypeAssert[Circle](reflect.ValueOf(s))
}
