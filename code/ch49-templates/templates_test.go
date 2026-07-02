package main

import (
	"strings"
	"testing"
)

func TestRenderInvoice(t *testing.T) {
	inv := Invoice{
		Customer: "café du coin",
		Lines: []Line{
			{Label: "Expresso", Cents: 150},
			{Label: "Croissant", Cents: 120},
		},
	}
	got, err := render(inv)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// Rendu exact : upper sur le client, frMoney sur chaque montant et le total,
	// et la branche {{else}} du statut (facture non payée).
	want := "Facture — CAFÉ DU COIN\n" +
		"- Expresso: 1,50 €\n" +
		"- Croissant: 1,20 €\n" +
		"Total: 2,70 €\n" +
		"Statut: à régler\n"
	if got != want {
		t.Errorf("render =\n%q\nvoulu\n%q", got, want)
	}
}

func TestRenderInvoicePaid(t *testing.T) {
	inv := Invoice{Customer: "x", Paid: true}
	got, err := render(inv)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, "Statut: payée") {
		t.Errorf("branche {{if .Paid}} non prise : %q", got)
	}
}

func TestFrMoney(t *testing.T) {
	cases := []struct {
		cents int64
		want  string
	}{
		{150, "1,50 €"},
		{120, "1,20 €"},
		{5, "0,05 €"},
		{1205, "12,05 €"},
	}
	for _, c := range cases {
		if got := frMoney(c.cents); got != c.want {
			t.Errorf("frMoney(%d) = %q, voulu %q", c.cents, got, c.want)
		}
	}
}

func TestRenderMenu(t *testing.T) {
	// range + composition {{template "item" .}} : une puce par élément.
	got, err := renderMenu([]string{"Entrée", "Plat", "Dessert"})
	if err != nil {
		t.Fatalf("renderMenu: %v", err)
	}
	want := "Menu:\n- Entrée\n- Plat\n- Dessert\n"
	if got != want {
		t.Errorf("renderMenu =\n%q\nvoulu\n%q", got, want)
	}
}

func TestMissingKey(t *testing.T) {
	// missingkey=error : une clé absente doit provoquer une erreur d'exécution.
	if _, err := renderStrict(`Bonjour {{.name}}`, map[string]any{}); err == nil {
		t.Error("clé absente : erreur attendue, obtenu nil")
	}
	// La même clé présente rend normalement.
	got, err := renderStrict(`Bonjour {{.name}}`, map[string]any{"name": "Go"})
	if err != nil {
		t.Fatalf("renderStrict (clé présente) : %v", err)
	}
	if got != "Bonjour Go" {
		t.Errorf("renderStrict = %q, voulu %q", got, "Bonjour Go")
	}
}

func TestRenderPageEscaping(t *testing.T) {
	// La même entrée hostile est échappée DIFFÉREMMENT selon le contexte.
	got, err := renderPage("<script>alert(1)</script>")
	if err != nil {
		t.Fatalf("renderPage: %v", err)
	}
	// Contexte « corps HTML » : les chevrons deviennent des entités, la balise
	// <script> est neutralisée.
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("corps HTML non échappé :\n%s", got)
	}
	// Contexte « URL d'attribut href » : la valeur est encodée pour une URL, donc
	// aucune balise <script> brute ne subsiste dans l'attribut.
	if strings.Contains(got, `href="/u/<script>`) {
		t.Errorf("URL d'attribut non échappée :\n%s", got)
	}
}
