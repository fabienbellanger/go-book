// Package analyze contient le « point chaud » du service wordstats : compter les
// mots d'un texte et en extraire les n plus fréquents.
//
// On en fournit DEUX implémentations volontairement comparables :
//
//   - TopWordsRegexp — la version « naïve » : découpage par expression régulière.
//     Lisible, mais coûteuse (CPU + allocations).
//   - TopWordsScan   — la version optimisée : un balayage d'octets, sans regexp,
//     qui exploite l'astuce « clé de map en string(buf) sans allocation ».
//
// C'est le matériau du Projet 7 : on profile la première, on lit le graphe de
// flamme, on optimise, et on chiffre le gain (voir RAPPORT.md).
package analyze

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Count associe un mot à son nombre d'occurrences.
type Count struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// splitRE découpe sur tout ce qui n'est ni lettre ni chiffre (Unicode).
// Compilée une fois ; c'est malgré tout le poste de coût dominant de la v1.
var splitRE = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// TopWordsRegexp — implémentation NAÏVE (v1).
//
// Deux sources de coût, bien visibles au profilage :
//   - strings.ToLower alloue une copie de TOUT le texte ;
//   - splitRE.Split alloue une tranche de toutes les sous-chaînes.
func TopWordsRegexp(text string, n int) []Count {
	counts := make(map[string]int)
	for _, w := range splitRE.Split(strings.ToLower(text), -1) {
		if w != "" {
			counts[w]++
		}
	}
	out := make([]Count, 0, len(counts))
	for w, c := range counts {
		out = append(out, Count{Word: w, Count: c})
	}
	return sortTop(out, n)
}

// TopWordsScan — implémentation OPTIMISÉE (v2).
//
// Trois leviers, tous visibles au profilage de la v1 :
//   - plus de regexp : on balaye le texte rune par rune ;
//   - plus de copie du texte entier (pas de strings.ToLower global) : on met le
//     mot courant en minuscule dans un tampon RÉUTILISÉ ;
//   - allocations de clés réduites au strict minimum grâce à map[string]*int :
//     la LECTURE counts[string(buf)] ne déclenche aucune allocation (motif
//     reconnu par le compilateur), donc un mot DÉJÀ vu se contente d'un *p++ ;
//     on n'alloue la chaîne-clé qu'une seule fois, à la première rencontre.
func TopWordsScan(text string, n int) []Count {
	counts := make(map[string]*int)
	buf := make([]byte, 0, 32)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		if p := counts[string(buf)]; p != nil { // lecture : aucune allocation
			*p++
		} else {
			c := 1
			counts[string(buf)] = &c // alloue la clé : une seule fois par mot distinct
		}
		buf = buf[:0]
	}
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		i += size
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf = utf8.AppendRune(buf, unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()

	out := make([]Count, 0, len(counts))
	for w, c := range counts {
		out = append(out, Count{Word: w, Count: *c})
	}
	return sortTop(out, n)
}

// sortTop trie les comptes (décroissant, puis lexicographique à égalité, pour un
// résultat déterministe) et renvoie les n premiers.
func sortTop(out []Count, n int) []Count {
	slices.SortFunc(out, func(a, b Count) int {
		if a.Count != b.Count {
			return b.Count - a.Count // décroissant
		}
		return strings.Compare(a.Word, b.Word)
	})
	if n >= 0 && n < len(out) {
		out = out[:n]
	}
	return out
}
