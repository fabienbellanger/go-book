// Démonstrations du chapitre 17 : panic, recover, frontière de récupération.
// Lancement : depuis code/, `go run ./ch17-panic-recover`
package main

import "fmt"

// safeCall exécute fn et convertit une éventuelle panique en erreur. recover ne
// fonctionne QUE dans une fonction différée, et seulement pour la goroutine qui
// panique. Le retour nommé err permet au defer de renseigner l'erreur.
func safeCall(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panique rattrapée : %v", r)
		}
	}()
	fn()
	return nil
}

// divide panique sur une division entière par zéro (erreur runtime). safeCall
// peut la transformer en erreur exploitable.
func divide(a, b int) int {
	return a / b
}

// mustPositive illustre le pattern "Must" : paniquer plutôt que renvoyer une
// erreur, réservé aux invariants/initialisations où un échec est un bug fatal.
func mustPositive(n int) int {
	if n <= 0 {
		panic(fmt.Sprintf("valeur attendue > 0, reçu %d", n))
	}
	return n
}

// validationPanic marque une panique "attendue" qu'on accepte de rattraper.
type validationPanic struct{ field string }

// checkPositive panique avec un validationPanic si n <= 0.
func checkPositive(field string, n int) {
	if n <= 0 {
		panic(validationPanic{field})
	}
}

// validate convertit SEULEMENT les validationPanic en erreur ; toute autre
// panique (vrai bug) est RE-DÉCLENCHÉE pour remonter au-dessus.
func validate(age, score int) (err error) {
	defer func() {
		switch r := recover().(type) {
		case nil:
			// aucune panique : rien à faire
		case validationPanic:
			err = fmt.Errorf("champ %q invalide", r.field)
		default:
			panic(r) // pas un validationPanic -> on laisse remonter
		}
	}()
	checkPositive("age", age)
	checkPositive("score", score)
	return nil
}
