// Démonstrations du chapitre 3 : déclarations, zero values, conversions,
// constantes/iota et builtins (min/max/clear, new(expr)).
// Lancement : depuis code/, `go run ./ch03-basics`
package main

import "fmt"

func main() {
	// --- Zero values : une variable déclarée sans valeur reçoit la valeur nulle de son type.
	var (
		i int
		f float64
		b bool
		s string
	)
	fmt.Printf("zero values : i=%d f=%g b=%t s=%q\n", i, f, b, s)

	// --- Déclaration courte := avec inférence de type (hors niveau package).
	count := 42   // int
	ratio := 3.14 // float64
	name := "Go"  // string
	fmt.Printf("inférés     : %T, %T, %T\n", count, ratio, name)

	// --- Conversions EXPLICITES obligatoires : pas de coercition implicite.
	var big int64 = 9_000        // séparateur _ pour la lisibilité
	small := int32(big)          // conversion explicite int64 -> int32
	half := float64(small) / 2.0 // int32 -> float64 avant la division
	fmt.Printf("conversions : small=%d half=%.1f\n", small, half)

	// --- new(expr) : 🆕 Go 1.26, pointeur vers une valeur déjà initialisée.
	p := new(7) // *int pointant vers 7 (type inféré depuis l'expression)
	fmt.Printf("new(7)      : %T -> %d\n", p, *p)

	// --- Builtins 1.21.
	fmt.Printf("min/max     : %d, %g\n", min(count, 100), max(ratio, 2.0))
	seen := map[string]bool{"a": true, "b": true}
	clear(seen)
	fmt.Printf("clear(map)  : len=%d\n", len(seen))

	// --- Constantes typées + iota (voir sizes.go).
	for _, n := range []ByteSize{512, 1536, 5 * MB, 3 * GB} {
		fmt.Printf("humanSize(%-12d) = %s\n", int64(n), humanSize(n))
	}

	// --- Conversion sûre vs débordement silencieux (voir conv.go).
	if v, ok := toInt8(200); ok {
		fmt.Println("toInt8(200) =", v)
	} else {
		fmt.Println("toInt8(200) : débordement détecté (200 ne tient pas dans int8)")
	}
}
