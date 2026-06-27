package main

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestSlowSquareRoot(t *testing.T) {
	cases := map[int]int{0: 0, 1: 1, 2: 1, 4: 2, 6: 2, 9: 3, 10: 3}
	for n, want := range cases {
		if got := slowSquareRoot(n); got != want {
			t.Errorf("slowSquareRoot(%d) = %d ; attendu %d", n, got, want)
		}
	}
}

func TestProcessBatch(t *testing.T) {
	// items=[0,1,2,3] -> parsed=[0,2,4,6] -> racines 0+1+2+2 = 5.
	if got := processBatch(context.Background(), []int{0, 1, 2, 3}); got != 5 {
		t.Errorf("processBatch = %d ; attendu 5", got)
	}
}

// hasTraceMagic vérifie l'en-tête d'une trace d'exécution ("go 1.26 trace...").
// On teste le préfixe stable, indépendant de la version mineure.
func hasTraceMagic(b []byte) bool {
	return bytes.HasPrefix(b, []byte("go 1."))
}

func TestCaptureTrace(t *testing.T) {
	var buf bytes.Buffer
	err := CaptureTrace(&buf, func() {
		_ = processBatch(context.Background(), []int{1, 2, 3, 4, 5})
	})
	if err != nil {
		t.Fatalf("CaptureTrace: %v", err)
	}
	if !hasTraceMagic(buf.Bytes()) {
		t.Errorf("la sortie n'a pas l'en-tête d'une trace ; %d octets", buf.Len())
	}
}

// Le Flight Recorder ne capture QUE lorsqu'une étape dépasse le seuil.
func TestMonitorLatencyTriggers(t *testing.T) {
	var buf bytes.Buffer
	captured, err := MonitorLatency(&buf, 15*time.Millisecond, 6, func(i int) {
		d := time.Millisecond
		if i == 3 {
			d = 30 * time.Millisecond // dépasse le seuil -> déclenche la capture
		}
		time.Sleep(d)
	})
	if err != nil {
		t.Fatalf("MonitorLatency: %v", err)
	}
	if !captured {
		t.Fatal("la capture aurait dû se déclencher au tour 3")
	}
	if !hasTraceMagic(buf.Bytes()) {
		t.Errorf("la capture du Flight Recorder n'a pas l'en-tête d'une trace ; %d octets", buf.Len())
	}
}

// Sans dépassement, aucune capture.
func TestMonitorLatencyQuiet(t *testing.T) {
	var buf bytes.Buffer
	captured, err := MonitorLatency(&buf, 100*time.Millisecond, 5, func(i int) {
		time.Sleep(time.Millisecond)
	})
	if err != nil {
		t.Fatalf("MonitorLatency: %v", err)
	}
	if captured {
		t.Error("aucune étape ne dépasse le seuil : pas de capture attendue")
	}
}
