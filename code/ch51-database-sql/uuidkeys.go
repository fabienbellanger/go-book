package main

import (
	"cmp"
	"slices"
	"uuid"
)

// Ce fichier illustre le package `uuid` de la bibliothèque standard
// (🆕 Go 1.27, RFC 9562) dans son usage le plus courant : la clé primaire.
//
// Avant 1.27, il fallait une dépendance tierce (github.com/google/uuid).

// Document modélise une ligne dont la clé primaire est un UUID plutôt qu'un
// entier auto-incrémenté.
//
// UUID est un [16]byte : un tableau COMPARABLE (utilisable avec ==, ou comme
// clé de map) qui implémente encoding.TextMarshaler/TextUnmarshaler — il se
// sérialise donc tout seul en JSON, et se stocke soit en TEXT (36 octets),
// soit en BLOB/BYTEA de 16 octets si l'espace compte.
type Document struct {
	ID    uuid.UUID `json:"id"`
	Titre string    `json:"titre"`
}

// NewDocument crée un document avec un identifiant de version 7.
//
// ⚠️ V7 plutôt que V4, et ce n'est pas un détail de style. Un V4 est aléatoire
// sur toute sa longueur : inséré comme clé primaire d'un index B-tree, chaque
// nouvelle ligne atterrit à un endroit imprévisible de l'arbre, ce qui
// fragmente l'index et fait chuter le taux de succès du cache de pages. Un V7
// commence par un horodatage : les insertions successives sont quasi
// séquentielles, comme un auto-increment, tout en restant globalement uniques
// et non devinables à l'échelle.
func NewDocument(titre string) Document {
	return Document{ID: uuid.NewV7(), Titre: titre}
}

// ParseDocumentID décode un identifiant reçu sous forme de texte (paramètre
// d'URL, corps JSON), en refusant ce qui n'est pas un UUID valide.
func ParseDocumentID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// SortByID trie des documents par identifiant. Avec des V7, ce tri est
// équivalent à un tri par DATE DE CRÉATION — la propriété qui justifie V7.
func SortByID(docs []Document) {
	slices.SortFunc(docs, func(a, b Document) int {
		return cmp.Compare(a.ID.String(), b.ID.String())
	})
}
