package main

import "fmt"

// ch12Amount représente un montant en centimes, pour éviter les erreurs
// d'arrondi des float64 sur de la monnaie. La règle d'export : seul le nom en
// MAJUSCULE initiale (Amount, String) est visible hors du paquet ; la
// convention « internal/ » (exercice 1) empêche même l'import depuis un autre
// module — c'est le compilateur qui l'impose, pas une simple convention.
type ch12Amount int64

// String formate le montant en euros. Sa sortie est verrouillée par un Example
// testable (exercice 2 : casser le format fait échouer `go test`).
func (a ch12Amount) String() string {
	sign := ""
	if a < 0 {
		sign, a = "-", -a
	}
	return fmt.Sprintf("%s%d.%02d €", sign, a/100, a%100)
}
