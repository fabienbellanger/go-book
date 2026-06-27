package main

import (
	"slices"
	"testing"
)

// parallelMap préserve l'ordre malgré l'exécution concurrente : chaque goroutine
// écrit à son index. Lancer `go test -race` confirme l'absence de course.
func TestParallelMapPreservesOrder(t *testing.T) {
	got := parallelMap([]int{1, 2, 3, 4, 5}, square)
	want := []int{1, 4, 9, 16, 25}
	if !slices.Equal(got, want) {
		t.Errorf("parallelMap = %v ; attendu %v", got, want)
	}
}

// parallelMap sur une entrée vide ne lance aucune goroutine et renvoie une slice
// vide (cas limite).
func TestParallelMapEmpty(t *testing.T) {
	got := parallelMap([]int{}, square)
	if len(got) != 0 {
		t.Errorf("parallelMap([]) = %v ; attendu []", got)
	}
}

// Arrêt propre : après la fermeture de stop, la goroutine se termine (done est
// fermé) et n'incrémente plus le compteur. Pas de fuite.
func TestGracefulStop(t *testing.T) {
	stop := make(chan struct{})
	count, done := tickUntilStop(stop)

	for count.Load() == 0 { // attendre que la goroutine ait démarré
	}
	close(stop) // demande d'arrêt
	<-done      // la fermeture de done « synchronise avant » la lecture qui suit

	final := count.Load()
	if final == 0 {
		t.Fatal("la goroutine n'a jamais tourné")
	}
	if again := count.Load(); again != final {
		t.Errorf("la goroutine tourne encore après done : %d puis %d", final, again)
	}
}
