// code/ch48-processus/main.go
//
// Command ch48-processus illustre les trois façons de piloter un programme
// depuis l'extérieur, toutes avec la bibliothèque standard : les arguments de
// ligne de commande (flag), le lancement de sous-processus (os/exec) et la
// réaction aux signaux du système (os/signal). Le tout reste testable
// hermétiquement — voir processus_test.go.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// config rassemble les options analysées sur la ligne de commande. On la remplit
// via parseFlags, une fonction PURE (elle ne touche ni os.Args ni os.Exit), donc
// entièrement testable.
type config struct {
	name    string
	count   int
	verbose bool
	timeout time.Duration
}

// parseFlags analyse args (SANS le nom du programme) avec un FlagSet dédié en
// mode ContinueOnError : une erreur est RENVOYÉE au lieu de terminer le process
// (flag.ExitOnError, le défaut du FlagSet global, appelle os.Exit(2)). C'est ce
// qui rend la fonction testable.
func parseFlags(name string, args []string) (config, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)

	var cfg config
	fs.StringVar(&cfg.name, "name", "Go", "nom à saluer")
	fs.IntVar(&cfg.count, "count", 1, "nombre de répétitions")
	fs.BoolVar(&cfg.verbose, "verbose", false, "sortie détaillée")
	fs.DurationVar(&cfg.timeout, "timeout", 5*time.Second, "délai maximal des sous-commandes")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if cfg.count < 0 {
		return config{}, fmt.Errorf("count doit être >= 0, reçu %d", cfg.count)
	}
	return cfg, nil
}

// capture lance une commande externe et renvoie sa sortie standard. Points clés :
//   - exec.CommandContext lie la commande à un context : si ctx expire ou est
//     annulé, le process enfant est tué. Toujours un timeout pour ne pas bloquer.
//   - .Output() capture stdout ; en cas d'échec, on distingue le « la commande a
//     renvoyé un code != 0 » (*exec.ExitError, avec stderr disponible) du reste
//     (binaire introuvable, ctx expiré...).
//   - les arguments sont passés SÉPARÉMENT : aucun shell n'interprète la chaîne,
//     donc aucune injection possible (🔁 Ch. 47).
func capture(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("%s: code de sortie %d: %s",
				name, ee.ExitCode(), bytes.TrimSpace(ee.Stderr))
		}
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}

// notify installe l'écoute des signaux demandés sur un canal BUFFERISÉ (au moins
// 1 : sinon un signal envoyé alors que personne ne lit encore serait perdu, car
// le runtime ne bloque jamais pour livrer un signal). Elle renvoie le canal et
// une fonction d'arrêt à différer (signal.Stop détache le canal).
func notify(sigs ...os.Signal) (<-chan os.Signal, func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	return ch, func() { signal.Stop(ch) }
}

// serve simule un service qui tourne jusqu'à ce que son context soit annulé, puis
// effectue un arrêt propre. En production, ctx vient de signal.NotifyContext, si
// bien qu'un Ctrl-C (SIGINT) ou un SIGTERM (docker stop, kubernetes) déclenche
// l'arrêt gracieux au lieu de tuer le process net (🔁 Ch. 45, Projet 2).
func serve(ctx context.Context) string {
	<-ctx.Done() // bloque jusqu'au signal / à l'annulation
	return "arrêt propre : connexions drainées, ressources libérées"
}

func main() {
	cfg, err := parseFlags("greet", os.Args[1:])
	if err != nil {
		// ContinueOnError a déjà affiché le message et l'usage sur stderr.
		os.Exit(2)
	}
	for i := 0; i < cfg.count; i++ {
		fmt.Printf("Bonjour, %s !\n", cfg.name)
	}

	// os/exec : on interroge la version de Go via un sous-processus borné.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()
	if out, err := capture(ctx, "go", "version"); err == nil {
		fmt.Printf("toolchain : %s", out)
	} else if cfg.verbose {
		fmt.Fprintf(os.Stderr, "capture: %v\n", err)
	}

	// os/signal : arrêt propre sur SIGINT (Ctrl-C) ou SIGTERM. NotifyContext est
	// l'idiome moderne — il annule le context à la réception d'un de ces signaux.
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Println("service démarré (Ctrl-C pour un arrêt propre)")
	fmt.Println(serve(sigCtx))
}
