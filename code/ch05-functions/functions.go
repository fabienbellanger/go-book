package main

import "errors"

// divmod renvoie le quotient ET le reste : illustration des RETOURS MULTIPLES.
func divmod(a, b int) (int, int) {
	return a / b, a % b
}

// safeDivide illustre la convention idiomatique : l'erreur est le DERNIER retour.
// L'appelant teste `err != nil` avant d'utiliser le résultat.
func safeDivide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division par zéro")
	}
	return a / b, nil
}

// sum est VARIADIQUE : elle accepte un nombre quelconque d'entiers.
// À l'intérieur, nums est un []int.
func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// minMax utilise des RETOURS NOMMÉS : lo et hi sont déclarés dans la signature
// (et initialisés à leur zero value). On les renvoie explicitement pour la clarté.
func minMax(nums ...int) (lo, hi int) {
	if len(nums) == 0 {
		return 0, 0
	}
	lo, hi = nums[0], nums[0]
	for _, n := range nums[1:] {
		lo = min(lo, n) // builtins 1.21
		hi = max(hi, n)
	}
	return lo, hi
}

// apply prend une FONCTION EN PARAMÈTRE (les fonctions sont des valeurs de
// première classe) et l'applique à chaque élément.
func apply(nums []int, f func(int) int) []int {
	out := make([]int, len(nums))
	for i, n := range nums {
		out[i] = f(n)
	}
	return out
}

// factorial illustre la RÉCURSIVITÉ.
func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n-1)
}
