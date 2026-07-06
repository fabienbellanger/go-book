// Package model définit les types partagés du générateur de site : la structure
// du livre (parties, pages, titres) et les entrées de l'index de recherche.
package model

import "html/template"

// Book représente le livre complet, prêt à être rendu en site.
type Book struct {
	Title   string
	Version string  // version affichée dans le pied de page (ex. « v1.0.0 »)
	Year    int     // année de copyright affichée dans le pied de page
	Parts   []*Part // ordre canonique du SOMMAIRE
	Pages   []*Page // toutes les pages rendues, à plat (recherche & navigation)
}

// Part est une section du sommaire (« Partie I — Fondamentaux… »).
type Part struct {
	Title string
	Desc  string  // description optionnelle (blockquote sous le titre de partie)
	Pages []*Page // pages internes (rendues) de la partie
	Links []*Link // entrées non rendues (projets pointant vers le dépôt)
}

// Link est une entrée de sommaire qui n'est pas une page rendue : un projet
// pointant vers un dossier du dépôt, listé dans la navigation sans rendu.
type Link struct {
	Title string
	Href  string // chemin tel quel (relatif à la racine du livre)
}

// Page est un fichier Markdown rendu en HTML.
type Page struct {
	SrcPath   string        // ex: chapitres/41-io-flux.md
	OutPath   string        // ex: chapitres/41-io-flux.html
	Title     string        // titre H1
	PartTitle string        // partie d'appartenance (pour la sidebar)
	HTML      template.HTML // corps rendu
	Headings  []Heading     // ToC locale (H2/H3 + ancres)
	PlainText string        // texte brut, pour l'index de recherche
	Prev      *Page         // page précédente (ordre du SOMMAIRE)
	Next      *Page         // page suivante
}

// Heading est un titre de niveau H2/H3 avec son ancre, pour la ToC locale.
type Heading struct {
	Level int    // 2 ou 3
	Text  string // texte affiché
	ID    string // ancre slugifiée (attribut id du titre HTML)
}

// SearchDoc est une entrée de l'index de recherche sérialisé en JSON.
type SearchDoc struct {
	URL     string `json:"url"`               // ex: chapitres/41-io-flux.html#scanner
	Title   string `json:"title"`             // titre de la page ou de la section
	Part    string `json:"part"`              // partie d'appartenance
	Section string `json:"section,omitempty"` // titre de section H2 (si applicable)
	Content string `json:"content"`           // texte brut normalisé (sans accents)
}
