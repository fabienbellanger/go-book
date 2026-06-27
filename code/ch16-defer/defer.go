// Démonstrations du chapitre 16 : defer (LIFO, évaluation des arguments, retours
// nommés). Lancement : depuis code/, `go run ./ch16-defer`
package main

// lifoOrder enregistre des defers pour 0, 1, 2 mais ils s'exécutent en ordre
// INVERSE (LIFO) : 2, 1, 0. Chaque closure capture sa propre i (portée par
// itération, Go 1.22+) et écrit dans le retour nommé out.
func lifoOrder() (out []int) {
	for i := range 3 {
		defer func() { out = append(out, i) }()
	}
	return // c'est ICI que les defers s'exécutent : 2, puis 1, puis 0
}

// evalContrast oppose les deux moments d'évaluation :
//   - l'ARGUMENT d'un defer est évalué à l'ENREGISTREMENT (snapshot fige x=1) ;
//   - une CLOSURE différée lit la variable à l'EXÉCUTION (live voit x=99).
func evalContrast() (snapshot, live int) {
	x := 1
	defer func(v int) { snapshot = v }(x) // v = 1, figé maintenant
	defer func() { live = x }()           // lira x au moment du retour
	x = 99
	return
}

// doubleViaDefer montre qu'un defer peut MODIFIER un retour nommé : la base du
// pattern recover -> erreur du chapitre 17.
func doubleViaDefer() (result int) {
	defer func() { result *= 2 }()
	result = 21
	return result // enregistre result=21, puis le defer le double -> 42
}
