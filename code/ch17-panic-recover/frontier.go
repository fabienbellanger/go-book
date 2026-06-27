package main

import "fmt"

// Modèle simplifié d'un serveur HTTP (cf. projet 2 pour le vrai net/http).
type Request struct{ path string }

type Response struct {
	status int
	body   string
}

type Handler func(Request) Response

// recoverMiddleware est la "frontière de recover" : il enveloppe un handler de
// sorte qu'une panique pendant le traitement d'UNE requête renvoie 500 au lieu
// de faire planter tout le serveur. Chaque requête est isolée.
func recoverMiddleware(next Handler) Handler {
	return func(req Request) (resp Response) {
		defer func() {
			if r := recover(); r != nil {
				// En vrai : on loggerait r + la pile (debug.Stack()).
				resp = Response{status: 500, body: fmt.Sprintf("internal error: %v", r)}
			}
		}()
		return next(req)
	}
}

// app est un handler qui panique sur "/boom" et répond normalement ailleurs.
func app(req Request) Response {
	if req.path == "/boom" {
		panic("handler en échec")
	}
	return Response{status: 200, body: "ok:" + req.path}
}
