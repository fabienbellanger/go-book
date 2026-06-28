package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
	"unicode"
)

// counts agrège les compteurs d'une source.
type counts struct {
	lines, words, runes, bytes int
}

// add accumule les compteurs (utilisé pour la ligne de total).
func (c *counts) add(o counts) {
	c.lines += o.lines
	c.words += o.words
	c.runes += o.runes
	c.bytes += o.bytes
}

// countResult associe un nom de source à ses compteurs (ou à une erreur).
type countResult struct {
	name string
	counts
	err error
}

// runCount implémente « txtkit count ».
func runCount(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("count", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jobs := fs.Int("j", defaultWorkers(), "nombre de fichiers traités en parallèle")
	total := fs.Bool("total", true, "afficher une ligne de total")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage : txtkit count [-j N] [-total] [fichiers...]")
		fmt.Fprintln(stderr, "Compte lignes, mots, runes et octets de chaque entrée.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2 // flag a déjà écrit le message ; -h renvoie aussi ici
	}

	srcs := sourcesFrom(fs.Args(), stdin)

	// Chaque source est comptée dans son propre worker (parallélisme borné).
	results := mapBounded(srcs, *jobs, func(s source) countResult {
		return countSource(s)
	})

	// Sortie alignée en colonnes via tabwriter.
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	var sum counts
	failed := false
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(stderr, "txtkit count : %s : %v\n", r.name, r.err)
			failed = true
			continue
		}
		sum.add(r.counts)
		fmt.Fprintf(tw, "%d\t%d\t%d\t%d\t %s\n", r.lines, r.words, r.runes, r.bytes, r.name)
	}
	if *total && countSucceeded(results) > 1 {
		fmt.Fprintf(tw, "%d\t%d\t%d\t%d\t %s\n", sum.lines, sum.words, sum.runes, sum.bytes, "total")
	}
	tw.Flush()

	if failed {
		return 1
	}
	return 0
}

// countSucceeded renvoie le nombre de sources comptées sans erreur.
func countSucceeded(results []countResult) int {
	n := 0
	for _, r := range results {
		if r.err == nil {
			n++
		}
	}
	return n
}

// countSource ouvre une source et la parcourt rune par rune.
//
// La taille renvoyée par ReadRune est sommée dans `bytes` : pour de l'UTF-8
// valide, c'est le nombre d'octets ; sur un octet invalide, ReadRune renvoie
// utf8.RuneError avec size=1, donc le total d'octets reste exact.
func countSource(s source) countResult {
	rc, err := s.open()
	if err != nil {
		return countResult{name: s.name, err: err}
	}
	defer rc.Close()

	var c counts
	inWord := false
	br := bufio.NewReaderSize(rc, 64*1024)
	for {
		r, size, err := br.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return countResult{name: s.name, err: err}
		}
		c.bytes += size
		c.runes++
		if r == '\n' {
			c.lines++
		}
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			c.words++
		}
	}
	return countResult{name: s.name, counts: c}
}
