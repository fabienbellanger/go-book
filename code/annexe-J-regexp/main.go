// Command annexe-J-regexp illustre le paquet regexp : moteur RE2 (temps
// linéaire, sans backtracking), compilation, recherche, sous-groupes nommés,
// remplacement et découpage. Tous les motifs sont compilés UNE seule fois, au
// niveau du paquet — le coût de compilation ne doit jamais être payé en boucle.
package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Motifs compilés une fois pour toutes. MustCompile panique si le motif est
// invalide : parfait à l'initialisation d'un paquet (l'erreur est un bug de
// programmation, pas une condition d'exécution).
var (
	// reWord repère un « mot » (au moins une lettre/chiffre/underscore).
	reWord = regexp.MustCompile(`\w+`)

	// reDate capture une date ISO via des sous-groupes NOMMÉS.
	reDate = regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)

	// reSpaces découpe sur une ou plusieurs espaces (gourmand).
	reSpaces = regexp.MustCompile(`\s+`)

	// reLogLevel, insensible à la casse et ancré, valide un niveau de log
	// occupant TOUTE la chaîne (^...$), pas une simple sous-chaîne.
	reLogLevel = regexp.MustCompile(`(?i)^(debug|info|warn|error)$`)
)

// firstWord renvoie le premier mot trouvé, ou "" si aucun.
func firstWord(s string) string {
	return reWord.FindString(s) // "" si pas de correspondance
}

// allWords renvoie tous les mots (au plus n, -1 = tous).
func allWords(s string) []string {
	return reWord.FindAllString(s, -1) // nil si aucun
}

// parseDate extrait année, mois, jour d'une date ISO via les groupes nommés.
// Renvoie ok == false si la chaîne ne contient pas de date.
func parseDate(s string) (year, month, day string, ok bool) {
	m := reDate.FindStringSubmatch(s) // m[0] = tout, m[1..] = sous-groupes
	if m == nil {
		return "", "", "", false
	}
	// SubexpNames() donne le nom de chaque groupe (index 0 = groupe entier, "").
	names := reDate.SubexpNames()
	byName := map[string]string{}
	for i, name := range names {
		if name != "" {
			byName[name] = m[i]
		}
	}
	return byName["year"], byName["month"], byName["day"], true
}

// isLogLevel indique si s est EXACTEMENT un niveau de log connu (casse ignorée).
func isLogLevel(s string) bool {
	return reLogLevel.MatchString(s)
}

// redactDigits remplace chaque suite de chiffres par des astérisques de même
// longueur, via une fonction de remplacement (ReplaceAllStringFunc).
func redactDigits(s string) string {
	reDigits := regexp.MustCompile(`\d+`)
	return reDigits.ReplaceAllStringFunc(s, func(match string) string {
		return strings.Repeat("*", len(match)) // autant d'astérisques que de chiffres
	})
}

// swapDate réécrit une date "YYYY-MM-DD" en "DD/MM/YYYY" avec les références
// $name dans le motif de remplacement.
func swapDate(s string) string {
	return reDate.ReplaceAllString(s, "${day}/${month}/${year}")
}

// splitFields découpe une ligne sur les espaces.
func splitFields(s string) []string {
	return reSpaces.Split(s, -1)
}

func main() {
	fmt.Println("premier mot :", firstWord("  Go 1.26 !"))
	fmt.Println("tous les mots :", allWords("un deux trois"))

	if y, mo, d, ok := parseDate("release 2026-07-04 ok"); ok {
		fmt.Printf("date : %s / %s / %s\n", y, mo, d)
	}

	fmt.Println("niveau valide :", isLogLevel("INFO"), isLogLevel("verbose"))
	fmt.Println("censuré :", redactDigits("carte 4242 4242"))
	fmt.Println("date FR :", swapDate("2026-07-04"))
	fmt.Printf("champs : %q\n", splitFields("a  b   c"))
}
