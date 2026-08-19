package main

import (
	"io"
	"runtime"
	"runtime/pprof"
	"strings"
)

// Détection de fuites de goroutines avec le profil `goroutineleak`
// (🆕 stable en Go 1.27 ; expérimental en 1.26 derrière un GOEXPERIMENT).
//
// COMMENT ÇA MARCHE. Le profil ne mesure pas une durée de blocage : il
// s'appuie sur le GC. Si une goroutine G est bloquée sur une primitive P
// (canal, sync.Mutex, sync.Cond…) et que P n'est JOIGNABLE depuis aucune
// goroutine exécutable — ni depuis une goroutine que celles-ci pourraient
// réveiller — alors plus personne ne peut toucher P : G ne se réveillera
// jamais. C'est une certitude, d'où l'absence de faux positifs.
//
// ⚠️ LA RÉCIPROQUE EST FAUSSE. Une goroutine bloquée sur un canal encore
// référencé par une variable GLOBALE, ou par la pile d'une goroutine
// exécutable, reste « joignable » aux yeux du GC : elle n'apparaîtra pas ici,
// même si en pratique plus personne n'écrira jamais dedans.

// leakOnUnreadChannel fabrique une fuite DÉTECTABLE : la goroutine attend sur
// un canal que la fonction abandonne en sortant. Plus aucune référence
// n'existe côté émetteur, donc le GC peut conclure.
func leakOnUnreadChannel() {
	ch := make(chan int) // personne n'écrira jamais
	go func() {
		<-ch // bloqué pour toujours
	}()
}

// GoroutineLeakReport déclenche un GC puis écrit le profil `goroutineleak`
// dans w, au format lisible (debug=1).
//
// Le runtime.GC() n'est pas cosmétique : c'est le ramasse-miettes qui établit
// l'inaccessibilité des primitives, donc le profil n'a de contenu qu'après un
// cycle.
func GoroutineLeakReport(w io.Writer) error {
	runtime.GC()
	p := pprof.Lookup("goroutineleak")
	if p == nil {
		// Avant Go 1.27, sans GOEXPERIMENT=goroutineleakprofile, Lookup
		// renvoyait nil. Depuis 1.27, ce cas ne se produit plus.
		return nil
	}
	return p.WriteTo(w, 1)
}

// CountLeakedGoroutines renvoie le nombre de goroutines que le profil
// considère comme définitivement bloquées.
//
// L'en-tête du profil a la forme « goroutineleak profile: total N ».
func CountLeakedGoroutines() (int, error) {
	var buf strings.Builder
	if err := GoroutineLeakReport(&buf); err != nil {
		return 0, err
	}
	return parseProfileTotal(buf.String()), nil
}

// parseProfileTotal extrait le N de « <nom> profile: total N ».
func parseProfileTotal(profil string) int {
	const marqueur = "profile: total "
	i := strings.Index(profil, marqueur)
	if i < 0 {
		return 0
	}
	reste := profil[i+len(marqueur):]
	if j := strings.IndexAny(reste, " \n"); j >= 0 {
		reste = reste[:j]
	}
	n := 0
	for _, c := range reste {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}
