// Package greeting construit des messages de bienvenue multilingues.
// (Le commentaire de doc d'un package commence par « Package <nom> ».)
package greeting

import "fmt"

// defaultLang est la langue de repli (identifiant non exporté : commence par une minuscule).
var defaultLang = "fr"

// salutations associe un code langue à sa formule de politesse.
// On l'initialise dans init() pour illustrer l'ordre d'initialisation du package.
var salutations map[string]string

// init prépare l'état du package. Il s'exécute APRÈS l'initialisation des
// variables de package et AVANT la première utilisation depuis l'extérieur.
func init() {
	fmt.Println("[init greeting]")

	salutations = map[string]string{
		"fr": "Bonjour",
		"en": "Hello",
		"es": "Hola",
	}
}

// Greet renvoie une salutation pour name dans la langue lang.
// Greet commence par une majuscule : il est EXPORTÉ (utilisable hors du package).
func Greet(lang, name string) string {
	return fmt.Sprintf("%s, %s !", hello(lang), name)
}

// hello renvoie la formule pour lang, avec repli sur la langue par défaut.
// hello commence par une minuscule : il est NON exporté (privé au package).
func hello(lang string) string {
	if s, ok := salutations[lang]; ok {
		return s
	}
	return salutations[defaultLang]
}
