// Package domain contient les types métier et leurs règles, SANS aucune
// dépendance technique (ni base de données, ni HTTP, ni framework). C'est le
// cœur de l'application : tout le reste dépend de lui, jamais l'inverse.
package domain

import "errors"

// ErrNotFound signale qu'une note demandée n'existe pas. Placer les erreurs
// sentinelles dans le paquet feuille domain évite les cycles d'import : le cœur
// (service) comme la périphérie (store) peuvent l'importer sans se référencer.
var ErrNotFound = errors.New("note introuvable")

// Note est l'entité métier. Champs exportés, aucune méthode technique : c'est
// une donnée pure que toutes les couches se partagent.
type Note struct {
	ID    string
	Title string
	Body  string
}
