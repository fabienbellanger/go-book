package main

import (
	"slices"
	"testing"
	"time"
)

// range sur un canal : collecte jusqu'à la fermeture.
func TestGenAndRange(t *testing.T) {
	var got []int
	for v := range gen(1, 2, 3) {
		got = append(got, v)
	}
	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("gen+range = %v ; attendu %v", got, want)
	}
}

// Fan-in : toutes les valeurs des deux sources arrivent, dans un ordre
// indéterminé. On compare donc des ensembles triés.
func TestFanInMergesAll(t *testing.T) {
	var got []int
	for v := range fanIn(gen(1, 2), gen(3, 4)) {
		got = append(got, v)
	}
	slices.Sort(got)
	if want := []int{1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("fanIn = %v ; attendu %v (à l'ordre près)", got, want)
	}
}

// select + default : envoi non bloquant. Plein -> false ; place libre -> true.
func TestTrySend(t *testing.T) {
	ch := make(chan int, 1)
	if !trySend(ch, 1) {
		t.Error("trySend sur tampon vide devrait réussir")
	}
	if trySend(ch, 2) {
		t.Error("trySend sur tampon plein devrait échouer")
	}
}

// select + time.After : le délai expire sur un canal jamais alimenté.
func TestRecvTimeoutFires(t *testing.T) {
	if _, ok := recvWithTimeout(make(chan int), 10*time.Millisecond); ok {
		t.Error("recvWithTimeout aurait dû expirer")
	}
}

// select + time.After : une valeur déjà prête gagne contre un long délai.
func TestRecvTimeoutValueWins(t *testing.T) {
	v, ok := recvWithTimeout(gen(42), time.Second)
	if !ok || v != 42 {
		t.Errorf("recvWithTimeout = (%d, %v) ; attendu (42, true)", v, ok)
	}
}
