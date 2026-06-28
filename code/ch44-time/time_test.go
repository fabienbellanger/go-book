package main

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

// TestMeasure vérifie que measure renvoie une durée non négative et que le
// travail a bien été exécuté (effet de bord observable).
func TestMeasure(t *testing.T) {
	ran := false
	d := measure(func() { ran = true })
	if !ran {
		t.Fatal("work n'a pas été exécuté")
	}
	if d < 0 {
		t.Errorf("durée négative : %v", d)
	}
}

// TestSlowDoubleSucceeds : la deadline (1s) laisse le temps au délai (10ms),
// donc slowDouble renvoie le double. Grâce à synctest (1.25), l'horloge est
// VIRTUELLE : le test ne dort pas réellement 10ms.
func TestSlowDoubleSucceeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		got, err := slowDouble(ctx, 21, 10*time.Millisecond)
		if err != nil {
			t.Fatalf("erreur inattendue : %v", err)
		}
		if got != 42 {
			t.Errorf("got = %d, voulu 42", got)
		}
	})
}

// TestSlowDoubleTimeout : la deadline (20ms) expire AVANT le délai (200ms).
// On teste un timeout sans aucun Sleep réel : l'horloge virtuelle avance
// instantanément dès que toutes les goroutines sont durablement bloquées.
func TestSlowDoubleTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err := slowDouble(ctx, 21, 200*time.Millisecond)
		if err != context.DeadlineExceeded {
			t.Errorf("err = %v, voulu context.DeadlineExceeded", err)
		}
	})
}

// TestCountTicks vérifie le comptage des tops sous horloge virtuelle :
// une fenêtre de 55ms à 10ms d'intervalle produit 5 tops (à 10,20,30,40,50ms ;
// le 6e à 60ms tombe après la deadline).
func TestCountTicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if got := countTicks(10*time.Millisecond, 55*time.Millisecond); got != 5 {
			t.Errorf("tops = %d, voulu 5", got)
		}
	})
}

// TestMonotonicStripped illustre que Round(0) retire la composante monotone :
// after n'a plus de lecture monotone, mais représente le même instant (Equal).
func TestMonotonicStripped(t *testing.T) {
	now := time.Now()
	stripped := now.Round(0)
	if !now.Equal(stripped) {
		t.Error("Round(0) doit conserver l'instant (Equal), seulement retirer le monotone")
	}
}
