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
		"  Go 1.27  ":         "go-1-27",
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

// --- encoding/json/v2 et jsontext (🆕 Go 1.27) ---

// TestMarshalV2NilSlice montre LA différence de défaut la plus visible entre
// v1 et v2 : un slice nil s'encode en null (v1) ou en [] (v2).
func TestMarshalV2NilSlice(t *testing.T) {
	r := Réponse{} // Items et Labels sont nil, et sans omitempty

	v1, err := marshalRéponseV1(r)
	if err != nil {
		t.Fatalf("marshalRéponseV1: %v", err)
	}
	v2, err := marshalRéponseV2(r)
	if err != nil {
		t.Fatalf("marshalRéponseV2: %v", err)
	}

	if !strings.Contains(v1, `"items": null`) || !strings.Contains(v1, `"labels": null`) {
		t.Errorf("v1 devrait encoder nil en null ; obtenu :\n%s", v1)
	}
	if !strings.Contains(v2, `"items": []`) || !strings.Contains(v2, `"labels": {}`) {
		t.Errorf("v2 devrait encoder nil en [] / {} ; obtenu :\n%s", v2)
	}

	// marshalEventV2 reste couvert : omitempty l'emporte, Tags est omis.
	if got, err := marshalEventV2(Event{ID: 1, Name: "deploy"}); err != nil {
		t.Fatalf("marshalEventV2: %v", err)
	} else if strings.Contains(got, "tags") {
		t.Errorf("omitempty devrait omettre tags en v2 aussi ; obtenu :\n%s", got)
	}
}

// TestRejectUnknownMembers vérifie l'option v2 équivalente à
// Decoder.DisallowUnknownFields() de v1.
func TestRejectUnknownMembers(t *testing.T) {
	const avecIntrus = `{"id":1,"name":"deploy","inconnu":true}`

	if _, err := strictDecodeV2(avecIntrus); err == nil {
		t.Error("strictDecodeV2 aurait dû refuser la clé inconnue")
	}
	// Sans l'option, la clé inconnue est simplement ignorée.
	const propre = `{"id":1,"name":"deploy"}`
	e, err := strictDecodeV2(propre)
	if err != nil {
		t.Fatalf("strictDecodeV2(propre) : %v", err)
	}
	if e.Name != "deploy" {
		t.Errorf("Name = %q ; attendu \"deploy\"", e.Name)
	}
}

// TestDuplicateNameRejected : v2 refuse un nom d'objet dupliqué, v1 l'accepte
// (la dernière valeur l'emporte). C'est un durcissement, pas un bug.
func TestDuplicateNameRejected(t *testing.T) {
	const dup = `{"name":"a","name":"b"}`

	if !hasDuplicateName(dup) {
		t.Error("v2 aurait dû signaler jsontext.ErrDuplicateName")
	}

	// v1 accepte et garde la dernière valeur.
	var e Event
	if err := json.Unmarshal([]byte(dup), &e); err != nil {
		t.Fatalf("v1 devrait accepter le doublon : %v", err)
	}
	if e.Name != "b" {
		t.Errorf("v1 : Name = %q ; attendu \"b\" (dernière valeur)", e.Name)
	}
}

// TestCountStrings exerce la couche syntaxique jsontext : compter les chaînes
// d'un document sans jamais le typer.
func TestCountStrings(t *testing.T) {
	// 3 chaînes : "deploy", "prod", "eu" — les clés ne sont pas des tokens
	// de type chaîne au sens de ReadToken ? Si, elles le sont : "name",
	// "tags", "region" comptent aussi. Total : 3 valeurs + 3 clés = 6.
	const src = `{"name":"deploy","tags":["prod"],"region":"eu"}`

	n, err := countStrings(src)
	if err != nil {
		t.Fatalf("countStrings: %v", err)
	}
	if n != 6 {
		t.Errorf("countStrings = %d ; attendu 6 (3 clés + 3 valeurs)", n)
	}
}
