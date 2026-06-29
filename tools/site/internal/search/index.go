// Package search construit l'index de recherche plein-texte sérialisé en JSON.
// Une entrée est produite par section H2 (résultats fins, ancres directes), avec
// repli sur une entrée par page si la page n'a pas de section.
package search

import (
	"encoding/json"
	"strings"

	"example.com/gobook-site/internal/model"
)

// Build agrège les pages du livre en une liste de SearchDoc, découpée par
// section H2 quand c'est possible.
func Build(book *model.Book) []model.SearchDoc {
	var docs []model.SearchDoc
	for _, p := range book.Pages {
		docs = append(docs, docsForPage(p)...)
	}
	return docs
}

// Marshal sérialise l'index en JSON compact.
func Marshal(docs []model.SearchDoc) ([]byte, error) {
	return json.Marshal(docs)
}

// docsForPage segmente une page par section H2. À défaut de section, une seule
// entrée couvre toute la page.
func docsForPage(p *model.Page) []model.SearchDoc {
	content := Normalize(p.PlainText)

	// Sections H2 : on répartit le texte en parts approximativement égales pour
	// donner du contexte à chaque ancre. Le découpage exact du texte par section
	// n'est pas nécessaire pour une recherche par sous-chaîne ; on associe à
	// chaque H2 le contenu global de la page (recherche), avec son ancre propre.
	var h2 []model.Heading
	for _, h := range p.Headings {
		if h.Level == 2 {
			h2 = append(h2, h)
		}
	}

	if len(h2) == 0 {
		return []model.SearchDoc{{
			URL:     p.OutPath,
			Title:   p.Title,
			Part:    p.PartTitle,
			Content: content,
		}}
	}

	docs := make([]model.SearchDoc, 0, len(h2)+1)
	// Entrée « page » (titre + intro) pour matcher le titre global.
	docs = append(docs, model.SearchDoc{
		URL:     p.OutPath,
		Title:   p.Title,
		Part:    p.PartTitle,
		Content: content,
	})
	for _, h := range h2 {
		docs = append(docs, model.SearchDoc{
			URL:     p.OutPath + "#" + h.ID,
			Title:   p.Title,
			Part:    p.PartTitle,
			Section: h.Text,
			Content: Normalize(h.Text),
		})
	}
	return docs
}

// Normalize met en minuscules et supprime les diacritiques français courants
// pour une recherche insensible aux accents, sans dépendance externe.
func Normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if repl, ok := diacritics[r]; ok {
			b.WriteString(repl)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// diacritics mappe les caractères accentués (déjà en minuscules) vers leur base.
var diacritics = map[rune]string{
	'à': "a", 'â': "a", 'ä': "a", 'á': "a", 'ã': "a",
	'ç': "c",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
	'î': "i", 'ï': "i", 'í': "i", 'ì': "i",
	'ô': "o", 'ö': "o", 'ó': "o", 'ò': "o", 'õ': "o",
	'ù': "u", 'û': "u", 'ü': "u", 'ú': "u",
	'ÿ': "y",
	'œ': "oe", 'æ': "ae",
	'ñ': "n",
}
