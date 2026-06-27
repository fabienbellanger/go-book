// Démonstrations du chapitre 9 : interfaces (satisfaction implicite, assertions,
// type switch, error, piège nil). Lancement : depuis code/, `go run ./ch09-interfaces`
package main

import (
	"fmt"
	"strings"
)

func main() {
	// =========================================================================
	// SATISFACTION IMPLICITE & valeur d'interface = (type, valeur)
	// =========================================================================

	var s Shape = Circle{Radius: 2} // Circle satisfait Shape sans le déclarer
	fmt.Printf("iface  : s=%v  type dynamique=%T  Area=%.4f\n", s, s, s.Area())

	// fmt utilise automatiquement String() (fmt.Stringer) — d'où "Circle(r=2)".
	shapes := []Shape{Circle{Radius: 1}, Rect{W: 3, H: 4}, Circle{Radius: 5}}
	fmt.Printf("iface  : shapes=%v\n", shapes)

	// =========================================================================
	// ACCEPTER UNE INTERFACE : code générique sans connaître le type concret
	// =========================================================================

	fmt.Printf("accept : aire totale=%.2f\n", totalArea(shapes))
	if b, ok := biggest(shapes); ok {
		fmt.Printf("accept : plus grande=%v (aire=%.2f)\n", b, b.Area())
	}

	// =========================================================================
	// TYPE ASSERTION : récupérer le type concret derrière l'interface
	// =========================================================================

	if c, ok := s.(Circle); ok { // comma-ok : pas de panique si ça échoue
		fmt.Printf("assert : s est un Circle de rayon %g\n", c.Radius)
	}
	if _, ok := s.(Rect); !ok {
		fmt.Println("assert : s n'est PAS un Rect (ok=false, aucune panique)")
	}

	// =========================================================================
	// TYPE SWITCH
	// =========================================================================

	for _, x := range []any{42, "go", Rect{W: 2, H: 2}, validateAge(-1), 3.14} {
		fmt.Printf("switch : %s\n", classify(x))
	}

	// =========================================================================
	// L'interface error
	// =========================================================================

	if err := validateAge(-5); err != nil {
		fmt.Println("error  : ", err) // appelle Error()
	}
	fmt.Printf("error  : validateAge(30) == nil ? %t\n", validateAge(30) == nil)

	// =========================================================================
	// ⚠️ LE PIÈGE : interface nil vs pointeur nil typé
	// =========================================================================

	err := typedNilError()
	fmt.Printf("piège  : typedNilError()==nil ? %t  (type dynamique=%T) <- NON nil !\n",
		err == nil, err)
	var realNil error
	fmt.Printf("piège  : interface réellement nil ==nil ? %t\n", realNil == nil)

	// =========================================================================
	// Interface standard : io.Writer (strings.Builder l'implémente)
	// =========================================================================

	var w strings.Builder // *strings.Builder satisfait io.Writer
	fmt.Fprintf(&w, "Go %d.%d", 1, 26)
	fmt.Printf("stdlib : écrit dans un io.Writer -> %q\n", w.String())
}
