// Démonstration des verbes de formatage du paquet fmt (cf. Annexe I).
// À lancer avec « go run ./annexe-I-fmt » depuis le dossier code/.
package main

import (
	"errors"
	"fmt"
)

// point n'implémente PAS Stringer : on observe la sortie brute de %v, %+v et %#v.
type point struct {
	X, Y int
}

// color implémente fmt.Stringer : %v et %s passent alors par String().
type color int

const (
	red color = iota
	green
	blue
)

// String rend une couleur lisible ; le cas par défaut évite une récursion
// infinie en n'utilisant PAS %v sur le receveur (cf. piège du chapitre).
func (c color) String() string {
	switch c {
	case red:
		return "rouge"
	case green:
		return "vert"
	case blue:
		return "bleu"
	default:
		return fmt.Sprintf("color(%d)", int(c))
	}
}

func main() {
	// --- Verbes généraux ---
	p := point{X: 3, Y: 4}
	fmt.Printf("%%v   -> %v\n", p)   // {3 4}
	fmt.Printf("%%+v  -> %+v\n", p)  // {X:3 Y:4}
	fmt.Printf("%%#v  -> %#v\n", p)  // main.point{X:3, Y:4}
	fmt.Printf("%%T   -> %T\n", p)   // main.point
	fmt.Printf("Stringer -> %v / %s\n", green, green) // vert / vert

	// --- Booléen ---
	fmt.Printf("%%t   -> %t\n", true)

	// --- Entiers ---
	fmt.Printf("%%d %%b %%o %%x %%X -> %d %b %o %x %X\n", 255, 255, 255, 255, 255)
	const han rune = 0x4E2D // le caractère 中
	fmt.Printf("%%c %%q %%U -> %c %q %U\n", han, han, han) // 中 '中' U+4E2D

	// --- Flottants ---
	fmt.Printf("%%f %%e %%g -> %f %e %g\n", 1234.5678, 1234.5678, 1234.5678)
	fmt.Printf("%%.2f -> %.2f\n", 1234.5678) // 1234.57

	// --- Chaînes & octets ---
	s := "Go café"
	fmt.Printf("%%s %%q %%x -> %s %q %x\n", s, s, s)

	// --- Pointeur (adresse non déterministe : non testée) ---
	fmt.Printf("%%p   -> %p\n", &p)

	// --- Largeur, précision, flags, indexation d'arguments ---
	fmt.Printf("largeur/flags -> [%6d] [%-6d] [%06d] [%+d]\n", 42, 42, 42, 42)
	fmt.Printf("précision str -> %.3s\n", "abcdef")       // abc
	fmt.Printf("largeur '*'   -> [%*d]\n", 6, 42)         // [    42]
	fmt.Printf("index args    -> %[2]d %[1]d\n", 7, 9)    // 9 7

	// --- Verbe d'erreur %w (uniquement avec fmt.Errorf) ---
	base := errors.New("disque plein")
	wrapped := fmt.Errorf("écriture du cache : %w", base)
	fmt.Printf("%%w    -> %v (unwrap == base ? %t)\n", wrapped, errors.Is(wrapped, base))

	// --- Erreurs de formatage : argument manquant ou verbe inadapté ---
	fmt.Printf("verbe inadapté -> %s\n", describeBadVerb())
}

// describeBadVerb illustre la sortie « %!verbe » produite quand le verbe ne
// correspond pas au type de l'argument (ici %d sur une chaîne). Le format est
// volontairement passé par une variable : sans cela, « go vet » refuserait de
// compiler ce contre-exemple pédagogique.
func describeBadVerb() string {
	verb := "%d"
	return fmt.Sprintf(verb, "texte") // -> %!d(string=texte)
}
