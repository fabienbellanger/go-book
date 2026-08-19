package main

// Ce fichier illustre encoding/json/v2 et encoding/json/jsontext, stables
// depuis 🆕 Go 1.27. Le reste du chapitre utilise encoding/json (v1), qui
// reste pleinement supporté — et qui, depuis 1.27, tourne SUR le moteur v2.
//
// L'import est aliasé pour cohabiter avec "encoding/json" dans le même
// programme : `jsonv2` ici, `json` là.

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"io"
	"strings"
)

// Réponse est volontairement dépourvue d'`omitempty` : c'est le cas où la
// différence v1/v2 sur les slices/maps nil se voit. En v1, une API qui renvoie
// `{"items": null}` oblige chaque client à traiter le null ; d'où le réflexe
// défensif `if items == nil { items = []T{} }` avant de sérialiser. En v2, ce
// réflexe devient inutile.
type Réponse struct {
	Items  []string          `json:"items"`
	Labels map[string]string `json:"labels"`
}

// marshalRéponseV1 sérialise avec encoding/json (v1) : nil -> null.
func marshalRéponseV1(r Réponse) (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	return string(b), err
}

// marshalRéponseV2 sérialise avec encoding/json/v2 : nil -> [] et {}.
func marshalRéponseV2(r Réponse) (string, error) {
	b, err := jsonv2.Marshal(r, jsontext.WithIndent("  "))
	return string(b), err
}

// marshalEventV2 sérialise avec v2 et une option d'indentation.
//
// Deux différences visibles avec v1 sur le même Event :
//   - un slice/map nil s'encode en [] / {} au lieu de null ;
//   - l'indentation est une OPTION (jsontext.WithIndent) au lieu d'une
//     fonction distincte (MarshalIndent).
func marshalEventV2(e Event) (string, error) {
	b, err := jsonv2.Marshal(e, jsontext.WithIndent("  "))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// strictDecodeV2 est l'équivalent v2 de strictDecode : refuser une clé
// inconnue devient une OPTION déclarative, au lieu d'un réglage sur le Decoder.
//
// v2 est de toute façon plus strict par défaut : noms d'objet dupliqués et
// UTF-8 invalide sont des erreurs, là où v1 les acceptait silencieusement.
func strictDecodeV2(src string) (Event, error) {
	var e Event
	err := jsonv2.UnmarshalRead(strings.NewReader(src), &e,
		jsonv2.RejectUnknownMembers(true))
	if err != nil {
		return Event{}, err
	}
	return e, nil
}

// hasDuplicateName indique si le document JSON contient un nom d'objet dupliqué.
// C'est un cas que v1 accepte (la dernière valeur gagne) et que v2 REFUSE.
func hasDuplicateName(src string) bool {
	var e Event
	err := jsonv2.Unmarshal([]byte(src), &e)
	return errors.Is(err, jsontext.ErrDuplicateName)
}

// countStrings parcourt un document JSON au niveau SYNTAXIQUE, sans le mapper
// sur un type Go : jsontext livre une suite de tokens.
//
// C'est le bon outil pour filtrer, valider ou réindenter un flux dont on ne
// connaît pas (ou dont on ne veut pas décrire) la structure.
func countStrings(src string) (int, error) {
	dec := jsontext.NewDecoder(strings.NewReader(src))
	n := 0
	for {
		tok, err := dec.ReadToken()
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return 0, err
		}
		if tok.Kind() == jsontext.KindString {
			n++
		}
	}
}
