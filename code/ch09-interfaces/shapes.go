package main

import (
	"fmt"
	"math"
)

// Shape est une INTERFACE : un ensemble de méthodes. Tout type qui possède ces
// méthodes la satisfait IMPLICITEMENT — aucun mot-clé « implements » à écrire.
type Shape interface {
	Area() float64
	Perimeter() float64
}

// Circle satisfait Shape (il a Area et Perimeter), sans le déclarer.
type Circle struct{ Radius float64 }

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }

// String fait en plus de Circle un fmt.Stringer : fmt l'utilisera automatiquement.
func (c Circle) String() string { return fmt.Sprintf("Circle(r=%g)", c.Radius) }

// Rect satisfait Shape lui aussi.
type Rect struct{ W, H float64 }

func (r Rect) Area() float64      { return r.W * r.H }
func (r Rect) Perimeter() float64 { return 2 * (r.W + r.H) }
func (r Rect) String() string     { return fmt.Sprintf("Rect(%g x %g)", r.W, r.H) }
