// Démonstrations du chapitre 7 : maps (comma-ok, set, itération) et strings (UTF-8,
// conversions, Builder, strconv). Lancement : depuis code/, `go run ./ch07-maps-strings`
package main

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

func main() {
	// =========================================================================
	// MAPS
	// =========================================================================

	// --- Création : littéral, make (avec capacité indicative).
	ages := map[string]int{"alice": 30, "bob": 25}
	scores := make(map[string]int, 8) // réserve de la place pour ~8 entrées
	scores["alice"] = 42

	// --- comma-ok : distinguer « absent » de « présent à la zero value ».
	if v, ok := ages["alice"]; ok {
		fmt.Printf("maps   : alice a %d ans\n", v)
	}
	v, ok := ages["zoe"]
	fmt.Printf("maps   : ages[\"zoe\"] -> v=%d ok=%t (absente, zero value)\n", v, ok)

	// --- delete et clear.
	delete(ages, "bob")
	fmt.Printf("maps   : après delete(bob) -> %v\n", ages)
	clear(scores)
	fmt.Printf("maps   : après clear(scores) -> len=%d (mais != nil : %t)\n",
		len(scores), scores != nil)

	// --- Itération NON ORDONNÉE : l'ordre change d'une exécution à l'autre.
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	fmt.Print("maps   : itération brute (ordre non garanti) : ")
	for k, v := range m {
		fmt.Printf("%s=%d ", k, v)
	}
	// --- Parade : trier les clés pour un parcours déterministe.
	fmt.Print("\nmaps   : clés triées : ")
	for _, k := range slices.Sorted(maps.Keys(m)) {
		fmt.Printf("%s=%d ", k, m[k])
	}
	fmt.Println()

	// --- Set : map[T]struct{} (struct{} ne pèse aucun octet).
	seen := map[string]struct{}{}
	for _, w := range []string{"go", "rust", "go"} {
		seen[w] = struct{}{}
	}
	_, isMember := seen["go"]
	fmt.Printf("maps   : set -> %d éléments distincts, contient \"go\" ? %t\n", len(seen), isMember)

	// --- Helpers de l'exemple.
	fmt.Printf("maps   : wordCount -> %v\n", wordCount("Go go GO rust"))
	fmt.Printf("maps   : uniqueSorted -> %v\n", uniqueSorted("le chat le chien le chat"))

	// =========================================================================
	// STRINGS
	// =========================================================================

	s := "café" // 'é' = U+00E9 = octets 0xC3 0xA9 (2 octets)

	// --- len = nombre d'OCTETS ; l'indexation renvoie un byte, pas une rune.
	fmt.Printf("\nstrings: %q -> len(octets)=%d runes=%d s[0]=%c s[3]=0x%X\n",
		s, len(s), utf8.RuneCountInString(s), s[0], s[3])

	// --- range sur string : index = position en OCTETS, valeur = rune décodée.
	fmt.Print("strings: range -> ")
	for i, r := range s {
		fmt.Printf("(i=%d %q) ", i, r)
	}
	fmt.Println("  <- l'index saute de 3 à 5 : 'é' occupe 2 octets")

	// --- Conversions explicites.
	fmt.Printf("strings: []byte=%v []rune=%v\n", []byte(s), []rune(s))
	fmt.Printf("strings: string(rune(0x1F680))=%q string([]byte{0x48,0x69})=%q\n",
		string(rune(0x1F680)), string([]byte{0x48, 0x69}))

	// --- Immutabilité : s[0] = 'C' ne compile pas. On reconstruit via []rune/Builder.
	// s[0] = 'C' // ERREUR : cannot assign to s[0]

	// --- Quelques helpers du package strings.
	fmt.Printf("strings: ToUpper=%s Split=%v Join=%q Contains=%t\n",
		strings.ToUpper("go"), strings.Split("a,b,c", ","),
		strings.Join([]string{"x", "y", "z"}, "/"), strings.Contains("golang", "lang"))

	// --- strconv : texte <-> nombres (avec gestion d'erreur).
	if n, err := strconv.Atoi("42"); err == nil {
		fmt.Printf("strconv: Atoi(\"42\")=%d Itoa(255)=%s\n", n, strconv.Itoa(255))
	}
	if _, err := strconv.Atoi("12x"); err != nil {
		fmt.Printf("strconv: Atoi(\"12x\") -> erreur: %v\n", err)
	}

	// --- strings.Builder : concaténer SANS réallouer à chaque étape.
	var b strings.Builder
	b.Grow(32) // préalloue (optionnel mais utile si on connaît la taille)
	for i := range 3 {
		fmt.Fprintf(&b, "ligne%d;", i)
	}
	fmt.Printf("builder: %q (len=%d octets)\n", b.String(), b.Len())

	// --- Helpers UTF-8 de l'exemple.
	fmt.Printf("strings: reverseString(%q)=%q\n", s, reverseString(s))
	fmt.Printf("strings: truncate(\"bonjour le monde\", 7)=%q\n", truncate("bonjour le monde", 7))
}
