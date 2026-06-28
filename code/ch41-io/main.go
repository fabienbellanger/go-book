// Command ch41-io illustre le modèle de flux de Go : io.Reader / io.Writer,
// les buffers (bufio) et les tampons en mémoire (bytes).
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
)

// upperWriter est un io.Writer qui met en majuscules ce qu'on lui écrit avant
// de le transmettre à un Writer sous-jacent. Implémenter une seule méthode
// (Write) suffit à s'insérer dans tout le reste de l'écosystème io.
type upperWriter struct {
	dst io.Writer
}

// Write met en majuscules p puis l'écrit dans dst. On renvoie toujours len(p)
// pour respecter le contrat de io.Writer (n == len(p) si err == nil).
func (w upperWriter) Write(p []byte) (int, error) {
	if _, err := w.dst.Write(bytes.ToUpper(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// copyThrough recopie src dans un upperWriter qui écrit dans une mémoire tampon.
// io.Copy ne charge jamais tout en mémoire : il streame par blocs.
func copyThrough(src io.Reader) (string, error) {
	var sink bytes.Buffer
	if _, err := io.Copy(upperWriter{dst: &sink}, src); err != nil {
		return "", err
	}
	return sink.String(), nil
}

// countLines compte les lignes d'un flux avec un bufio.Scanner. Le Scanner
// découpe le flux selon une fonction de split (ici, par défaut, ligne à ligne).
func countLines(r io.Reader) (int, error) {
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() { // avance d'un token (une ligne) à chaque appel
		n++
	}
	return n, sc.Err() // Scan renvoie false à EOF ET à la première erreur
}

// teeAndCount lit src une seule fois mais en obtient DEUX résultats : la copie
// intégrale (via io.TeeReader, qui recopie au passage) et le nombre d'octets.
func teeAndCount(src io.Reader) (string, int64, error) {
	var mirror bytes.Buffer
	tee := io.TeeReader(src, &mirror) // tout ce qu'on lit dans tee atterrit dans mirror
	n, err := io.Copy(io.Discard, tee)
	if err != nil {
		return "", 0, err
	}
	return mirror.String(), n, nil
}

// pipeProducerConsumer relie un producteur et un consommateur par un io.Pipe :
// pas de fichier, pas de buffer géant, juste un tube synchrone en mémoire.
func pipeProducerConsumer(chunks []string) (string, error) {
	pr, pw := io.Pipe()
	var wg sync.WaitGroup
	wg.Go(func() { // producteur : écrit puis ferme le tube (idiome 1.25)
		defer pw.Close()
		for _, c := range chunks {
			if _, err := io.WriteString(pw, c); err != nil {
				return
			}
		}
	})
	out, err := io.ReadAll(pr) // consommateur : lit jusqu'à la fermeture du tube
	wg.Wait()
	return string(out), err
}

func main() {
	up, _ := copyThrough(strings.NewReader("flux en majuscules"))
	fmt.Println(up)

	lines, _ := countLines(strings.NewReader("a\nb\nc\n"))
	fmt.Println("lignes:", lines)

	mirror, n, _ := teeAndCount(strings.NewReader("miroir"))
	fmt.Printf("copie=%q octets=%d\n", mirror, n)

	joined, _ := pipeProducerConsumer([]string{"pipe", "-", "line"})
	fmt.Println(joined)
}
