# Ch. 38 — Traces d'exécution & Flight Recorder

> **Objectif** — Comprendre le **comportement temporel** d'un programme (ordonnanceur, GC, latence,
> blocages) avec le **traceur d'exécution** : `runtime/trace`, `go test -trace`, les vues de
> **`go tool trace`**, les **tâches/régions** utilisateur ; et capturer **l'avant-incident** d'un
> évènement rare avec le **Flight Recorder** (🆕 1.25).
>
> **Prérequis** — [Ch. 37](37-profiling-pprof.md), [Ch. 28](28-ordonnanceur-gmp.md), [Ch. 27](27-garbage-collector.md)

---

## Introduction

Un **profil** ([Ch. 37](37-profiling-pprof.md)) agrège : il dit **où** part le temps, mais pas **quand**
ni **pourquoi** une requête a mis 200 ms. Pour ça, il faut une **trace d'exécution** : un journal
**horodaté à la nanoseconde** de tout ce que fait le runtime — création/blocage/réveil de goroutines,
syscalls, cycles de GC, démarrage/arrêt des P. La trace **reconstitue la chronologie**, indispensable
pour diagnostiquer une **latence** ou une **contention**. Code dans
[`code/ch38-traces-flightrecorder/`](../code/ch38-traces-flightrecorder/).

---

## Capturer une trace

Deux voies. Pour un **test/benchmark**, le flag suffit :

```bash
go test -trace=trace.out ./...
```

Pour un **programme**, on encadre une fenêtre d'exécution avec `runtime/trace` — même patron que pprof :

```go
// code/ch38-traces-flightrecorder/traced.go
func CaptureTrace(w io.Writer, work func()) error {
	if err := trace.Start(w); err != nil {
		return err
	}
	defer trace.Stop()
	work()
	return nil
}
```

Le fichier produit est un format binaire propre (`"go 1.26 trace…"` en tête), lu **uniquement** par
`go tool trace`.

## Lire la trace : `go tool trace`

```bash
go tool trace trace.out      # ouvre un navigateur avec plusieurs vues
```

L'outil offre des **vues** complémentaires :

| Vue                                   | Ce qu'elle révèle                                     |
| ------------------------------------- | ----------------------------------------------------- |
| **View trace by proc/thread**         | timeline par P : qui tourne, quand, les **STW** du GC |
| **Goroutine analysis**                | par goroutine : temps d'exécution vs **attente**      |
| **Scheduler latency profile**         | délai entre « prête » et « exécutée » (manque de P)   |
| **Network/Sync/Syscall blocking**     | où les goroutines **bloquent** (I/O, canal, verrou)   |
| **User-defined tasks / regions**      | **vos** annotations (voir ci-dessous)                 |
| **Minimum mutator utilization (MMU)** | part du CPU laissée à **votre** code par le GC        |

```
  Timeline (View trace by proc) — largeur = temps réel :

  P0  |##G1##|  STW  |##### G1 #####|      |##G4##|
  P1  |###### G2 ######|        |##### G3 #####|
  P2          |## G5 ##|  (idle)        |##G2##|
              ^                ^
              GC mark          goroutine en attente d'un P (scheduler latency)
```

## Annoter : tâches, régions, logs

Une trace brute est dense. Les **annotations utilisateur** y projettent **votre** vocabulaire métier :

- **Tâche** (`trace.NewTask`) — un intervalle de bout en bout, possiblement multi-goroutines (ex. « une
  requête »). Terminée par `task.End()`.
- **Région** (`trace.WithRegion`/`StartRegion`) — un sous-intervalle dans **une** goroutine (ex. « parse »,
  « compute »).
- **Log** (`trace.Log`) — un évènement ponctuel horodaté, avec catégorie.

```go
// code/ch38-traces-flightrecorder/traced.go
func processBatch(ctx context.Context, items []int) int {
	ctx, task := trace.NewTask(ctx, "batch") // tâche = intervalle nommé
	defer task.End()

	var parsed []int
	trace.WithRegion(ctx, "parse", func() { /* ... */ })  // région 1
	sum := 0
	trace.WithRegion(ctx, "compute", func() { /* ... */ }) // région 2
	trace.Log(ctx, "result", "lot traité")                 // évènement
	return sum
}
```

Dans `go tool trace`, les vues **User-defined tasks/regions** montrent alors la **durée de chaque
phase** et leur **distribution** — on voit immédiatement si « compute » domine, et si une requête lente
l'est à cause d'une phase précise.

> ⚠️ Régions et tâches ont un **coût** (un évènement chacune). Annotez les **frontières** (requête,
> phase), pas chaque ligne. Comme le rappelle la doc : « only a handful of unique region types ».

## Flight Recorder : capturer l'avant-incident (🆕 1.25)

Tracer **en continu** en production est trop coûteux (fichiers énormes). Mais les bugs intéressants sont
**rares** (un pic de latence toutes les heures). Le **Flight Recorder** résout le dilemme : il maintient
en mémoire un **anneau glissant** des **dernières secondes** de trace, et ne l'**écrit** que lorsque
**vous** le décidez — sur l'évènement rare.

```
  Tracer en continu : [================ tout, tout le temps ================]  trop cher

  Flight Recorder : anneau en mémoire des N dernières secondes
                    +----------------------+
              ... ->|  t-2s  ......  t-0s  |  (écrase le plus ancien)
                    +----------------------+
                          | pic de latence détecté -> WriteTo()
                          v
                     trace.out  (juste l'avant-incident, ~quelques Ko)
```

```go
// code/ch38-traces-flightrecorder/traced.go
func MonitorLatency(w io.Writer, threshold time.Duration, steps int, step func(i int)) (bool, error) {
	fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{
		MinAge:   2 * time.Second, // garder au moins les 2 dernières secondes
		MaxBytes: 1 << 20,         // plafond mémoire de la fenêtre
	})
	if err := fr.Start(); err != nil {
		return false, err
	}
	defer fr.Stop()

	for i := range steps {
		start := time.Now()
		step(i)
		if time.Since(start) >= threshold { // évènement rare
			_, err := fr.WriteTo(w) // fige les dernières secondes
			return err == nil, err
		}
	}
	return false, nil
}
```

```
$ go run ./ch38-traces-flightrecorder
Flight Recorder : capture déclenchée = true (3216 octets figés)
```

Le pic de latence (40 ms au tour 7) déclenche `WriteTo` : on obtient une trace **ciblée** des instants
qui ont **précédé** l'incident — exactement ce qu'il faut pour comprendre **pourquoi** la goroutine a
décroché (GC ? préemption ? attente d'un verrou ?). `WriteTo` est quasi **instantané** : il fige
simplement la fenêtre déjà en mémoire.

> 💡 Un seul Flight Recorder peut être actif à la fois ; il peut **coexister** avec un `trace.Start`
> classique. `Enabled()` indique s'il tourne (vérifié sur 1.26.4).

---

## 🆕 Go 1.2x

- **1.25** — **`runtime/trace.FlightRecorder`** : `NewFlightRecorder(cfg)` puis `Start`/`WriteTo`/`Stop`.
  Réglages **`MinAge`** (âge minimal de la fenêtre) et **`MaxBytes`** (plafond mémoire). C'est l'outil de
  **diagnostic en production** des évènements rares.
- **1.22** — le traceur d'exécution a été **réécrit** (format v2) : surcoût réduit (~1 %), traces
  découpables, base sur laquelle s'appuie le Flight Recorder.
- **1.21+** — `go tool trace` regroupe les vues par **tâches/régions** utilisateur, rendant les
  annotations métier directement exploitables.

## ⚠️ Pièges

- **Tracer en continu en prod** — volume ingérable et surcoût. Préférez le **Flight Recorder** (capture
  sur évènement) ou des fenêtres courtes.
- **Sur-annoter** — une région par ligne **noie** le signal et coûte cher. Annotez les **frontières**.
- **Lire une trace comme un profil** — la trace montre la **chronologie**, pas l'agrégat. Pour « quelle
  fonction brûle le CPU », c'est pprof ([Ch. 37](37-profiling-pprof.md)).
- **Oublier `task.End()` / `region.End()`** — une tâche jamais terminée fausse les durées. Utilisez
  `defer`.
- **Confondre tâche et région** — la **tâche** peut traverser plusieurs goroutines ; la **région** est
  **locale** à une goroutine.

## ⚡ Performance

- Le traceur v2 (1.22) coûte **~1 %** : acceptable par fenêtres, même en production.
- Le **Flight Recorder** garde tout **en mémoire** (`MaxBytes`) : aucun I/O tant que `WriteTo` n'est pas
  appelé — d'où son intérêt pour les chemins chauds.
- La trace est **la** source pour diagnostiquer une **latence p99** : elle montre l'attente
  (scheduler, GC, verrou) qu'un profil CPU **ne voit pas**.
- 🔁 Reliez à l'ordonnanceur ([Ch. 28](28-ordonnanceur-gmp.md)) et au GC ([Ch. 27](27-garbage-collector.md)) :
  la trace **donne à voir** leurs effets.

## 🧪 À tester soi-même

```bash
cd code/ch38-traces-flightrecorder
go run . trace               # écrit trace.out
go tool trace trace.out      # explore les vues (régions parse/compute visibles)
go test -trace=t.out ./...   # tracer les tests eux-mêmes
```

À essayer :

1. Ouvrez `go tool trace` et trouvez la vue **User-defined regions** : comparez la durée de « parse » et
   de « compute ».
2. Abaissez le `threshold` de `MonitorLatency` à 1 ms : la capture se déclenche-t-elle plus tôt ?
3. Repérez un cycle de **GC** dans « View trace by proc » et mesurez la pause STW.

---

## 📌 À retenir

- La **trace** reconstitue la **chronologie** (goroutines, GC, scheduler, blocages) — là où le profil
  n'agrège. Indispensable pour la **latence**.
- Capture : **`go test -trace`** ou **`trace.Start`/`Stop`** ; lecture **uniquement** via
  **`go tool trace`** (vues proc, goroutine, scheduler, blocking, MMU).
- **Annotez** avec **tâches** (intervalle, multi-goroutines), **régions** (phase locale) et **logs** —
  aux **frontières** seulement.
- 🆕 **Flight Recorder** (1.25) : anneau en mémoire des dernières secondes, **`WriteTo`** sur évènement
  rare — le diagnostic d'**avant-incident** en production, à coût mémoire borné (`MaxBytes`).
- Trace pour le **quand/pourquoi**, pprof pour le **où** : les deux sont complémentaires.

## 🔁 Pour aller plus loin

- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : l'agrégat (où part le temps).
- [Ch. 28 — L'ordonnanceur](28-ordonnanceur-gmp.md) : ce que la timeline des P donne à voir.
- [Ch. 27 — Garbage collector](27-garbage-collector.md) : repérer pauses et cycles dans la trace.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : diagnostiquer une contention.
- Doc : `go doc runtime/trace`, `go doc runtime/trace.FlightRecorder`, `go tool trace` (aide intégrée).
