# 29 — Observabilité du runtime & monitoring

> **Objectif** — Savoir **voir** ce que fait le runtime en production : lire l'état des goroutines, du
> tas et du GC avec **`runtime`**, **`runtime/metrics`** (l'API moderne) et **`runtime/debug`** ;
> connaître les sondes **`GODEBUG`** ; et **exposer** ces signaux (`expvar`, `net/http/pprof`,
> Prometheus).
>
> **Prérequis** — [Ch. 27](27-garbage-collector.md), [Ch. 28](28-ordonnanceur-gmp.md), [Ch. 19](19-goroutines.md)

---

## Introduction

Comprendre le runtime, c'est bien ; le **mesurer en vol**, c'est indispensable. Combien de goroutines
tournent ? Le tas grossit-il ? Le GC s'emballe-t-il ? Go expose tout cela **sans agent externe**, via
des packages standard. Ce chapitre **clôt la Partie IV** en reliant les notions vues (GC,
ordonnanceur, mémoire) à leurs **compteurs observables**. Code dans
[`code/ch29-observabilite-runtime/`](../code/ch29-observabilite-runtime/).

---

## `runtime` : les sondes directes

Les plus simples sont dans le package `runtime` :

- **`runtime.NumGoroutine()`** — nombre de goroutines **utilisateur**.
- **`runtime.ReadMemStats(&m)`** — un `MemStats` **complet** (tas, pile, GC, allocs). ⚠️ Il fige
  brièvement le monde pour figer les compteurs : à **ne pas** appeler en boucle serrée.

```go
// code/ch29-observabilite-runtime/metrics.go
func LegacyHeapAlloc() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // court arrêt pour figer les stats
	return m.HeapAlloc
}
```

## `runtime/metrics` : l'API moderne

Depuis Go 1.16, **`runtime/metrics`** est l'API **recommandée** : stable, extensible (~112 métriques en
1.26), histogrammes inclus, et **plus légère** que `ReadMemStats`. On déclare les noms voulus, on
appelle `Read` :

```go
// code/ch29-observabilite-runtime/metrics.go
func readUint(name string) uint64 {
	s := []metrics.Sample{{Name: name}}
	metrics.Read(s)
	if s[0].Value.Kind() == metrics.KindUint64 {
		return s[0].Value.Uint64()
	}
	return 0
}

func ReadSnapshot() Snapshot {
	return Snapshot{
		Goroutines:        runtime.NumGoroutine(),
		GoroutinesAll:     readUint("/sched/goroutines:goroutines"),
		GoroutinesCreated: readUint("/sched/goroutines-created:goroutines"),
		HeapAllocBytes:    readUint("/memory/classes/heap/objects:bytes"),
		HeapObjects:       readUint("/gc/heap/objects:objects"),
		NumGC:             readUint("/gc/cycles/total:gc-cycles"),
		/* ... */
	}
}
```

```
$ go run ./ch29-observabilite-runtime
Go go1.26.4, GOMAXPROCS=8
au repos : goroutines utilisateur=1, toutes=7 (écart = système)
50 lancées : goroutines=51, toutes=57, créées (cumul)=57
```

> 💡 `runtime.NumGoroutine()` compte les goroutines **utilisateur** ; `/sched/goroutines` compte
> **toutes** (y compris les ~6 goroutines **système** du runtime). L'écart est normal.

## Les métriques `/sched` (1.26)

Go 1.26 enrichit la famille `/sched/` — précieuse pour diagnostiquer un service concurrent :

| Métrique                                | Sens                                               |
| --------------------------------------- | -------------------------------------------------- |
| `/sched/goroutines:goroutines`          | total de goroutines (système comprises)            |
| `/sched/goroutines-created:goroutines`  | **cumul** de goroutines créées (détecte une fuite) |
| `/sched/goroutines/running:goroutines`  | en cours d'exécution                               |
| `/sched/goroutines/runnable:goroutines` | prêtes, en attente d'un P                          |
| `/sched/goroutines/waiting:goroutines`  | bloquées (I/O, canal, verrou)                      |
| `/sched/threads/total:threads`          | threads OS (M)                                     |
| `/sched/gomaxprocs:threads`             | nombre de P                                        |

Beaucoup de `waiting` ? Vos goroutines bloquent (lock, canal non drainé). `runnable` qui s'accumule ?
Vous manquez de P. Le **cumul** `goroutines-created` qui ne cesse de monter alors que le total stagne
révèle un **flux** de goroutines éphémères. 🔁 [Ch. 23](23-patterns-concurrence.md), `goroutineleak`.

## `runtime/debug` : agir et identifier

- **`debug.ReadBuildInfo()`** — version Go, chemin du module, **infos VCS** (commit, date), réglages de
  build. Idéal pour un endpoint `/version`.
- **`debug.SetGCPercent`**, **`debug.SetMemoryLimit`** — régler le GC ([Ch. 27](27-garbage-collector.md)).
- **`debug.FreeOSMemory()`** — rendre la mémoire libre à l'OS **immédiatement** (sinon le runtime la
  garde un temps). À utiliser avec parcimonie.
- **`debug.Stack()`** / **`debug.PrintStack()`** — dump de la pile courante (diagnostic, `recover`).

```go
if bi, ok := debug.ReadBuildInfo(); ok {
	s.GoVersion = bi.GoVersion // ex. "go1.26.4"
}
```

## Les sondes `GODEBUG` (récapitulatif)

Sans toucher au code, `GODEBUG` ouvre des fenêtres sur le runtime :

| `GODEBUG=...`       | Ce que ça montre                                  | Détail                            |
| ------------------- | ------------------------------------------------- | --------------------------------- |
| `gctrace=1`         | une ligne **par cycle de GC** (pauses, tas, goal) | [Ch. 27](27-garbage-collector.md) |
| `schedtrace=N`      | **photo de l'ordonnanceur** toutes les N ms       | [Ch. 28](28-ordonnanceur-gmp.md)  |
| `inittrace=1`       | coût d'**init** de chaque package                 | [Ch. 24](24-runtime-bootstrap.md) |
| `checkfinalizers=1` | diagnostique finalizers/cleanups (1.25)           | [Ch. 27](27-garbage-collector.md) |

## Exposer en production

Trois niveaux d'exposition, du plus simple au plus complet :

```go
// code/ch29-observabilite-runtime/expvar.go
// expvar : variables JSON sur /debug/vars, ZÉRO dépendance.
var requestsServed = expvar.NewInt("requests_served")

func init() {
	expvar.Publish("goroutines_live", expvar.Func(func() any {
		return runtime.NumGoroutine() // jauge évaluée à chaque lecture
	}))
}
```

- **`expvar`** — publie des variables sur **`/debug/vars`** (JSON). Le runtime y ajoute
  automatiquement `memstats` et `cmdline`. Suffisant pour un dashboard maison.
- **`net/http/pprof`** — un simple `import _ "net/http/pprof"` greffe **`/debug/pprof/`** (profils CPU,
  heap, goroutine, **`goroutineleak`**). Détaillé au [Ch. 37](37-profiling-pprof.md).
- **Prometheus** — en production, on expose `runtime/metrics` au format Prometheus (via
  `client_golang`, dépendance externe). Standard de l'industrie pour l'alerting.

```go
import (
	_ "net/http/pprof" // greffe /debug/pprof/ sur le mux par défaut
	"net/http"
)
// go http.ListenAndServe("localhost:6060", nil) // /debug/vars + /debug/pprof/
```

---

## 🆕 Go 1.2x

- **1.25** — **`GODEBUG=decoratemappings=1`** (défaut **on** sous Linux) **nomme** les régions mémoire
  anonymes (VMA) : dans `/proc/PID/maps` et les outils, on lit `[anon: Go: heap]`, `[anon: Go: stacks]`…
  L'attribution mémoire devient lisible. (Non visible sous macOS/Windows.)
- **1.26** — la famille **`/sched/*`** est enrichie (`goroutines-created`, ventilation
  `running`/`runnable`/`waiting`, `threads/total`) — vérifié sur 1.26.4. Le profil **`goroutineleak`**
  ([Ch. 23](23-patterns-concurrence.md)) est aussi exposé via `/debug/pprof/goroutineleak`.
- **1.26** — l'UI web de **pprof** passe au **flame graph par défaut** ([Ch. 37](37-profiling-pprof.md)).

## ⚠️ Pièges

- **`ReadMemStats` en boucle** — chaque appel fige brièvement le monde. Pour du temps réel, préférez
  **`runtime/metrics`** (sans STW).
- **Exposer `/debug/pprof` publiquement** — c'est une **fuite d'informations** (et un vecteur DoS).
  Réservez-le à un port **interne** (`localhost`) ou derrière authentification.
- **Confondre `NumGoroutine` et le total `/sched`** — l'un compte l'utilisateur, l'autre **tout** ;
  l'écart (~6) est attendu.
- **`FreeOSMemory()` par réflexe** — forcer la restitution à l'OS provoque un GC complet ; laissez le
  runtime gérer, sauf besoin précis.

## ⚡ Performance

- **`runtime/metrics`** est conçu pour être échantillonné **fréquemment** : coût faible, pas de STW.
  C'est la bonne source pour un exporter.
- Réutilisez le **slice de `Sample`** entre les lectures plutôt que d'en réallouer un à chaque scrape.
- Surveiller `/gc/cycles/total` et `/sched/latencies` ([Ch. 27](27-garbage-collector.md),
  [Ch. 28](28-ordonnanceur-gmp.md)) suffit souvent à détecter une régression avant les utilisateurs.
- 🔁 Pour aller au-delà des compteurs : profils ([Ch. 37](37-profiling-pprof.md)) et traces
  ([Ch. 38](38-traces-flight-recorder.md)).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch29-observabilite-runtime
go test ./ch29-observabilite-runtime/...
go doc runtime/metrics            # parcourir les ~112 métriques disponibles
```

À essayer :

1. Listez **toutes** les métriques avec `metrics.All()` et affichez celles de la famille `/gc/`.
2. Branchez `net/http/pprof` + `expvar` sur `localhost:6060` et ouvrez `/debug/vars`.
3. Lancez un flux de goroutines éphémères et observez `/sched/goroutines-created` grimper sans fin.

---

## 📌 À retenir

- **`runtime`** : `NumGoroutine` (utilisateur), `ReadMemStats` (complet mais avec STW — à éviter en boucle).
- **`runtime/metrics`** est l'API **moderne** : stable, ~112 métriques, histogrammes, **sans STW** ;
  préférez-la dans tout code neuf.
- Les métriques **`/sched/*`** (enrichies en **1.26**) diagnostiquent la concurrence : ventilation
  `running`/`runnable`/`waiting`, cumul `goroutines-created` (fuites).
- **`runtime/debug`** : `ReadBuildInfo` (version/VCS), réglage du GC, `FreeOSMemory`, `Stack`.
- **Exposez** via **`expvar`** (`/debug/vars`), **`net/http/pprof`** (`/debug/pprof/`) ou **Prometheus** —
  jamais sur un port public.

## 🔁 Pour aller plus loin

- [Ch. 27 — Garbage collector](27-garbage-collector.md) : interpréter les métriques `/gc/*`.
- [Ch. 28 — L'ordonnanceur](28-ordonnanceur-gmp.md) : les métriques `/sched/*` en contexte.
- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : des compteurs aux profils détaillés.
- [Ch. 38 — Traces](38-traces-flight-recorder.md) : le comportement temporel fin.
- Doc : `go doc runtime/metrics`, `go doc runtime/debug.BuildInfo`, `go doc expvar`.
