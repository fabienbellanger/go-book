package main

// counter sert à montrer le passage PAR VALEUR (Go copie TOUJOURS les arguments).
type counter struct{ n int }

// incVal reçoit une COPIE du counter : l'original de l'appelant n'est pas modifié.
func incVal(c counter) {
	c.n++ // modifie la copie locale, sans effet à l'extérieur
}

// incPtr reçoit un POINTEUR : il peut modifier le counter de l'appelant.
func incPtr(c *counter) {
	c.n++ // déréférencement implicite : équivaut à (*c).n++
}

// scale modifie les éléments d'un slice EN PLACE.
//
// Le slice est lui aussi passé par valeur : c'est son HEADER (ptr/len/cap) qui est
// copié. Mais ce header pointe vers le MÊME tableau sous-jacent, donc écrire
// nums[i] est visible par l'appelant. (Détail des slices au Ch. 6.)
func scale(nums []int, factor int) {
	for i := range nums {
		nums[i] *= factor
	}
}
