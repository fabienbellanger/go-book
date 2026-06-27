package main

import (
	"expvar"
	"runtime"
)

// expvar publie des variables sur /debug/vars (au format JSON) dès que le package
// est importé. C'est l'exposition « zéro dépendance » du runtime. En production,
// on branche souvent Prometheus à la place ou en plus.

// requestsServed : un compteur applicatif exposé automatiquement.
var requestsServed = expvar.NewInt("requests_served")

func init() {
	// goroutines_live : une JAUGE évaluée à CHAQUE lecture de /debug/vars.
	expvar.Publish("goroutines_live", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
}

// RecordRequest incrémente le compteur applicatif (à appeler par requête servie).
func RecordRequest() { requestsServed.Add(1) }

// RequestsServed lit la valeur courante du compteur.
func RequestsServed() int64 { return requestsServed.Value() }
