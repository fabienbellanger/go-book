package main

import (
	"strings"
	"testing"
)

// TestGoroutineLeakDetected vérifie que le profil `goroutineleak` (🆕 stable en
// Go 1.27) repère une goroutine définitivement bloquée.
//
// ⚠️ Ce test fuit VOLONTAIREMENT une goroutine : c'est son objet. La goroutine
// reste bloquée jusqu'à la fin du processus de test — c'est sans conséquence
// ici (quelques kio de pile), mais ne reproduisez pas ce motif en production.
func TestGoroutineLeakDetected(t *testing.T) {
	avant, err := CountLeakedGoroutines()
	if err != nil {
		t.Fatalf("CountLeakedGoroutines: %v", err)
	}

	leakOnUnreadChannel()

	après, err := CountLeakedGoroutines()
	if err != nil {
		t.Fatalf("CountLeakedGoroutines: %v", err)
	}
	if après <= avant {
		t.Errorf("fuites : %d avant, %d après ; le profil aurait dû en détecter une de plus",
			avant, après)
	}
}

// TestGoroutineLeakReportPointeLaLigne vérifie que le profil nomme bien la
// fonction bloquée — c'est ce qui le rend exploitable, par opposition à un
// simple compteur de goroutines.
func TestGoroutineLeakReportPointeLaLigne(t *testing.T) {
	leakOnUnreadChannel()

	var buf strings.Builder
	if err := GoroutineLeakReport(&buf); err != nil {
		t.Fatalf("GoroutineLeakReport: %v", err)
	}
	rapport := buf.String()

	if !strings.Contains(rapport, "goroutineleak profile:") {
		t.Fatalf("en-tête de profil absent :\n%s", rapport)
	}
	if !strings.Contains(rapport, "leakOnUnreadChannel") {
		t.Errorf("le profil devrait nommer leakOnUnreadChannel ; obtenu :\n%s", rapport)
	}
}

// TestProfileTotalParsing couvre l'analyse de l'en-tête, y compris les cas
// dégradés (profil vide, format inattendu).
func TestProfileTotalParsing(t *testing.T) {
	cases := map[string]int{
		"goroutineleak profile: total 3\n1 @ 0x1\n": 3,
		"goroutineleak profile: total 0\n":          0,
		"goroutineleak profile: total 12":           12,
		"":                                          0,
		"format inattendu":                          0,
	}
	for in, want := range cases {
		if got := parseProfileTotal(in); got != want {
			t.Errorf("parseProfileTotal(%q) = %d ; attendu %d", in, got, want)
		}
	}
}
