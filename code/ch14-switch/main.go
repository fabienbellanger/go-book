// Démonstrations du chapitre 14 : switch d'expression, tagless, init, fallthrough,
// type switch, et switch vs map. Lancement : depuis code/, `go run ./ch14-switch`
package main

import "fmt"

func main() {
	// switch tagless (intervalles) et d'expression (cas multiples) :
	fmt.Println("grade(85)       :", grade(85))
	fmt.Println("dayKind(samedi) :", dayKind("samedi"))
	fmt.Println("dayKind(mardi)  :", dayKind("mardi"))
	fmt.Printf("nameStatus(\"\")  : %s\n", nameStatus(""))

	// fallthrough : héritage des droits.
	fmt.Println("caps(admin)     :", capabilities("admin"))
	fmt.Println("caps(editor)    :", capabilities("editor"))
	fmt.Println("caps(viewer)    :", capabilities("viewer"))

	// type switch :
	fmt.Println("describe(42)    :", describe(42))
	fmt.Println("describe(\"go\")  :", describe("go"))
	fmt.Println("describe(3.14)  :", describe(3.14))
	fmt.Println("describe(nil)   :", describe(nil))

	// switch vs map : même résultat, codegen différent.
	fmt.Println("levelFromString :", levelFromString("warn"), "(switch)")
	fmt.Println("levelFromMap    :", levelFromMap("warn"), "(map)")
	fmt.Println("levelFromInt(5) :", levelFromInt(5), "(jump table)")
}
