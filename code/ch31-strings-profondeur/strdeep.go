package main

import (
	"strings"
	"unicode/utf8"
	"unique"
)

// ByteVsRune renvoie la taille d'une chaîne en OCTETS (len) et en RUNES (points de
// code Unicode). Les deux diffèrent dès qu'un caractère dépasse l'ASCII.
func ByteVsRune(s string) (bytes, runes int) {
	return len(s), utf8.RuneCountInString(s)
}

// RuneWidths renvoie, pour chaque rune de s, son index d'OCTET et sa largeur en
// octets. `range` sur une string décode l'UTF-8 et itère par rune, pas par octet :
// l'index avance donc de la largeur de chaque rune.
func RuneWidths(s string) [][2]int {
	out := make([][2]int, 0, len(s))
	for i, r := range s {
		out = append(out, [2]int{i, utf8.RuneLen(r)})
	}
	return out
}

// JoinCSV concatène des éléments via strings.Builder avec préallocation : Grow
// réserve la taille finale d'un coup, évitant les réallocations du backing interne.
func JoinCSV(items []string) string {
	size := 0
	for _, s := range items {
		size += len(s) + 1 // +1 pour la virgule
	}
	var b strings.Builder
	b.Grow(size)
	for i, s := range items {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s)
	}
	return b.String()
}

// ToUpperASCII met en majuscules les lettres ASCII. Comme une string est IMMUABLE,
// on passe par []byte pour modifier : une copie à l'aller (string->[]byte) et une au
// retour ([]byte->string).
func ToUpperASCII(s string) string {
	b := []byte(s) // copie n°1 : le backing de string est en lecture seule
	for i := range b {
		if c := b[i]; c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b) // copie n°2
}

// Symbol est une chaîne internée : deux Symbol de contenu égal partagent le même
// backing canonique et se comparent en comparant un seul pointeur.
type Symbol = unique.Handle[string]

// Intern renvoie le handle canonique d'une chaîne (interning, Go 1.23).
func Intern(s string) Symbol { return unique.Make(s) }

// CountDistinct compte les valeurs distinctes d'une liste via interning : les handles
// sont comparables et utilisables comme clés de map sans rehacher la chaîne entière.
func CountDistinct(values []string) int {
	seen := make(map[Symbol]struct{})
	for _, v := range values {
		seen[Intern(v)] = struct{}{}
	}
	return len(seen)
}
