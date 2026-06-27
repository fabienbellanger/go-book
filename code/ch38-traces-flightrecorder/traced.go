package main

import (
	"context"
	"io"
	"runtime/trace"
	"time"
)

// CaptureTrace écrit dans w une trace d'exécution couvrant `work`. Patron
// identique à pprof : Start -> travail -> Stop. La trace se lit avec
// `go tool trace`.
func CaptureTrace(w io.Writer, work func()) error {
	if err := trace.Start(w); err != nil {
		return err
	}
	defer trace.Stop()
	work()
	return nil
}

// processBatch annote son exécution pour `go tool trace` : une TÂCHE englobe le
// lot, deux RÉGIONS délimitent les phases « parse » puis « compute », et un LOG
// marque un évènement ponctuel. Ces annotations apparaissent dans la vue
// « User-defined tasks/regions ».
func processBatch(ctx context.Context, items []int) int {
	ctx, task := trace.NewTask(ctx, "batch") // tâche : un intervalle nommé
	defer task.End()

	var parsed []int
	trace.WithRegion(ctx, "parse", func() { // région : sous-intervalle nommé
		for _, n := range items {
			parsed = append(parsed, n*2)
		}
	})

	sum := 0
	trace.WithRegion(ctx, "compute", func() {
		for _, n := range parsed {
			sum += slowSquareRoot(n)
		}
	})

	trace.Log(ctx, "result", "lot traité") // évènement ponctuel horodaté
	return sum
}

// slowSquareRoot calcule une racine entière par soustractions successives :
// volontairement lent, pour donner de la matière temporelle à la trace.
func slowSquareRoot(n int) int {
	r := 0
	for (r+1)*(r+1) <= n {
		r++
	}
	return r
}

// MonitorLatency illustre le Flight Recorder (🆕 1.25) : on garde en mémoire une
// fenêtre glissante de trace, et on ne l'écrit (`WriteTo`) QUE lorsqu'un évènement
// rare survient — ici, une étape dont la durée dépasse `threshold`. On capture
// ainsi « les dernières secondes avant l'incident » sans tracer en continu.
//
// Renvoie true si une capture a eu lieu.
func MonitorLatency(w io.Writer, threshold time.Duration, steps int, step func(i int)) (bool, error) {
	fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{
		MinAge:   2 * time.Second, // fenêtre : au moins les 2 dernières secondes
		MaxBytes: 1 << 20,         // plafond mémoire de la fenêtre
	})
	if err := fr.Start(); err != nil {
		return false, err
	}
	defer fr.Stop()

	for i := range steps {
		start := time.Now()
		step(i)
		if time.Since(start) >= threshold {
			// Évènement rare : on fige la fenêtre et on s'arrête.
			if _, err := fr.WriteTo(w); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}
