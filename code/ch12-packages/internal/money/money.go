// Package money fournit un type monétaire en centimes, sûr pour l'arithmétique
// (pas de float64, donc pas d'erreur d'arrondi). Il est placé sous internal/ : seul
// le code de ch12-packages peut l'importer — c'est la visibilité « à l'échelle du
// projet ».
package money

import "fmt"

// centsPerEuro est un détail d'implémentation : non exporté (minuscule), il reste
// invisible hors du package.
const centsPerEuro = 100

// Amount représente un montant en centimes d'euro. Travailler en entiers évite les
// erreurs d'arrondi des float64.
type Amount int64

// Euros construit un montant à partir d'euros et de centimes.
func Euros(euros, cents int64) Amount {
	return Amount(euros*centsPerEuro + cents)
}

// Add additionne deux montants.
func (a Amount) Add(b Amount) Amount { return a + b }

// String formate le montant (ex. "12,50 €"). En implémentant fmt.Stringer
// (Ch. 9), Amount s'affiche automatiquement avec fmt.
func (a Amount) String() string {
	sign := ""
	if a < 0 {
		sign, a = "-", -a
	}
	return fmt.Sprintf("%s%d,%02d €", sign, a/centsPerEuro, a%centsPerEuro)
}
