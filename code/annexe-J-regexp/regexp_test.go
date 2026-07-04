package main

import (
	"reflect"
	"testing"
)

func TestFirstWord(t *testing.T) {
	if got := firstWord("  Go 1.26 !"); got != "Go" {
		t.Errorf("firstWord = %q, veut %q", got, "Go")
	}
	if got := firstWord("  !!! "); got != "" {
		t.Errorf("firstWord sans mot = %q, veut \"\"", got)
	}
}

func TestAllWords(t *testing.T) {
	got := allWords("un deux trois")
	want := []string{"un", "deux", "trois"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("allWords = %q, veut %q", got, want)
	}
	if got := allWords("!!!"); got != nil {
		t.Errorf("allWords sans mot = %q, veut nil", got)
	}
}

func TestParseDate(t *testing.T) {
	y, mo, d, ok := parseDate("release 2026-07-04 ok")
	if !ok || y != "2026" || mo != "07" || d != "04" {
		t.Errorf("parseDate = %q/%q/%q ok=%v", y, mo, d, ok)
	}
	if _, _, _, ok := parseDate("pas de date"); ok {
		t.Error("parseDate devrait échouer sans date")
	}
}

func TestIsLogLevel(t *testing.T) {
	cases := map[string]bool{
		"INFO":    true, // insensible à la casse
		"debug":   true,
		"warn":    true,
		"verbose": false, // inconnu
		"info ":   false, // ancré : l'espace final casse la correspondance
	}
	for in, want := range cases {
		if got := isLogLevel(in); got != want {
			t.Errorf("isLogLevel(%q) = %v, veut %v", in, got, want)
		}
	}
}

func TestRedactDigits(t *testing.T) {
	if got := redactDigits("carte 4242 42"); got != "carte **** **" {
		t.Errorf("redactDigits = %q", got)
	}
}

func TestSwapDate(t *testing.T) {
	if got := swapDate("2026-07-04"); got != "04/07/2026" {
		t.Errorf("swapDate = %q, veut %q", got, "04/07/2026")
	}
}

func TestSplitFields(t *testing.T) {
	got := splitFields("a  b   c")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitFields = %q, veut %q", got, want)
	}
}
