package sommaire

import (
	"strings"
	"testing"
)

const mini = `# Sommaire

> Intro à ignorer (sous le H1).

## Partie I — Fondamentaux

- Ch. 0 — [Premier](chapitres/00-premier.md)
- Ch. 1 — [Deuxième](chapitres/01-deuxieme.md)

## Partie VIII — Projets

> Note descriptive de la partie.

- Projet 1 — [CLI](projets/1-cli/)

## Annexes

- A — [Glossaire](annexes/A-glossaire.md)
`

func TestParse(t *testing.T) {
	book, err := Parse(strings.NewReader(mini), "Test")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if book.Title != "Test" {
		t.Errorf("titre = %q, attendu Test", book.Title)
	}
	if len(book.Parts) != 3 {
		t.Fatalf("parties = %d, attendu 3", len(book.Parts))
	}
	if got := book.Parts[0].Title; got != "Partie I — Fondamentaux" {
		t.Errorf("partie 0 = %q", got)
	}

	// 3 pages rendues (2 chapitres + 1 annexe) ; le projet est un Link.
	if len(book.Pages) != 3 {
		t.Fatalf("pages = %d, attendu 3", len(book.Pages))
	}
	if book.Pages[0].OutPath != "chapitres/00-premier.html" {
		t.Errorf("OutPath = %q", book.Pages[0].OutPath)
	}
	projets := book.Parts[1]
	if len(projets.Links) != 1 || projets.Links[0].Href != "projets/1-cli/" {
		t.Errorf("lien projet mal parsé : %+v", projets.Links)
	}
	if projets.Desc != "Note descriptive de la partie." {
		t.Errorf("desc partie = %q", projets.Desc)
	}
}

func TestParseOrderAndPrevNext(t *testing.T) {
	book, _ := Parse(strings.NewReader(mini), "T")
	if book.Pages[0].Prev != nil {
		t.Error("première page ne doit pas avoir de Prev")
	}
	if book.Pages[0].Next != book.Pages[1] {
		t.Error("Next de la page 0 doit être la page 1")
	}
	last := book.Pages[len(book.Pages)-1]
	if last.Next != nil {
		t.Error("dernière page ne doit pas avoir de Next")
	}
}

func TestMdToHTML(t *testing.T) {
	cases := map[string]string{
		"a/b.md":         "a/b.html",
		"a/b.md#sec":     "a/b.html#sec",
		"chapitres/x.md": "chapitres/x.html",
	}
	for in, want := range cases {
		if got := mdToHTML(in); got != want {
			t.Errorf("mdToHTML(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestIsRenderable(t *testing.T) {
	yes := []string{"a.md", "a.md#x"}
	no := []string{"projets/1/", "https://x.md", "PLAN.md/"}
	for _, h := range yes {
		if !isRenderable(h) {
			t.Errorf("%q devrait être rendable", h)
		}
	}
	for _, h := range no {
		if isRenderable(h) {
			t.Errorf("%q ne devrait pas être rendable", h)
		}
	}
}
