package main

import "testing"

func TestByteVsRune(t *testing.T) {
	cases := []struct {
		s            string
		bytes, runes int
	}{
		{"Go", 2, 2},
		{"héllo", 6, 5}, // é = 2 octets
		{"日本", 6, 2},    // 3 octets par idéogramme
		{"", 0, 0},
	}
	for _, c := range cases {
		b, r := ByteVsRune(c.s)
		if b != c.bytes || r != c.runes {
			t.Errorf("ByteVsRune(%q) = (%d,%d) ; attendu (%d,%d)", c.s, b, r, c.bytes, c.runes)
		}
	}
}

func TestRuneWidths(t *testing.T) {
	// "aé日" : 'a'=1 octet @0, 'é'=2 octets @1, '日'=3 octets @3.
	got := RuneWidths("aé日")
	want := [][2]int{{0, 1}, {1, 2}, {3, 3}}
	if len(got) != len(want) {
		t.Fatalf("len = %d ; attendu %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rune %d : index/largeur = %v ; attendu %v", i, got[i], want[i])
		}
	}
}

func TestJoinCSV(t *testing.T) {
	if got := JoinCSV([]string{"a", "b", "c"}); got != "a,b,c" {
		t.Errorf("JoinCSV = %q ; attendu \"a,b,c\"", got)
	}
	if got := JoinCSV(nil); got != "" {
		t.Errorf("JoinCSV(nil) = %q ; attendu vide", got)
	}
}

func TestToUpperASCII(t *testing.T) {
	if got := ToUpperASCII("héllo"); got != "HéLLO" {
		t.Errorf("ToUpperASCII = %q ; attendu \"HéLLO\"", got)
	}
}

func TestIntern(t *testing.T) {
	a, b := Intern("same.value"), Intern("same.value")
	if a != b {
		t.Error("handles de chaînes égales devraient être ==")
	}
	if a.Value() != "same.value" {
		t.Errorf("Value() = %q ; attendu \"same.value\"", a.Value())
	}
	if c := Intern("other"); a == c {
		t.Error("handles de chaînes différentes ne devraient pas être ==")
	}
	if got := CountDistinct([]string{"x", "x", "y", "z", "z", "z"}); got != 3 {
		t.Errorf("CountDistinct = %d ; attendu 3", got)
	}
}

// Les conversions string<->[]byte qui ÉCHAPPENT copient (1 alloc) ; les conversions
// "consommées sur place" (lookup map, comparaison) sont optimisées SANS copie.
func TestConversionAllocations(t *testing.T) {
	bs := []byte("clé de recherche assez longue pour éviter l'inlining")
	str := string(bs)
	m := map[string]int{str: 1}

	if a := testing.AllocsPerRun(100, func() { sink = string(bs) }); a != 1 {
		t.Errorf("string([]byte) qui échappe = %.0f alloc ; attendu 1", a)
	}
	if a := testing.AllocsPerRun(100, func() {
		if m[string(bs)] == 0 { // lookup : pas de copie
			t.Fatal("clé absente")
		}
	}); a != 0 {
		t.Errorf("m[string(bs)] = %.0f alloc ; attendu 0 (no-copy)", a)
	}
	if a := testing.AllocsPerRun(100, func() {
		if string(bs) != str { // comparaison : pas de copie
			t.Fatal("diff")
		}
	}); a != 0 {
		t.Errorf("string(bs)==str = %.0f alloc ; attendu 0 (no-copy)", a)
	}
}

var sink string
