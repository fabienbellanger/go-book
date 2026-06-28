package main

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestCopyThrough(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"simple", "abc", "ABC"},
		{"déjà majuscule", "ABC", "ABC"},
		{"mixte", "Go 1.26", "GO 1.26"},
		{"vide", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := copyThrough(strings.NewReader(tc.in))
			if err != nil {
				t.Fatalf("copyThrough: %v", err)
			}
			if got != tc.want {
				t.Errorf("copyThrough(%q) = %q ; attendu %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	cases := []struct {
		name, in string
		want     int
	}{
		{"trois lignes", "a\nb\nc\n", 3},
		{"sans newline final", "a\nb", 2},
		{"vide", "", 0},
		{"une ligne", "seule\n", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := countLines(strings.NewReader(tc.in))
			if err != nil {
				t.Fatalf("countLines: %v", err)
			}
			if got != tc.want {
				t.Errorf("countLines(%q) = %d ; attendu %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestTeeAndCount(t *testing.T) {
	mirror, n, err := teeAndCount(strings.NewReader("miroir"))
	if err != nil {
		t.Fatalf("teeAndCount: %v", err)
	}
	if mirror != "miroir" {
		t.Errorf("copie = %q ; attendu %q", mirror, "miroir")
	}
	if n != int64(len("miroir")) {
		t.Errorf("octets = %d ; attendu %d", n, len("miroir"))
	}
}

func TestPipeProducerConsumer(t *testing.T) {
	got, err := pipeProducerConsumer([]string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("pipeProducerConsumer: %v", err)
	}
	if got != "abc" {
		t.Errorf("résultat = %q ; attendu %q", got, "abc")
	}
}

// TestScannerWords montre le découpage par mots (bufio.ScanWords).
func TestScannerWords(t *testing.T) {
	sc := bufio.NewScanner(strings.NewReader("  le  flux   Go "))
	sc.Split(bufio.ScanWords)
	var words []string
	for sc.Scan() {
		words = append(words, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	want := []string{"le", "flux", "Go"}
	if strings.Join(words, ",") != strings.Join(want, ",") {
		t.Errorf("mots = %v ; attendu %v", words, want)
	}
}

// TestScannerTooLong illustre le piège ⚠️ : une ligne plus longue que le buffer
// maximal du Scanner provoque bufio.ErrTooLong.
func TestScannerTooLong(t *testing.T) {
	long := strings.Repeat("x", 1024)
	sc := bufio.NewScanner(strings.NewReader(long))
	sc.Buffer(make([]byte, 16), 64) // buffer plafonné à 64 octets
	for sc.Scan() {
	}
	if err := sc.Err(); err != bufio.ErrTooLong {
		t.Errorf("err = %v ; attendu bufio.ErrTooLong", err)
	}
}

// TestBufferedWriterFlush illustre le piège ⚠️ : sans Flush, les octets restent
// dans le tampon et n'atteignent jamais le Writer sous-jacent.
func TestBufferedWriterFlush(t *testing.T) {
	var sink bytes.Buffer
	bw := bufio.NewWriter(&sink)
	io.WriteString(bw, "tamponné")
	if sink.Len() != 0 {
		t.Fatalf("avant Flush, sink devrait être vide, a %d octets", sink.Len())
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if sink.String() != "tamponné" {
		t.Errorf("après Flush = %q ; attendu %q", sink.String(), "tamponné")
	}
}

// TestLinesIterator montre l'alternative itérateur (Go 1.24) au Scanner.
func TestLinesIterator(t *testing.T) {
	var got []string
	for line := range strings.Lines("a\nb\nc\n") { // les lignes incluent le \n
		got = append(got, strings.TrimRight(line, "\n"))
	}
	if strings.Join(got, ",") != "a,b,c" {
		t.Errorf("lignes = %v ; attendu [a b c]", got)
	}
}
