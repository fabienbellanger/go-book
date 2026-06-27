// Tests « boîte noire » (package money_test) : on importe money comme le ferait un
// vrai utilisateur, via son chemin complet. Cela montre aussi le chemin d'import
// d'un package internal/.
package money_test

import (
	"fmt"
	"testing"

	"example.com/gobook/ch12-packages/internal/money"
)

// ExampleAmount_String est À LA FOIS de la documentation et un test : le commentaire
// // Output: est comparé à la sortie réelle par `go test`.
func ExampleAmount_String() {
	fmt.Println(money.Euros(12, 50))
	// Output: 12,50 €
}

func TestAdd(t *testing.T) {
	got := money.Euros(10, 0).Add(money.Euros(2, 50))
	want := money.Euros(12, 50)
	if got != want {
		t.Errorf("Add = %s ; attendu %s", got, want)
	}
}

func TestStringNegative(t *testing.T) {
	got := money.Euros(-3, 5).String() // -3*100 + 5 = -295 centimes
	if got != "-2,95 €" {
		t.Errorf("String = %q ; attendu \"-2,95 €\"", got)
	}
}
