package main

import (
	"slices"
	"testing"
)

// LIFO : enregistrés 0,1,2 -> exécutés 2,1,0.
func TestLIFOOrder(t *testing.T) {
	if got, want := lifoOrder(), []int{2, 1, 0}; !slices.Equal(got, want) {
		t.Errorf("lifoOrder = %v ; attendu %v", got, want)
	}
}

// Argument évalué à l'enregistrement vs closure lue à l'exécution.
func TestEvalContrast(t *testing.T) {
	snap, live := evalContrast()
	if snap != 1 {
		t.Errorf("snapshot = %d ; attendu 1 (figé à l'enregistrement)", snap)
	}
	if live != 99 {
		t.Errorf("live = %d ; attendu 99 (lu au retour)", live)
	}
}

// Un defer modifie le retour nommé.
func TestDoubleViaDefer(t *testing.T) {
	if got := doubleViaDefer(); got != 42 {
		t.Errorf("doubleViaDefer = %d ; attendu 42", got)
	}
}

// La closure par itération ferme chaque ressource à son tour...
func TestProcessScoped(t *testing.T) {
	got := processScoped([]string{"a", "b"})
	want := []string{"open:a", "use:a", "close:a", "open:b", "use:b", "close:b"}
	if !slices.Equal(got, want) {
		t.Errorf("processScoped =\n  %v\nattendu\n  %v", got, want)
	}
}

// ...alors que defer en boucle repousse tous les Close à la fin (LIFO).
func TestProcessDeferInLoop(t *testing.T) {
	got := processDeferInLoop([]string{"a", "b"})
	want := []string{"open:a", "use:a", "open:b", "use:b", "close:b", "close:a"}
	if !slices.Equal(got, want) {
		t.Errorf("processDeferInLoop =\n  %v\nattendu\n  %v", got, want)
	}
}

// Le trace journalise enter avant le corps, exit après.
func TestTrace(t *testing.T) {
	var log []string
	func() {
		defer trace("f", &log)()
		log = append(log, "body")
	}()
	want := []string{"enter:f", "body", "exit:f"}
	if !slices.Equal(log, want) {
		t.Errorf("trace = %v ; attendu %v", log, want)
	}
}
