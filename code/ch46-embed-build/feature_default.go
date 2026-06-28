//go:build !prod

package main

// featureName renvoie le libellé de la build de développement.
// Ce fichier est compilé SAUF si le tag « prod » est présent
// (« go build -tags prod » sélectionne feature_prod.go à la place).
func featureName() string {
	return "dev (build par défaut)"
}
