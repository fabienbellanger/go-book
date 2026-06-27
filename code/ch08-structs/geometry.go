package main

import "math"

// Point est un point 2D. Ses méthodes utilisent un récepteur VALEUR : le struct est
// petit, le copier est bon marché, et on n'a jamais besoin de muter le récepteur.
type Point struct {
	X, Y float64
}

// Distance renvoie la distance euclidienne entre p et q.
func (p Point) Distance(q Point) float64 {
	return math.Hypot(q.X-p.X, q.Y-p.Y)
}

// Rectangle est COMPOSÉ de deux Points : un struct peut contenir d'autres structs
// (composition par valeur, pas d'héritage).
type Rectangle struct {
	Min, Max Point
}

func (r Rectangle) Width() float64  { return r.Max.X - r.Min.X }
func (r Rectangle) Height() float64 { return r.Max.Y - r.Min.Y }

// Area combine les autres méthodes : une méthode peut en appeler d'autres du type.
func (r Rectangle) Area() float64 {
	return r.Width() * r.Height()
}
