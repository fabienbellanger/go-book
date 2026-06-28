# Projet 3 — Pipeline concurrent : `pipe`

> **Objectif** — Construire un **pipeline concurrent générique et réutilisable**,
> puis l'appliquer à un outil réel (hachage SHA-256 de fichiers en parallèle).
> On y assemble les briques de la concurrence Go : **fan-out / fan-in**,
> `select`, **annulation par `context`**, **pression arrière** (canaux bornés),
> **`errgroup`** (propagation de la première erreur), **limitation de débit**,
> **arrêt propre** et **métriques** — le tout testé avec **`testing/synctest`**
> (temps virtuel + détection de fuite de goroutines).
>
> **Réinvestit** — [Ch. 19 Goroutines](../../chapitres/19-goroutines.md),
> [Ch. 20 Canaux & select](../../chapitres/20-channels-select.md),
> [Ch. 21 Synchronisation](../../chapitres/21-synchronisation.md),
> [Ch. 22 Context](../../chapitres/22-context.md),
> [Ch. 23 Patrons de concurrence](../../chapitres/23-patterns-concurrence.md),
> [Ch. 38 Traces & synctest](../../chapitres/38-traces-flightrecorder.md).

---

## 1. Cahier des charges

`pipe` lit une liste de fichiers (en arguments **ou** sur stdin, un par ligne)
et calcule leur **SHA-256 en parallèle**, façon `shasum -a 256` mais concurrent :

| Élément       | Choix                                                          |
| ------------- | -------------------------------------------------------------- |
| Source        | arguments, sinon stdin (`iter.Seq` paresseuse).                |
| Traitement    | `crypto/sha256` par fichier, lecture **annulable**.            |
| Concurrence   | `-j N` workers (défaut `GOMAXPROCS`).                          |
| Débit         | `-rate N` fichiers/seconde (0 = illimité).                     |
| Sortie        | `somme  chemin`, **triée** par chemin (déterministe).          |
| Erreurs       | un fichier illisible annule le pipeline (errgroup) ⇒ code `1`. |
| Observabilité | métriques sur stderr (traités / échecs / pic de concurrence).  |

```
$ pipe -j 4 a.txt b.txt c.txt
8ed3f6ad…  a.txt
f44e64e7…  b.txt
be9d587d…  c.txt
pipe : traités=3 échecs=0 pic_concurrence=3

$ find . -name '*.go' | pipe -rate 50
```

---

## 2. Architecture

```
   items (iter.Seq[I])                          out (<-chan O, borné)
        │                                              ▲
        ▼                                              │
   ┌─────────┐   in (chan I)    ┌──────────────────┐   │
   │ feeder  │ ───────────────► │  worker ×N       │ ──┘   fan-in
   │ (1 gor.)│   fan-out        │  fn(ctx, item)   │
   └─────────┘                  └──────────────────┘
        │                              │
        └────────── errgroup.WithContext(ctx) ──────────┘
               première erreur ⇒ annule gctx ⇒
               feeder et workers s'arrêtent (zéro fuite)
```

Tout vit dans un **`errgroup`** partageant un `context` : un canal interne `in`
relie le feeder aux workers (**fan-out**), un canal `out` borné collecte les
résultats (**fan-in**). La signature tient en une ligne :

```go
func Process[I, O any](
    ctx context.Context,
    items iter.Seq[I],
    fn Stage[I, O],
    cfg Config,
) (<-chan O, *Metrics, func() error)
```

> 💡 **Pourquoi une source `iter.Seq` plutôt qu'un canal d'entrée ?** Parce que
> le **feeder est interne** au pipeline : il alimente `in` _en surveillant
> `gctx.Done()`_. Si l'on exposait le canal d'entrée, le code appelant pourrait
> rester bloqué sur `in <- x` après une annulation — une **fuite de goroutine**
> classique. En possédant la source, `Process` garantit l'arrêt propre.

---

## 3. Les briques, une à une

### Fan-out / fan-in

Un seul **feeder** lit `items` et écrit dans `in` ; il est donc le seul à
**fermer** `in` (règle d'or des canaux). **N workers** lisent `in` et écrivent
dans `out`. Un `WaitGroup` interne détecte la fin des workers pour **fermer
`out`** — l'instant exact que `errgroup` n'expose pas :

```go
go func() { wg.Wait(); close(out) }() // out fermé quand tous les workers ont fini
```

### Pression arrière (_backpressure_)

`in` n'est **pas** bufferisé et `out` l'est faiblement (`-buffer`). Quand le
consommateur ralentit, `out` se remplit, les workers **bloquent** sur l'envoi,
cessent de lire `in`, et le feeder **bloque** à son tour : la source est tirée
au rythme de la consommation. Pas de file qui gonfle sans limite en mémoire.

### Annulation (`context`)

Chaque envoi/réception est un `select` avec `<-gctx.Done()`. À l'annulation
(signal, erreur, délai), feeder et workers retournent immédiatement. L'étape
elle-même reçoit le `ctx` : `hashFile` lit le fichier **par blocs** en
vérifiant `ctx.Err()`, donc un gros fichier n'empêche pas un arrêt rapide.

### `errgroup` : première erreur gagnante

`errgroup.WithContext` annule le `context` dès qu'**un** worker renvoie une
erreur ; `Wait()` restitue **la première**. Un fichier manquant arrête donc tout
le pipeline. _(Pour « continuer malgré les erreurs », renvoyer l'erreur dans le
résultat plutôt que depuis l'étape — voir § 7.)_

### Limitation de débit

`RateLimiter` (seau à jetons sur `time.Ticker`) impose un `Wait(ctx)` avant
chaque traitement, partagé par tous les workers : le débit global est plafonné
quel que soit `-j`.

### Métriques

`Metrics` agrège des compteurs **atomiques** (sans verrou) : traités, échecs,
en vol, et **pic de concurrence** (via une boucle compare-and-swap). `Snapshot()`
en donne une vue figée.

---

## 4. `testing/synctest` : tester le temps et les fuites

Les tests de concurrence sont d'ordinaire **lents** (vrais `sleep`) et
**indéterministes**. `testing/synctest` (stable en Go 1.25) règle les deux :

- **Horloge virtuelle** — dans la « bulle », `time` est simulé. Le test du
  limiteur vérifie que 5 éléments à 10/s prennent **exactement 500 ms… simulées**,
  donc instantanément en temps réel :

  ```go
  synctest.Test(t, func(t *testing.T) {
      lim := NewRateLimiter(10)
      start := time.Now()
      out, _, wait := Process(ctx, seq([]int{0,1,2,3,4}), identity,
          Config{Workers: 1, Limiter: lim})
      collect(out); wait()
      if time.Since(start) != 500*time.Millisecond { t.Fatal("…") }
  })
  ```

- **Détection de fuite** — quand la fonction de bulle retourne, `synctest`
  **échoue si une goroutine reste bloquée**. Le test « première erreur » lance
  100 éléments : si le feeder ne respectait pas l'annulation, il fuirait sur
  `in <- x` et `synctest` le signalerait. C'est notre filet `goroutineleak`.

Tous les tests tournent en plus sous **`-race`**.

---

## 5. Tests

```bash
cd projets/3-pipeline
go test -race ./...
```

- **`pipeline_test.go`** — chemin nominal, **première erreur** (annulation +
  non-fuite), **annulation** par le contexte appelant, **pic de concurrence**
  borné par `-j`. Les trois derniers sous `synctest`.
- **`limiter_test.go`** — débit exact en **temps virtuel**, annulation.
- **`cli_test.go`** — hachage de vrais fichiers (`t.TempDir`), lecture stdin,
  fichier manquant ⇒ code `1`, erreurs d'usage, `-version`.

> 🧪 **À tester soi-même** : ajouter une étape `-upper` qui met le nom en
> majuscules avant le hachage, et composer **deux** `Process` en série
> (la sortie de l'un = la source de l'autre) — un vrai pipeline multi-étapes.

---

## 6. Build & cross-compilation

```bash
make run ARGS="-j 8 *.go"   # lance en local
make build                  # bin/pipe (version = git describe)
make dist                   # dist/pipe-<os>-<arch> pour 5 plateformes
```

La seule dépendance externe est **`golang.org/x/sync/errgroup`** ; tout le reste
est dans la bibliothèque standard. `CGO_ENABLED=0` reste possible (binaire
statique).

---

## 7. Points de vigilance

- **Qui ferme le canal ?** Toujours **l'émetteur**, **une fois**. Ici : le feeder
  ferme `in`, le _closer_ (`wg.Wait`) ferme `out`. Fermer un canal côté
  récepteur, ou deux fois, panique ([Ch. 20](../../chapitres/20-channels-select.md)).
- **Drainer avant `wait()`** : le code appelant doit consommer **tout** `out`
  (`for v := range out`) **puis** appeler `wait()`. Sinon les workers bloquent
  sur l'envoi et le pipeline ne se termine jamais.
- **Annulation vs continuation** : renvoyer une erreur depuis l'étape **annule
  tout** (errgroup). Pour traiter au mieux et collecter les échecs, faire de
  l'erreur une **donnée** : `Stage[I, Result]` où `Result` porte `err error`.
- **Le ticker ne fait pas de rafale** : `RateLimiter` régule mais n'autorise pas
  de pic. Pour des politiques fines (burst, attente bornée), utiliser
  [`golang.org/x/time/rate`](https://pkg.go.dev/golang.org/x/time/rate).
- **`synctest` interdit le vrai monde** : pas d'I/O réseau/disque ni de
  goroutines externes dans une bulle. Les tests CLI (vrais fichiers) sont donc
  des tests normaux, hors bulle.

---

## 8. Pour aller plus loin

- **Étapes en série** : généraliser à une chaîne `Process → Process → …`, chaque
  étape sur son propre pool.
- **Reprise sur erreur** : variante `ProcessAll` qui ne s'arrête pas au premier
  échec et renvoie tous les résultats + erreurs.
- **Observabilité** : exposer `Metrics` en JSON sur un endpoint, ou via
  `log/slog` à intervalle régulier.
- **`x/time/rate`** pour un vrai _token bucket_ avec rafale et `Reserve`.
- **Profilage** (Projet 7) : tracer ce pipeline avec `runtime/trace` +
  **FlightRecorder** ([Ch. 38](../../chapitres/38-traces-flightrecorder.md)) pour
  visualiser fan-out et contention.

---

## 📌 À retenir

- Un pipeline = **source → fan-out (N workers) → fan-in**, le tout sous un
  `context` partagé par **`errgroup`** (première erreur ⇒ annulation globale).
- La **pression arrière** vient des **canaux bornés** : pas de file infinie.
- **Posséder la source** (`iter.Seq` + feeder interne) évite la fuite de
  goroutine sur `in <- x` après annulation.
- L'**émetteur** ferme le canal, **une seule fois** ; drainer `out` **avant**
  `wait()`.
- **`testing/synctest`** rend les tests de concurrence **rapides, déterministes**
  et **détecteurs de fuites** — sous `-race`.
