// Démonstrations du chapitre 15 : fonctions anonymes & closures.
// Lancement : depuis code/, `go run ./ch15-closures`
package main

// counter renvoie une closure À ÉTAT : la variable n est capturée PAR RÉFÉRENCE,
// donc partagée entre les appels successifs (ce n'est pas une copie).
func counter() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

// makeAdders illustre la PORTÉE PAR ITÉRATION (Go 1.22+) : chaque tour de boucle
// a SA propre variable i, donc chaque closure capture une valeur distincte.
// Avant 1.22, les trois closures partageaient la même i et renvoyaient toutes 3.
func makeAdders() []func() int {
	var fns []func() int
	for i := range 3 {
		fns = append(fns, func() int { return i })
	}
	return fns
}
