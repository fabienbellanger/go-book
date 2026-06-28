//go:build prod

package main

// featureName renvoie le libellé de la build de production.
// Compilé uniquement avec « go build -tags prod ». La ligne vide après la
// directive //go:build est OBLIGATOIRE : sans elle, ce n'est qu'un commentaire.
func featureName() string {
	return "prod (build -tags prod)"
}
