package cli

import (
	"bufio"
	"cmp"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
)

// runFreq implémente « txtkit freq » : les mots les plus fréquents.
func runFreq(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("freq", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jobs := fs.Int("j", defaultWorkers(), "nombre de fichiers traités en parallèle")
	topN := fs.Int("n", 10, "nombre de mots à afficher (0 = tous)")
	minLen := fs.Int("min", 1, "longueur minimale d'un mot")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage : txtkit freq [-j N] [-n N] [-min L] [fichiers...]")
		fmt.Fprintln(stderr, "Affiche les mots les plus fréquents, toutes entrées confondues.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	srcs := sourcesFrom(fs.Args(), stdin)

	// Chaque source produit sa propre table (fan-out borné)...
	type partial struct {
		freq map[string]int
		err  error
		name string
	}
	parts := mapBounded(srcs, *jobs, func(s source) partial {
		freq, err := freqSource(s, *minLen)
		return partial{freq: freq, err: err, name: s.name}
	})

	// ...puis on fusionne (fan-in) en une seule table.
	merged := make(map[string]int)
	failed := false
	for _, p := range parts {
		if p.err != nil {
			fmt.Fprintf(stderr, "txtkit freq : %s : %v\n", p.name, p.err)
			failed = true
			continue
		}
		for word, n := range p.freq {
			merged[word] += n
		}
	}

	for _, e := range topWords(merged, *topN) {
		fmt.Fprintf(stdout, "%7d  %s\n", e.count, e.word)
	}

	if failed {
		return 1
	}
	return 0
}

// entry est un couple mot/occurrences, pour le tri.
type entry struct {
	word  string
	count int
}

// topWords trie la table par fréquence décroissante (puis ordre alphabétique à
// égalité, pour un résultat déterministe) et renvoie les n premiers.
func topWords(freq map[string]int, n int) []entry {
	entries := make([]entry, 0, len(freq))
	for w, c := range freq {
		entries = append(entries, entry{word: w, count: c})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		if d := cmp.Compare(b.count, a.count); d != 0 {
			return d // fréquence décroissante
		}
		return cmp.Compare(a.word, b.word) // puis alphabétique croissant
	})
	if n > 0 && n < len(entries) {
		entries = entries[:n]
	}
	return entries
}

// freqSource compte les mots d'une source dans une table dédiée.
func freqSource(s source, minLen int) (map[string]int, error) {
	rc, err := s.open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	freq := make(map[string]int)
	sc := bufio.NewScanner(rc)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20) // tolère des « mots » longs (1 Mo)
	sc.Split(bufio.ScanWords)
	for sc.Scan() {
		if w := normalize(sc.Text()); len(w) >= minLen {
			freq[w]++
		}
	}
	return freq, sc.Err()
}

// normalize replie un token en minuscules en ne gardant que lettres et chiffres.
// "Go," et "go" comptent ainsi pour le même mot ; "..." devient vide.
func normalize(token string) string {
	var b strings.Builder
	b.Grow(len(token))
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
