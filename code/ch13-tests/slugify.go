package main

import "strings"

// Slugify transforme un texte en « slug » : minuscules, lettres et chiffres ASCII
// conservés, toute autre séquence remplacée par un unique '-', sans tiret en bordure.
// Ex. "  Go 1.26 : Top!  " -> "go-1-26-top".
//
// La fonction est idempotente : Slugify(Slugify(s)) == Slugify(s) — propriété
// vérifiée par le fuzzing dans slugify_test.go.
func Slugify(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			// On n'insère un tiret que s'il y a déjà du contenu et que le
			// caractère précédent n'en était pas un (pas de tiret doublé ni en tête).
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-") // retire un éventuel tiret final
}
