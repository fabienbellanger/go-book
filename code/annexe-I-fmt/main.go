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

// celsius implémente Stringer sur un récepteur **pointeur** : seul *celsius
// satisfait fmt.Stringer, pas celsius (method set, cf. Ch. 09). %v sur une
// valeur n'invoque donc PAS String() ; %v sur &valeur l'invoque.
type celsius float64

func (c *celsius) String() string { return fmt.Sprintf("%.1f°C", float64(*c)) }

func main() {
	// --- Verbes généraux ---
	p := point{X: 3, Y: 4}
	fmt.Printf("%%v   -> %v\n", p)                    // {3 4}
	fmt.Printf("%%+v  -> %+v\n", p)                   // {X:3 Y:4}
	fmt.Printf("%%#v  -> %#v\n", p)                   // main.point{X:3, Y:4}
	fmt.Printf("%%T   -> %T\n", p)                    // main.point
	fmt.Printf("Stringer -> %v / %s\n", green, green) // vert / vert

	// Piège : Stringer sur récepteur pointeur n'est PAS dans le method set de
	// la valeur (🔁 Ch. 09). %v sur c ignore donc String() ; %v sur &c l'utilise.
	c := celsius(21.5)
	fmt.Printf("%%v valeur -> %v / %%v pointeur -> %v\n", c, &c) // 21.5 / 21.5°C

	// nil slice et slice vide sont INDISCERNABLES avec %v (idem pour les maps) :
	// seul `== nil` distingue les deux à l'exécution.
	var nilSlice []int
	var nilMap map[string]int
	fmt.Printf("%%v nil slice -> %v / vide -> %v\n", nilSlice, []int{})
	fmt.Printf("%%v nil map   -> %v / vide -> %v\n", nilMap, map[string]int{})

	// Les clés d'une map sont triées avant affichage par %v (sortie déterministe
	// depuis Go 1.12), pratique pour des tests sur la sortie.
	scores := map[string]int{"b": 2, "a": 1, "c": 3}
	fmt.Printf("%%v map (triée) -> %v\n", scores) // map[a:1 b:2 c:3]

	// --- Booléen ---
	fmt.Printf("%%t   -> %t\n", true)

	// --- Entiers ---
	fmt.Printf("%%d %%b %%o %%x %%X -> %d %b %o %x %X\n", 255, 255, 255, 255, 255)
	const han rune = 0x4E2D                                // le caractère 中
	fmt.Printf("%%c %%q %%U -> %c %q %U\n", han, han, han) // 中 '中' U+4E2D

	// --- Flottants ---
	fmt.Printf("%%f %%e %%g -> %f %e %g\n", 1234.5678, 1234.5678, 1234.5678)
	fmt.Printf("%%.2f -> %.2f\n", 1234.5678) // 1234.57

	// --- Chaînes & octets ---
	s := "Go café"
	fmt.Printf("%%s %%q %%x -> %s %q %x\n", s, s, s)

	// --- Pointeur (adresse non déterministe : non testée) ---
	fmt.Printf("%%p   -> %p\n", &p)

	// %v sur un pointeur vers struct/array/slice/map le déréférence (préfixe &) ;
	// sur un pointeur nil, %v affiche <nil> alors que %p affiche 0x0.
	fmt.Printf("%%v ptr -> %v\n", &p) // &{3 4}
	var pp *point
	fmt.Printf("%%v nil ptr -> %v / %%p nil ptr -> %p\n", pp, pp) // <nil> / 0x0

	// --- Largeur, précision, flags, indexation d'arguments ---
	fmt.Printf("largeur/flags -> [%6d] [%-6d] [%06d] [%+d]\n", 42, 42, 42, 42)
	fmt.Printf("précision str -> %.3s\n", "abcdef")    // abc
	fmt.Printf("largeur '*'   -> [%*d]\n", 6, 42)      // [    42]
	fmt.Printf("index args    -> %[2]d %[1]d\n", 7, 9) // 9 7

	// --- Verbe d'erreur %w (uniquement avec fmt.Errorf) ---
	base := errors.New("disque plein")
	wrapped := fmt.Errorf("écriture du cache : %w", base)
	fmt.Printf("%%w    -> %v (unwrap == base ? %t)\n", wrapped, errors.Is(wrapped, base))

	// Depuis Go 1.20, fmt.Errorf accepte PLUSIEURS %w : les deux erreurs restent
	// inspectables indépendamment par errors.Is/As (🔁 Ch. 10).
	errA := errors.New("erreur réseau")
	errB := errors.New("erreur disque")
	both := fmt.Errorf("échec combiné : %w / %w", errA, errB)
	fmt.Printf("%%w x2  -> Is(A)=%t Is(B)=%t\n", errors.Is(both, errA), errors.Is(both, errB))

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
