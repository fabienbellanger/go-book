package search

import (
	"encoding/json"
	"strings"
	"testing"

	"example.com/gobook-site/internal/model"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Élève À Côté": "eleve a cote",
		"FONCTION":     "fonction",
		"cœur":         "coeur",
		"déjà-vu":      "deja-vu",
		"io.Copy":      "io.copy",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestBuild(t *testing.T) {
	book := &model.Book{
		Pages: []*model.Page{
			{
				OutPath:   "chapitres/x.html",
				Title:     "Chapitre X",
				PartTitle: "Partie I",
				PlainText: "Contenu avec accents é à ç",
				Headings: []model.Heading{
					{Level: 2, Text: "Section A", ID: "section-a"},
					{Level: 3, Text: "Ignorée", ID: "ignoree"}, // H3 non indexé séparément
					{Level: 2, Text: "Section B", ID: "section-b"},
				},
			},
			{
				OutPath:   "annexes/a.html",
				Title:     "Annexe sans section",
				PartTitle: "Annexes",
				PlainText: "Juste du texte",
			},
		},
	}

	docs := Build(book)

	// Page 1 : 1 entrée page + 2 entrées H2 = 3 ; page 2 : 1 entrée. Total 4.
	if len(docs) != 4 {
		t.Fatalf("docs = %d, attendu 4", len(docs))
	}

	// Toutes les URLs sont non vides ; les entrées de section ont une ancre.
	var sectionDocs int
	for _, d := range docs {
		if d.URL == "" {
			t.Errorf("URL vide dans %+v", d)
		}
		if d.Section != "" {
			sectionDocs++
			if !strings.Contains(d.URL, "#") {
				t.Errorf("entrée de section sans ancre : %q", d.URL)
			}
		}
	}
	if sectionDocs != 2 {
		t.Errorf("entrées de section = %d, attendu 2 (les H2)", sectionDocs)
	}

	// Contenu normalisé sans accents.
	if strings.ContainsAny(docs[0].Content, "éàç") {
		t.Errorf("contenu non normalisé : %q", docs[0].Content)
	}

	// Sérialisation JSON valide.
	data, err := Marshal(docs)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back []model.SearchDoc
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(back) != len(docs) {
		t.Errorf("round-trip JSON : %d != %d", len(back), len(docs))
	}
}
