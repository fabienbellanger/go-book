package main

import (
	"fmt"
	"strings"
)

// --- Décorateur : envelopper une fonction pour lui ajouter un comportement. ---

// logged enveloppe fn pour tracer chaque appel, sans modifier fn elle-même.
func logged(name string, fn func(int) int) func(int) int {
	return func(x int) int {
		out := fn(x)
		fmt.Printf("[trace] %s(%d) = %d\n", name, x, out)
		return out
	}
}

// --- Mémoïsation : une closure capture un cache et évite de recalculer. ---

// memoize renvoie une version de fn qui mémorise ses résultats. calls compte les
// appels RÉELS à fn (pour prouver, dans les tests, que le cache fonctionne).
func memoize(fn func(int) int) (memo func(int) int, calls *int) {
	cache := map[int]int{}
	n := 0
	memo = func(x int) int {
		if v, ok := cache[x]; ok {
			return v // cache hit : fn n'est pas rappelée
		}
		n++
		v := fn(x)
		cache[x] = v
		return v
	}
	return memo, &n
}

// --- Middleware : des closures qui enveloppent un handler et se chaînent. ---

// Handler traite une requête (modèle simplifié de net/http).
type Handler func(req string) string

// Middleware transforme un Handler en un autre Handler.
type Middleware func(Handler) Handler

// chain applique les middlewares de gauche à droite : chain(h, a, b) exécute
// a puis b puis h.
func chain(h Handler, mws ...Middleware) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// upper met la réponse en majuscules.
func upper(next Handler) Handler {
	return func(req string) string { return strings.ToUpper(next(req)) }
}

// tagged est un middleware PARAMÉTRÉ : il capture tag dans la closure.
func tagged(tag string) Middleware {
	return func(next Handler) Handler {
		return func(req string) string { return tag + ":" + next(req) }
	}
}
