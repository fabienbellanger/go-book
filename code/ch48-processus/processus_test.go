package main

import (
	"context"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestParseFlags couvre le cas nominal et l'erreur sur un flag inconnu. Grâce à
// ContinueOnError, l'erreur est renvoyée au lieu de terminer le process de test.
func TestParseFlags(t *testing.T) {
	cfg, err := parseFlags("greet", []string{"-name", "Ada", "-count", "3", "-verbose"})
	if err != nil {
		t.Fatalf("parseFlags nominal: %v", err)
	}
	if cfg.name != "Ada" || cfg.count != 3 || !cfg.verbose {
		t.Errorf("config = %+v, valeurs attendues name=Ada count=3 verbose=true", cfg)
	}

	if _, err := parseFlags("greet", []string{"-inconnu"}); err == nil {
		t.Error("un flag inconnu aurait dû produire une erreur")
	}

	// count négatif : rejeté par la validation métier.
	if _, err := parseFlags("greet", []string{"-count", "-1"}); err == nil {
		t.Error("count négatif aurait dû produire une erreur")
	}
}

// TestCapture lance « go version » : le binaire go est forcément présent puisque
// le test lui-même tourne sous « go test ». On vérifie que la sortie mentionne go.
func TestCapture(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := capture(ctx, "go", "version")
	if err != nil {
		t.Fatalf("capture(go version): %v", err)
	}
	if !strings.Contains(out, "go") {
		t.Errorf("sortie inattendue : %q", out)
	}
}

// TestCaptureExitCode vérifie qu'un code de sortie != 0 est bien remonté comme
// erreur : « go help » avec un sujet inexistant renvoie un statut non nul.
func TestCaptureExitCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := capture(ctx, "go", "sujet-de-commande-inexistant"); err == nil {
		t.Error("une commande en échec aurait dû renvoyer une erreur")
	}
}

// TestNotify prouve la réception d'un signal : on écoute SIGUSR1, on se l'envoie à
// soi-même, et on vérifie qu'il arrive avant un délai de sécurité. SIGUSR1 est
// idéal en test : sa disposition par défaut tuerait le process, mais signal.Notify
// l'intercepte, et il n'est jamais émis spontanément.
func TestNotify(t *testing.T) {
	ch, stop := notify(syscall.SIGUSR1)
	defer stop()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatalf("envoi de SIGUSR1: %v", err)
	}

	select {
	case got := <-ch:
		if got != syscall.SIGUSR1 {
			t.Errorf("signal reçu = %v, voulu SIGUSR1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("signal SIGUSR1 non reçu dans le délai imparti")
	}
}

// TestServeStopsOnCancel montre l'arrêt propre : serve rend la main dès que le
// context est annulé (ce que ferait un vrai signal via signal.NotifyContext).
func TestServeStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan string, 1)
	go func() { done <- serve(ctx) }()

	cancel() // simule la réception du signal
	select {
	case msg := <-done:
		if !strings.Contains(msg, "arrêt propre") {
			t.Errorf("message d'arrêt inattendu : %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve n'a pas rendu la main après annulation du context")
	}
}
