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
  brièvement le monde pour figer les compteurs : à **ne pas** appeler en boucle serrée. Ce court
  arrêt n'est pas un caprice de l'API : les compteurs sont tenus **par P** (pour éviter toute
  contention), et le seul moyen de les agréger en une photo cohérente est de geler leur progression
  le temps de les sommer.

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
1.26), histogrammes inclus, et **plus légère** que `ReadMemStats`. Chaque nom suit un format fixe,
**chemin:unité** (ex. `/memory/classes/heap/objects:bytes`) — le chemin situe la métrique dans une
famille (`/gc/`, `/sched/`, `/memory/`...), l'unité après `:` précise ce qui est compté (`bytes`,
`objects`, `gc-cycles`, `seconds`...). `metrics.All()` renvoie la liste complète avec sa description
et son **type** (`KindUint64`, `KindFloat64`, ou `KindFloat64Histogram` pour des distributions comme
les pauses GC) : c'est le moyen de **découvrir** les métriques disponibles sans dépendre d'une liste
figée, utile après une montée de version. On déclare les noms voulus, on appelle `Read` :

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

`GODEBUG` n'est pas qu'une liste de drapeaux de debug : c'est **le** mécanisme générique par lequel
le runtime et la bibliothèque standard exposent un comportement réglable **sans recompiler**, pour
deux usages bien distincts :

1. **Diagnostics** — déclencher une trace texte sur `stderr` (GC, ordonnanceur, init...) sans rien
   changer au comportement du programme. C'est l'usage qui nous occupe ici.
2. **Bascules de compatibilité** — revenir temporairement à un **ancien comportement** quand une
   nouvelle version de Go a changé un défaut. Exemple réel : Go 1.22 a changé le routage de
   `http.ServeMux` (méthodes et wildcards dans les patterns) ; `GODEBUG=httpmuxgo121=1` restaure
   l'ancien routage. C'est ce filet qui rend la « Go 1 promise » tenable malgré des changements de
   défaut : on monte de version sans rien casser **immédiatement**, le temps de migrer.

Les deux usages se pilotent de la même façon : une liste **séparée par des virgules**, lue **une
seule fois au démarrage** du programme — on peut donc combiner plusieurs sondes en un seul lancement :

```bash
GODEBUG=gctrace=1,schedtrace=1000 ./binaire   # GC et ordonnanceur tracés simultanément sur stderr
```

Pour une bascule de compatibilité, fixer la valeur **dans le module** évite de devoir penser à
repositionner la variable à chaque déploiement : `go mod edit -godebug=httpmuxgo121=1` ajoute une
ligne `godebug` à `go.mod` (alternative : une directive `//go:debug` en tête d'un fichier du
`package main`). Implicitement, la ligne `go 1.26` de `go.mod` fixe déjà **tous** les défauts de
compatibilité tels qu'ils étaient en 1.26 — c'est ce mécanisme qui permet aux défauts de changer
d'une version à l'autre sans rompre le code existant.

Diagnostics les plus utiles pour ce chapitre :

| `GODEBUG=...`       | Ce que ça montre                                  | Détail                            |
| ------------------- | ------------------------------------------------- | --------------------------------- |
| `gctrace=1`         | une ligne **par cycle de GC** (pauses, tas, goal) | [Ch. 27](27-garbage-collector.md) |
| `schedtrace=N`      | **photo de l'ordonnanceur** toutes les N ms       | [Ch. 28](28-ordonnanceur-gmp.md)  |
| `inittrace=1`       | coût d'**init** de chaque package                 | [Ch. 24](24-runtime-bootstrap.md) |
| `checkfinalizers=1` | diagnostique finalizers/cleanups (1.25)           | [Ch. 27](27-garbage-collector.md) |

> 💡 Chaque bascule de compatibilité est aussi un compteur **`runtime/metrics`** :
> `/godebug/non-default-behavior/<nom>:events` (ex. `httpmuxgo121`, `panicnil`) s'incrémente à chaque
> fois que l'ancien comportement est réellement exercé. En production, c'est le moyen de vérifier
> qu'une bascule héritée n'est **plus utilisée** avant de l'enlever du code.

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

## Quel outil pour quel besoin ?

Le chapitre a montré cinq familles de sondes ; elles ne se concurrencent pas, elles répondent à des
besoins différents :

| Outil                  | Coût                      | Fréquence d'usage typique              | À utiliser pour                                                            |
| ---------------------- | ------------------------- | -------------------------------------- | -------------------------------------------------------------------------- |
| `GODEBUG=...`          | nul à faible              | ponctuel (debug, incident, migration)  | une trace texte lisible, ou une bascule de compatibilité                   |
| `expvar`               | faible (sauf `memstats`)  | dashboard maison, debug rapide         | quelques compteurs JSON, zéro dépendance                                   |
| `runtime/metrics`      | très faible, **sans STW** | scrape **continu** (Prometheus, agent) | la **source** d'un exporter de monitoring en production                    |
| `runtime.ReadMemStats` | STW bref                  | rare, ponctuel                         | un snapshot complet ; compatibilité avec du code déjà existant             |
| `net/http/pprof`       | élevé pendant la capture  | à la demande, sur une anomalie repérée | profils détaillés (CPU, tas, goroutines) — [Ch. 37](37-profiling-pprof.md) |

En pratique, trois de ces outils s'enchaînent souvent dans un incident : **`runtime/metrics`** (déjà
scrapé en continu) signale l'anomalie, **`GODEBUG`** ou **`pprof`** la creusent ponctuellement pour en
trouver la cause.

---

## 🆕 Go 1.2x

- **1.25** — **`GODEBUG=decoratemappings=1`** (défaut **on** sous Linux) **nomme** les régions mémoire
  anonymes (VMA) : dans `/proc/PID/maps` et les outils, on lit `[anon: Go: heap]`, `[anon: Go: stacks]`…
  L'attribution mémoire devient lisible. (Non visible sous macOS/Windows.)
- **1.26** — la famille **`/sched/*`** est enrichie (`goroutines-created`, ventilation
  `running`/`runnable`/`waiting`, `threads/total`) — vérifié sur 1.26.4. Le profil expérimental
  **`goroutineleak`** ([Ch. 23](23-patterns-concurrence.md)) est exposé via
  `/debug/pprof/goroutineleak`, mais **gated** par `GOEXPERIMENT=goroutineleakprofile` : sans cette
  variable au build, le profil n'existe simplement pas (🔁 [Ch. 37](37-profiling-pprof.md)).
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
- **`expvar` n'est pas gratuit pour autant** — la variable `memstats`, publiée automatiquement sur
  `/debug/vars`, appelle en interne `runtime.ReadMemStats`. Scraper `/debug/vars` souvent revient donc
  à payer le **même** court arrêt du monde que l'appel direct : un piège classique quand on choisit
  `expvar` « parce que c'est plus simple à brancher » sans réaliser qu'il embarque l'ancienne API.
- **Une métrique introuvable échoue en silence** — si le nom passé à `metrics.Read` n'existe pas
  (faute de frappe, métrique renommée entre deux versions de Go), `Value.Kind()` vaut `KindBad` ; notre
  `readUint` (`code/ch29-observabilite-runtime/metrics.go`) renvoie alors `0` sans erreur, comme si la
  valeur réelle était nulle. Vérifiez les noms avec `metrics.All()` après toute montée de version
  majeure plutôt que de les recopier depuis une doc figée.

## ⚡ Performance

- **`runtime/metrics`** est conçu pour être échantillonné **fréquemment** : coût faible, pas de STW.
  C'est la bonne source pour un exporter.
- Réutilisez le **slice de `Sample`** entre les lectures plutôt que d'en réallouer un à chaque scrape.
- Les métriques **histogrammes** (`/gc/pauses:seconds`, `/sched/latencies:seconds`...) coûtent un peu
  plus cher à lire qu'un simple `uint64` : chaque échantillon transporte un jeu de **buckets**. Pour un
  exporter à haute fréquence, ciblez les histogrammes réellement exploités plutôt que `metrics.All()`
  en boucle.
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
4. Combinez deux sondes en un seul lancement sur un de vos programmes qui tourne plus longtemps que
   la démo : `GODEBUG=gctrace=1,schedtrace=1000 ./votre-binaire`, et confrontez les deux traces sur
   la même fenêtre de temps.

---

## 📌 À retenir

- **`runtime`** : `NumGoroutine` (utilisateur), `ReadMemStats` (complet mais avec STW — à éviter en boucle).
- **`runtime/metrics`** est l'API **moderne** : stable, ~112 métriques, histogrammes, **sans STW** ;
  préférez-la dans tout code neuf.
- Les métriques **`/sched/*`** (enrichies en **1.26**) diagnostiquent la concurrence : ventilation
  `running`/`runnable`/`waiting`, cumul `goroutines-created` (fuites).
- **`runtime/debug`** : `ReadBuildInfo` (version/VCS), réglage du GC, `FreeOSMemory`, `Stack`.
- **`GODEBUG`** a un double rôle : **diagnostics** (`gctrace`, `schedtrace`, `inittrace`...) et
  **bascules de compatibilité** (`httpmuxgo121`, `panicnil`...), ces dernières comptabilisées dans
  `runtime/metrics` via `/godebug/non-default-behavior/*:events`.
- **Exposez** via **`expvar`** (`/debug/vars`), **`net/http/pprof`** (`/debug/pprof/`) ou **Prometheus** —
  jamais sur un port public.

## 🔁 Pour aller plus loin

- [Ch. 27 — Garbage collector](27-garbage-collector.md) : interpréter les métriques `/gc/*`.
- [Ch. 28 — L'ordonnanceur](28-ordonnanceur-gmp.md) : les métriques `/sched/*` en contexte.
- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : des compteurs aux profils détaillés.
- [Ch. 38 — Traces](38-traces-flight-recorder.md) : le comportement temporel fin.
- Doc : `go doc runtime/metrics`, `go doc runtime/debug.BuildInfo`, `go doc expvar`.
