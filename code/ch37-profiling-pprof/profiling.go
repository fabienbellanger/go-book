package main

import (
	"io"
	"runtime"
	"runtime/pprof"
	"strings"
)

// collatzSteps renvoie le nombre d'étapes de la suite de Collatz partant de n.
// Pur calcul entier, ZÉRO allocation : le profil CPU est donc dominé par cette
// fonction (pas par le GC), ce qui en fait une cible de profil CPU « propre ».
func collatzSteps(n int) int {
	steps := 0
	for n > 1 {
		if n%2 == 0 {
			n /= 2
		} else {
			n = 3*n + 1
		}
		steps++
	}
	return steps
}

// HotCompute additionne les longueurs de Collatz de 1..max, `repeat` fois.
// Charge CPU-bound représentative pour un profil CPU.
func HotCompute(max, repeat int) int {
	total := 0
	for range repeat {
		for n := 1; n <= max; n++ {
			total += collatzSteps(n)
		}
	}
	return total
}

// wordFrequencies compte la fréquence (en minuscules) de chaque mot du texte,
// répété `repeat` fois. Charge volontairement gourmande en MÉMOIRE : strings.Fields
// alloue un []string et strings.ToLower une string par mot -> cible idéale pour
// un profil TAS.
func wordFrequencies(text string, repeat int) map[string]int {
	counts := make(map[string]int)
	for range repeat {
		for _, w := range strings.Fields(text) {
			counts[strings.ToLower(w)]++ // ToLower : point chaud + allocation
		}
	}
	return counts
}

// CaptureCPUProfile écrit un profil CPU couvrant l'exécution de `work` dans w.
// C'est le patron programmatique : Start -> travail -> Stop (via defer).
func CaptureCPUProfile(w io.Writer, work func()) error {
	if err := pprof.StartCPUProfile(w); err != nil {
		return err
	}
	defer pprof.StopCPUProfile()
	work()
	return nil
}

// CaptureHeapProfile écrit un instantané du tas dans w. runtime.GC() force des
// statistiques à jour avant la capture (sinon on voit un tas « en retard »).
func CaptureHeapProfile(w io.Writer) error {
	runtime.GC()
	return pprof.WriteHeapProfile(w)
}

// CountProfiles renvoie le nombre de profils prédéfinis disponibles (6 par
// défaut : allocs, block, goroutine, heap, mutex, threadcreate).
func CountProfiles() int {
	return len(pprof.Profiles())
}
