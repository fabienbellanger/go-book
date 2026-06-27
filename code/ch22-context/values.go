package main

import "context"

// ctxKey est un type NON exporté : il garantit qu'aucune autre partie du code ne
// peut entrer en collision avec notre clé dans le contexte. On n'utilise JAMAIS
// une string nue comme clé (collisions et fuites d'abstraction).
type ctxKey int

const requestIDKey ctxKey = iota

// WithRequestID attache un identifiant de requête au contexte. Les valeurs de
// contexte servent aux données qui TRAVERSENT les frontières d'API (trace, auth),
// pas aux paramètres normaux d'une fonction.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID lit l'identifiant ; ok vaut false s'il est absent.
func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}
