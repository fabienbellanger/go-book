package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMarshalOmits(t *testing.T) {
	// Tags vide (omitempty), CreatedAt zéro (omitzero), Secret exclu (-),
	// internal non exporté : aucun ne doit apparaître. Score est encodé "chaîne".
	e := Event{ID: 7, Name: "x", Score: 5, Secret: "topsecret", internal: "y"}
	js, err := marshalEvent(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{"tags", "created_at", "topsecret", "internal", "Secret"} {
		if strings.Contains(js, absent) {
			t.Errorf("le champ %q ne devrait pas apparaître:\n%s", absent, js)
		}
	}
	if !strings.Contains(js, `"score": "5"`) { // ,string -> chaîne JSON
		t.Errorf("score devrait être une chaîne JSON:\n%s", js)
	}
}

func TestOmitzeroVsPresent(t *testing.T) {
	// Avec une date non nulle, created_at doit cette fois apparaître.
	e := Event{ID: 1, Name: "x", CreatedAt: time.Unix(1700000000, 0).UTC()}
	js, _ := marshalEvent(e)
	if !strings.Contains(js, "created_at") {
		t.Errorf("created_at non zéro devrait apparaître:\n%s", js)
	}
}

func TestStrictDecodeRejectsUnknown(t *testing.T) {
	if _, err := strictDecode(`{"id":1,"name":"x","oops":true}`); err == nil {
		t.Fatal("un champ inconnu devrait être refusé par DisallowUnknownFields")
	}
	if _, err := strictDecode(`{"id":1,"name":"x"}`); err != nil {
		t.Fatalf("entrée valide refusée: %v", err)
	}
}

func TestCustomMarshaler(t *testing.T) {
	var temp Temperature = 21.5
	b, err := json.Marshal(temp)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `"21.5°C"` {
		t.Errorf("Marshal = %s, voulu \"21.5°C\"", got)
	}
	var back Temperature
	if err := json.Unmarshal([]byte(`"19.0°C"`), &back); err != nil {
		t.Fatal(err)
	}
	if back != 19.0 {
		t.Errorf("Unmarshal = %v, voulu 19", back)
	}
}

func TestGobRoundTrip(t *testing.T) {
	in := Event{ID: 3, Name: "deploy", Tags: []string{"a", "b"}, Score: 9}
	out, err := gobRoundTrip(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != in.Name || len(out.Tags) != 2 || out.Score != 9 {
		t.Errorf("round-trip gob altéré: %+v", out)
	}
}

func TestCSVSum(t *testing.T) {
	got, err := csvSum("alice,10\nbob,32\ncarol,8\n")
	if err != nil {
		t.Fatal(err)
	}
	if got != 50 {
		t.Errorf("csvSum = %d, voulu 50", got)
	}
	// Un nombre de colonnes incohérent doit échouer (FieldsPerRecord).
	if _, err := csvSum("alice,10\nbob,32,extra\n"); err == nil {
		t.Error("un CSV à colonnes variables devrait échouer")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Bonjour, le Monde !": "bonjour-le-monde",
		"  Go 1.26  ":         "go-1-26",
		"---":                 "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, voulu %q", in, got, want)
		}
	}
}

func TestParseKV(t *testing.T) {
	got := parseKV("env=prod region=eu zone=a")
	want := map[string]string{"env": "prod", "region": "eu", "zone": "a"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("clé %q = %q, voulu %q", k, got[k], v)
		}
	}
}
