package example_test

import (
	"fmt"

	"example.com/enumgen/example"
)

// Les méthodes String() ci-dessous sont GÉNÉRÉES par enumgen
// (fichier example_enum.go) ; ce test vérifie qu'elles se comportent bien.
func ExampleColor_String() {
	fmt.Println(example.ColorRed, example.ColorGreen, example.ColorBlue)
	// Valeur hors énumération : repli "Type(n)".
	fmt.Println(example.Color(99))
	// Output:
	// Red Green Blue
	// Color(99)
}

func ExamplePriority_String() {
	fmt.Println(example.PriorityLow, example.PriorityCritical)
	// Output:
	// Low Critical
}
