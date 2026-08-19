package main

import "strings"

// reverseString inverse une chaîne EN RESPECTANT les caractères Unicode.
//
// ⚠️ Inverser octet par octet casserait les runes multi-octets (UTF-8). On convertit
// donc en []rune (un élément = un point de code), on inverse, puis on reconvertit.
func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// truncate raccourcit s à `max` RUNES (pas octets), en ajoutant « … » si coupé.
//
// Travailler en runes évite de trancher au milieu d'un caractère multi-octets, ce qui
// produirait de l'UTF-8 invalide. La comparaison utilise le nombre de runes, pas len(s).
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// splitName découpe un nom de fichier en (base, extension) autour du DERNIER
// point, avec strings.CutLast (🆕 Go 1.27).
//
// Avant 1.27, il fallait LastIndex + tranchage manuel — et ne pas oublier de
// tester le -1 :
//
//	if i := strings.LastIndex(nom, "."); i >= 0 {
//		base, ext = nom[:i], nom[i+1:]
//	}
//
// CutLast rend le cas « pas de séparateur » explicite via son booléen, comme
// strings.Cut (1.18) le fait pour la PREMIÈRE occurrence.
func splitName(nom string) (base, ext string, ok bool) {
	return strings.CutLast(nom, ".")
}

// firstKeyValue illustre la symétrie : Cut coupe à la PREMIÈRE occurrence,
// CutLast à la DERNIÈRE. Sur "a=b=c", Cut donne ("a", "b=c") et CutLast
// ("a=b", "c").
func firstKeyValue(pair string) (cle, valeur string, ok bool) {
	return strings.Cut(pair, "=")
}
