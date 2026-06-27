package main

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
