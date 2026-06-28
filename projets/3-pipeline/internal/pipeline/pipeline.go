// Package pipeline fournit un pipeline concurrent générique et réutilisable :
// une source (fan-out vers N workers bornés), une étape de traitement, puis un
// canal de sorties (fan-in). Il gère l'annulation (context), la pression
// arrière (canaux bornés), la limitation de débit, la propagation de la
// première erreur (errgroup) et des métriques.
//
// Le cœur est Process : il prend une séquence d'entrées (iter.Seq) et renvoie
// un canal de sorties plus une fonction Wait() qui restitue la première erreur.
package pipeline

import (
	"context"
	"iter"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// Stage transforme une entrée en sortie. Une erreur non nulle est fatale : via
// errgroup, elle annule le contexte partagé et arrête tout le pipeline.
type Stage[I, O any] func(ctx context.Context, in I) (O, error)

// Limiter borne le débit de traitement (un jeton par élément). nil = pas de
// limite. Voir RateLimiter.
type Limiter interface {
	Wait(ctx context.Context) error
}

// Config règle le pipeline.
type Config struct {
	Workers int     // nombre de workers concurrents (fan-out) ; < 1 => 1
	Buffer  int     // taille du canal de sorties (pression arrière) ; < 0 => 0
	Limiter Limiter // optionnel : limitation de débit
}

func (c *Config) normalize() {
	if c.Workers < 1 {
		c.Workers = 1
	}
	if c.Buffer < 0 {
		c.Buffer = 0
	}
}

// Process applique fn à chaque élément de items, avec cfg.Workers workers
// concurrents, et renvoie le canal des sorties.
//
// Schéma : un *feeder* lit items et alimente un canal interne `in` (fan-out) ;
// N *workers* lisent `in`, appliquent fn et écrivent dans `out` (fan-in). Tout
// vit dans un errgroup : la première erreur annule le contexte, ce qui débloque
// feeder et workers (aucune goroutine ne fuit).
//
// Utilisation type :
//
//	out, m, wait := pipeline.Process(ctx, items, fn, cfg)
//	for v := range out {     // draine d'abord toutes les sorties…
//		use(v)
//	}
//	err := wait()            // …puis récupère la première erreur éventuelle
//	log(m.Snapshot())
func Process[I, O any](ctx context.Context, items iter.Seq[I], fn Stage[I, O], cfg Config) (<-chan O, *Metrics, func() error) {
	cfg.normalize()
	m := &Metrics{}
	out := make(chan O, cfg.Buffer)
	in := make(chan I) // non bufferisé : la pression arrière remonte jusqu'au feeder

	g, gctx := errgroup.WithContext(ctx)

	// Feeder : seule goroutine à écrire dans `in`, donc seule à le fermer. Le
	// select sur gctx.Done() évite de rester bloqué si les workers s'arrêtent.
	g.Go(func() error {
		defer close(in)
		for it := range items {
			select {
			case in <- it:
			case <-gctx.Done():
				return gctx.Err()
			}
		}
		return nil
	})

	// Workers : un WaitGroup distinct nous dit quand fermer `out` (errgroup
	// n'expose pas cet instant). Chaque worker décrémente wg en sortant.
	var wg sync.WaitGroup
	wg.Add(cfg.Workers)
	for range cfg.Workers {
		g.Go(func() error {
			defer wg.Done()
			return worker(gctx, in, out, fn, cfg, m)
		})
	}

	// Fermeture de `out` une fois TOUS les workers terminés : le consommateur
	// voit alors son `range out` se terminer proprement.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out, m, g.Wait
}

// worker boucle sur `in` jusqu'à épuisement (canal fermé) ou annulation.
func worker[I, O any](ctx context.Context, in <-chan I, out chan<- O, fn Stage[I, O], cfg Config, m *Metrics) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-in:
			if !ok {
				return nil // source épuisée : fin normale
			}
			if cfg.Limiter != nil {
				if err := cfg.Limiter.Wait(ctx); err != nil {
					return err
				}
			}

			cur := m.inFlight.Add(1)
			m.observeMax(cur)
			res, err := fn(ctx, item)
			m.inFlight.Add(-1)
			if err != nil {
				m.failed.Add(1)
				return err // errgroup capture et annule les autres
			}
			m.processed.Add(1)

			select {
			case out <- res:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Metrics agrège des compteurs concurrents (atomiques, donc sûrs sans verrou).
type Metrics struct {
	processed   atomic.Int64
	failed      atomic.Int64
	inFlight    atomic.Int64
	maxInFlight atomic.Int64
}

// Snapshot est une vue figée des métriques, sans atomiques (copiable, affichable).
type Snapshot struct {
	Processed   int64 `json:"processed"`
	Failed      int64 `json:"failed"`
	InFlight    int64 `json:"in_flight"`
	MaxInFlight int64 `json:"max_in_flight"`
}

// Snapshot lit l'état courant des compteurs.
func (m *Metrics) Snapshot() Snapshot {
	return Snapshot{
		Processed:   m.processed.Load(),
		Failed:      m.failed.Load(),
		InFlight:    m.inFlight.Load(),
		MaxInFlight: m.maxInFlight.Load(),
	}
}

// observeMax met à jour maxInFlight si cur le dépasse (boucle de compare-and-swap).
func (m *Metrics) observeMax(cur int64) {
	for {
		old := m.maxInFlight.Load()
		if cur <= old || m.maxInFlight.CompareAndSwap(old, cur) {
			return
		}
	}
}
