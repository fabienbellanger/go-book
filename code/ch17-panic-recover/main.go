package main

import "fmt"

func main() {
	// 1. recover -> erreur : la panique ne traverse pas safeCall.
	err := safeCall(func() { _ = divide(10, 0) })
	fmt.Println("safeCall(divide 10/0) :", err)
	fmt.Println("safeCall(ok)          :", safeCall(func() {})) // <nil>

	// 2. Pattern "Must" : valide ou panique.
	fmt.Println("mustPositive(42)      :", mustPositive(42))

	// 3. Recover sélectif : validationPanic -> erreur.
	fmt.Println("validate(30, 100)     :", validate(30, 100)) // <nil>
	fmt.Println("validate(-1, 100)     :", validate(-1, 100)) // champ "age" invalide

	// 4. Frontière de recover : une panique d'un handler devient un 500, le
	// "serveur" continue de tourner.
	h := recoverMiddleware(app)
	for _, p := range []string{"/home", "/boom", "/about"} {
		resp := h(Request{path: p})
		fmt.Printf("GET %-7s -> %d %s\n", p, resp.status, resp.body)
	}
}
