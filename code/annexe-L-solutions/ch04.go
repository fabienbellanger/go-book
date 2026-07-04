package main

// ch04Classify traduit un score /100 en mention, réécrit avec un switch
// d'EXPRESSION sur la dizaine (exercice 1). Plus lisible qu'une cascade de if :
// chaque case regroupe les dizaines équivalentes, sans fallthrough.
func ch04Classify(score int) string {
	switch score / 10 {
	case 10, 9:
		return "excellent"
	case 8, 7:
		return "bien"
	case 6, 5:
		return "passable"
	default:
		return "insuffisant"
	}
}

// ch04FirstPair cherche le premier couple (i, j) dont la somme vaut target dans
// deux tranches. L'étiquette `search` permet à `break` de quitter les DEUX
// boucles d'un coup (exercice 2 : sans étiquette, seule la boucle interne
// s'arrête et l'on continue à chercher à tort).
func ch04FirstPair(xs, ys []int, target int) (int, int, bool) {
	var fi, fj int
	found := false
search:
	for _, x := range xs {
		for _, y := range ys {
			if x+y == target {
				fi, fj, found = x, y, true
				break search
			}
		}
	}
	return fi, fj, found
}
