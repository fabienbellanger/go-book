package main

import "strings"

// ch13Slugify transforme un titre en « slug » URL : minuscules, accents repliés,
// tout ce qui n'est pas [a-z0-9] réduit à un seul tiret, sans tiret aux bords.
//
// La fonction est IDEMPOTENTE : Slugify(Slugify(s)) == Slugify(s). C'est la
// propriété que le fuzzing cherche à violer (exercice 3) ; le repli d'accents
// (exercice 1) doit donc produire une sortie déjà « slugifiée ».
func ch13Slugify(s string) string {
	s = strings.ToLower(s)

	var b strings.Builder
	b.Grow(len(s))
	prevDash := true // true au départ pour manger les tirets de tête
	for _, r := range s {
		r = ch13FoldAccent(r)
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// ch13FoldAccent replie les accents français courants sur leur lettre nue.
func ch13FoldAccent(r rune) rune {
	switch r {
	case 'à', 'â', 'ä', 'á', 'ã':
		return 'a'
	case 'ç':
		return 'c'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'î', 'ï', 'í', 'ì':
		return 'i'
	case 'ô', 'ö', 'ó', 'ò', 'õ':
		return 'o'
	case 'ù', 'û', 'ü', 'ú':
		return 'u'
	default:
		return r
	}
}
