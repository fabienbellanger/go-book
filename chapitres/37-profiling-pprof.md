# 37 — Profiling avec pprof

> **Objectif** — **Localiser** les coûts réels d'un programme : connaître les **profils** (CPU, tas,
> allocs, blocage, mutex, goroutines), les capturer avec **`runtime/pprof`** (programmes) ou
> **`net/http/pprof`** (services), puis les lire avec **`go tool pprof`** (`top`, `list`, `peek`,
> `web`, **flame graph**) en distinguant **`flat`** et **`cum`**.
>
> **Prérequis** — [Ch. 36](36-tests-benchmarks-fuzzing.md), [Ch. 26](26-allocation-escape.md), [Ch. 29](29-observabilite-runtime.md)

---

## Introduction

Un benchmark ([Ch. 36](36-tests-benchmarks-fuzzing.md)) répond à **« combien de temps ? »**. Un **profil**
répond à **« où ce temps part-il ? »**. La différence est décisive : l'intuition se trompe presque
toujours sur le point chaud. La règle d'or de cette Partie VI : **mesurez, puis localisez, puis
seulement optimisez**. Go embarque **`pprof`** — capture **et** visualisation — sans rien installer.
Code dans [`code/ch37-profiling-pprof/`](../code/ch37-profiling-pprof/).

---

## Les profils disponibles

`pprof` n'est pas qu'un profil CPU. Le runtime tient plusieurs **profils**, chacun répondant à une
question :

| Profil           | Question                              | Activation                           |
| ---------------- | ------------------------------------- | ------------------------------------ |
| **cpu**          | où part le **temps CPU** ?            | `StartCPUProfile` (échantillonné)    |
| **heap**         | qui **retient** la mémoire vivante ?  | toujours actif                       |
| **allocs**       | qui **alloue** (même libéré depuis) ? | toujours actif                       |
| **goroutine**    | que font **toutes les goroutines** ?  | toujours actif                       |
| **block**        | qui **bloque** sur canal/verrou ?     | `runtime.SetBlockProfileRate(n)`     |
| **mutex**        | qui se dispute les **verrous** ?      | `runtime.SetMutexProfileFraction(n)` |
| **threadcreate** | qui crée des **threads OS** ?         | toujours actif                       |

```
$ go run ./ch37-profiling-pprof
6 profils prédéfinis disponibles (allocs, block, goroutine, heap, mutex, threadcreate)
```

> ⚠️ **block** et **mutex** sont **désactivés par défaut** (ils ont un coût). On les arme explicitement
> avant de les capturer.

Le mode d'échantillonnage diffère radicalement entre les deux familles, et c'est ce qui explique leurs
réglages respectifs. Le profil **cpu** est cadencé par le **temps** : le runtime arme une interruption
périodique (signal `SIGPROF` sous Unix, **~100 Hz** par défaut, soit un échantillon toutes les ~10 ms)
qui capture la pile d'appels de la goroutine en cours d'exécution sur chaque thread actif — un code qui
ne tourne jamais n'apparaît jamais. Les profils **heap**/**allocs** sont cadencés par la **taille
allouée**, pas par le temps : un échantillon est pris en moyenne tous les **~512 Kio** alloués
(`runtime.MemProfileRate`), quelle que soit la durée que ça prend. Conséquence pratique : un programme
qui alloue beaucoup en peu de temps produit un profil tas **dense** même sur une fenêtre de 50 ms, alors
qu'un profil CPU sur la même durée n'aurait que **~5 échantillons** — inexploitable (voir le piège
« profiler trop court » plus bas).

**block** et **mutex** se ressemblent dans le tableau, mais répondent à deux questions différentes —
visibles dans la pile capturée par chacun :

- **block** enregistre la pile de la goroutine **qui attend** (`sync.Mutex.Lock`, `sync.RWMutex`,
  `sync.WaitGroup.Wait`, `sync.Cond.Wait`, envoi/réception sur canal, `select`...) — toute primitive de
  synchronisation confondue. `SetBlockProfileRate(n)` échantillonne en moyenne **un** événement de
  blocage tous les `n` nanosecondes passées bloquées ; `n = 1` capture tout.
- **mutex** enregistre la pile **à la fin de la section critique qui a causé l'attente** — c'est-à-dire
  l'appel à `Unlock` de la goroutine qui **tenait** le verrou, pas celle qui patientait. Si un verrou est
  gardé 1 s pendant que 5 goroutines attendent tout ce temps, son `Unlock` rapporte **5 s** de
  contention cumulée. `SetMutexProfileFraction(n)` échantillonne en moyenne 1 événement sur `n` ;
  `n = 1` capture tout.

En clair : **block** répond à « où mes goroutines perdent-elles du temps à attendre ? », **mutex**
répond à « quel verrou — et quelle ligne de code qui le détient — faut-il découper ? ».

## Capturer depuis un programme : `runtime/pprof`

Le patron est **`Start...` → travail → `Stop...`**. Pour le CPU, on profile une **fenêtre d'exécution** ;
pour le tas, on prend un **instantané** :

```go
// code/ch37-profiling-pprof/profiling.go
func CaptureCPUProfile(w io.Writer, work func()) error {
	if err := pprof.StartCPUProfile(w); err != nil {
		return err
	}
	defer pprof.StopCPUProfile()
	work()
	return nil
}

func CaptureHeapProfile(w io.Writer) error {
	runtime.GC()                    // statistiques à jour avant l'instantané
	return pprof.WriteHeapProfile(w)
}
```

`runtime.GC()` avant l'instantané n'est pas cosmétique : le profil tas ne reflète l'état qu'**à la
dernière fin de cycle de GC**. Sans ce forçage, les allocations très récentes — pas encore classées
vivantes ou mortes par le collecteur — biaiseraient le profil vers du **bruit transitoire** plutôt que
vers la rétention réelle.

Les autres profils s'obtiennent par **`pprof.Lookup(nom).WriteTo(w, debug)`** (`"goroutine"`, `"block"`,
`"mutex"`, `"allocs"`…). Le second paramètre n'est pas accessoire : `debug=0` écrit le format binaire
attendu par `go tool pprof` ; `debug=2`, pour le profil `"goroutine"` spécifiquement, écrit la pile
**complète** de chaque goroutine en texte lisible — exactement ce que renvoie
`/debug/pprof/goroutine?debug=2`, et le moyen le plus rapide de diagnostiquer un service **bloqué**
sans le redémarrer. Un profil pprof (`debug=0`) est un **protobuf compressé en gzip** (en-tête
`0x1f 0x8b`).

## Lire un profil CPU : `top`, `list`, `peek`, `flat` vs `cum`

Une fois `cpu.prof` écrit (`go run ./ch37-profiling-pprof profile`), on l'ouvre :

```
$ go tool pprof -top cpu.prof
      flat  flat%   sum%        cum   cum%
     2.31s 95.45% 95.45%      2.40s 99.17%  main.collatzSteps (inline)
     0.09s  3.72% 99.17%      0.09s  3.72%  runtime.asyncPreempt
     0.02s  0.83%   100%      2.42s   100%  main.HotCompute (inline)
```

La distinction **fondamentale** :

```
  flat = temps passé DANS la fonction elle-même
  cum  = temps passé dans la fonction ET tout ce qu'elle appelle (cumulé)

  main.HotCompute   flat=0.02s  cum=2.42s   <- ne fait presque rien soi-même,
                                                mais TOUT passe dans ses appelés
  main.collatzSteps flat=2.31s  cum=2.40s   <- LE point chaud : le temps est ICI
```

Trier par **`flat`** trouve **où le CPU brûle** ; trier par **`cum`** suit **le chemin** qui y mène.
Ensuite, **`list`** descend à la **ligne** :

```
$ go tool pprof -list=collatzSteps cpu.prof
         .          .     14:	steps := 0
     1.16s      1.19s     15:	for n > 1 {
     540ms      560ms     16:		if n%2 == 0 {
     610ms      650ms     19:			n = 3*n + 1
```

On voit la boucle (`for`) et la branche impaire (`3*n + 1`) concentrer le temps. **C'est ici, et nulle
part ailleurs, qu'une optimisation aurait un effet.**

> 💡 `(inline)` à côté de `collatzSteps` rappelle qu'elle a été **inlinée** dans `HotCompute`
> ([Ch. 39](39-compilation-inlining-pgo.md)) — `pprof` recompose néanmoins l'attribution par fonction.

`top` dit **quelle** fonction est chaude, `list` dit **quelle ligne**. Une troisième commande,
**`peek`**, situe une fonction dans son **contexte d'appel** — qui l'appelle, et qu'appelle-t-elle :

```
$ go tool pprof -peek=collatzSteps cpu.prof
----------------------------------------------------------+-------------
      flat  flat%   sum%        cum   cum%   calls calls% + context
----------------------------------------------------------+-------------
                                             2.42s   100% |   main.HotCompute (inline)
     2.35s 96.31% 96.31%      2.42s 99.18%                | main.collatzSteps
                                             0.07s  2.89% |   runtime.asyncPreempt
----------------------------------------------------------+-------------
```

La ligne du milieu est la fonction ciblée ; au-dessus, ses **appelants** (ici un seul, `HotCompute`,
responsable de 100 % de son temps) ; en dessous, ce avec quoi son exécution s'**entrelace** (ici
`asyncPreempt`, la préemption coopérative du runtime). Utile quand `top` désigne une fonction
**partagée** par plusieurs chemins d'appel et qu'il faut savoir **lequel** pèse le plus.

## Lire un profil tas

Le profil tas a **deux axes** : **`alloc_space`** (tout ce qui a été alloué, même libéré — révèle la
**pression** sur le GC) et **`inuse_space`** (ce qui est **encore vivant** — révèle les **fuites** et la
rétention). On choisit avec `-sample_index`, **indépendamment** du nom sous lequel le profil a été
capturé : `"heap"` et `"allocs"` enregistrent en réalité les **mêmes** quatre compteurs (`alloc_space`,
`alloc_objects`, `inuse_space`, `inuse_objects`) — seul **l'axe affiché par défaut** change quand on
omet `-sample_index` (`inuse_space` pour `"heap"`, `alloc_space` pour `"allocs"`). C'est pourquoi
`mem.prof`, écrit ici via `WriteHeapProfile` (donc capturé sous le nom « heap »), se lit tout aussi
bien en forçant `alloc_space` :

```
$ go tool pprof -sample_index=alloc_space -top mem.prof
      flat  flat%   sum%        cum   cum%
 1978.41MB 97.89% 97.89%  1978.41MB 97.89%  strings.FieldsFunc
      39MB  1.93% 99.82%       39MB  1.93%  internal/bytealg.MakeNoZero
```

Le verdict est sans appel : `wordFrequencies` alloue ~2 **Go** cumulés via `strings.Fields` (qui
construit un `[]string` à chaque appel). 🔁 Le remède — `strings.FieldsSeq`, un itérateur sans slice
([Ch. 18](18-iterateurs.md)) — se **mesure** ici avant de se décider.

Et pourtant, sous l'angle **`inuse_space`**, ce **même** `mem.prof` ne montre presque plus rien de
`wordFrequencies` :

```
$ go tool pprof -sample_index=inuse_space -top mem.prof
      flat  flat%   sum%        cum   cum%
 2051.06kB   100%   100%  2051.06kB   100%  runtime.mallocgc
```

**~2 Mio retenus contre ~2 Go alloués** : un facteur **~1000**. La raison tient au programme de démo
([`main.go`](../code/ch37-profiling-pprof/main.go)) : `_ = wordFrequencies(text, 20000)` **jette**
aussitôt la map calculée. Au moment du `runtime.GC()` qui précède l'instantané, plus rien ne la
référence : le GC l'a déjà réclamée. `alloc_space` voit le passage de ~2 Go par le tas ; `inuse_space`
ne voit que ce qui **subsiste** à l'instant précis de la capture — ici, rien de `wordFrequencies`, juste
quelques structures internes du runtime. ⚠️ C'est exactement la confusion à éviter : chercher une
**fuite** (« qu'est-ce qui reste accroché ? ») sur `alloc_space` afficherait les mêmes **~2 Go**
alarmants même en l'**absence totale** de fuite, puisque cet axe compte aussi ce qui a été légitimement
libéré.

## Profiler un service : `net/http/pprof`

Pour un service vivant, on greffe les endpoints HTTP par un **import à effet de bord** :

```go
import (
	"net/http"
	_ "net/http/pprof" // greffe /debug/pprof/ sur le mux par défaut
)

func init() {
	go http.ListenAndServe("localhost:6060", nil) // port INTERNE only
}
```

Puis on capture **à chaud**, sans redémarrer :

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30  # CPU sur 30 s
go tool pprof http://localhost:6060/debug/pprof/heap                # tas instantané
go tool pprof http://localhost:6060/debug/pprof/goroutine           # toutes les goroutines
```

> ⚠️ **Jamais sur un port public** : `/debug/pprof` divulgue le code chaud et offre un **vecteur de
> DoS**. `localhost` ou derrière authentification ([Ch. 29](29-observabilite-runtime.md)).

## Le flame graph

L'UI web (`go tool pprof -http=:8080 cpu.prof`) affiche un **flame graph**. Lecture :

```
  largeur d'une case = temps (cum) ; profondeur = pile d'appels

  +-------------------------------------------------------------+
  | main.main                                                   |  <- racine, 100 %
  +-------------------------------------------------------------+
  | main.HotCompute                                             |
  +-------------------------------------------------------------+
  | main.collatzSteps                                  |////////|  <- 95 % de large
  +----------------------------------------------------+--------+     = le point chaud
                                              asyncPreempt (3 %)
```

On cherche les cases **larges** (beaucoup de temps) et **profondes** (chaînes d'appels coûteuses).
Une case large tout en haut = un point chaud isolé ; une large « base » = un coût réparti.

---

## 🆕 Go 1.2x

- **1.26** — l'UI web de `pprof` ouvre désormais le **flame graph par défaut** (auparavant : le graphe
  d'appels). La vue la plus utile est en première page.
- **1.26** — profil expérimental **`goroutineleak`** ([Ch. 23](23-patterns-concurrence.md)) : il liste
  les goroutines **définitivement bloquées** (fuites), exposé via `/debug/pprof/goroutineleak`. Il est
  **gated** par `GOEXPERIMENT=goroutineleakprofile` — sans l'expérience, `pprof.Lookup("goroutineleak")`
  renvoie `nil` (vérifié sur 1.26.4).
- **1.21+** — le profil **`allocs`** par défaut de `go test -memprofile` distingue clairement
  `alloc_space` (pression GC) de `inuse_space` (rétention).

## ⚠️ Pièges

- **Confondre `flat` et `cum`** — optimiser une fonction à gros `cum` mais petit `flat` ne sert à rien :
  le temps est dans ses **appelés**. Visez le `flat`.
- **Profiler trop court** — un profil CPU de 100 ms n'a que ~10 échantillons : du bruit. Profilez
  **plusieurs secondes** de charge représentative.
- **Le mauvais profil pour le problème** — un profil **cpu** ne voit que le code **en train de
  s'exécuter** ; une fuite mémoire, c'est de la mémoire retenue mais **inactive**, invisible dans
  `cpu.prof` même en le scrutant longtemps. Pour une fuite : `heap`/`inuse_space` ; pour une fuite de
  **goroutines** : le profil `goroutine` (ou `goroutineleak`, 1.26 expérimental).
- **Oublier d'armer block/mutex** — sans `SetBlockProfileRate`/`SetMutexProfileFraction`, ces profils
  sont **vides**. À l'inverse, les armer au taux **maximal** (`rate=1`, `fraction=1` : tout capturer) en
  permanence sur un service à forte contention a un coût mesurable (voir ⚡ Performance plus bas) — on
  les active **le temps du diagnostic**, pas par défaut au démarrage.
- **`inuse` vs `alloc`** — chercher une fuite dans `alloc_space` (qui montre aussi le libéré) ou une
  pression GC dans `inuse_space` : on regarde le mauvais axe.
- **macOS** : le profil CPU peut afficher `runtime.kevent`/`pthread_cond_wait` (artefact
  d'échantillonnage). Concentrez-vous sur **vos** symboles ; au besoin, profilez sous Linux.

## ⚡ Performance

- Le profil CPU est **échantillonné** (~100 Hz) : coût négligeable, activable en production par fenêtres.
- `heap`/`allocs` sont **échantillonnés** aussi (1 sur ~512 Kio par défaut, `runtime.MemProfileRate`).
- **`block`/`mutex` au taux maximal coûtent cher** — `SetBlockProfileRate(1)` instrumente **chaque**
  événement de blocage, `SetMutexProfileFraction(1)` **chaque** contention : sur un service très
  concurrent, l'overhead devient mesurable. Démarrez avec un taux non maximal (ex.
  `SetMutexProfileFraction(5)`) ou n'armez ces profils que **ponctuellement**.
- **`cum` cumule des secondes-CPU, pas des secondes d'horloge** — avec plusieurs P actifs en parallèle,
  le total des échantillons d'un profil CPU peut **dépasser** la durée réelle de l'exécution : un `cum`
  de 8 s sur un programme qui a tourné 2 s n'est pas une anomalie sur une machine à 4+ cœurs occupés.
- **Capturez en production** : un profil de laboratoire ment souvent (données, charge, cache différents).
  `net/http/pprof` sur port interne est fait pour ça.
- 🔁 Les profils disent **où**, pas **quand** : pour la dimension **temporelle** (latence, scheduler, GC),
  passez aux **traces** ([Ch. 38](38-traces-flight-recorder.md)).

## 🧪 À tester soi-même

```bash
cd code/ch37-profiling-pprof
go run . profile                      # écrit cpu.prof et mem.prof
go tool pprof -top cpu.prof
go tool pprof -list=collatzSteps cpu.prof
go tool pprof -http=:8080 cpu.prof    # flame graph dans le navigateur
go tool pprof -sample_index=alloc_space -top mem.prof
```

À essayer :

1. Comparez `-sample_index=alloc_space` et `inuse_space` sur `mem.prof` : qui domine, et pourquoi ?
2. Remplacez `strings.Fields` par `strings.FieldsSeq` ([Ch. 18](18-iterateurs.md)) et reprofilez le tas.
3. Ouvrez le flame graph et retrouvez visuellement la case `collatzSteps` (95 % de large).

---

## 📌 À retenir

- Le benchmark dit **combien**, le **profil** dit **où**. Mesurer → localiser → optimiser, jamais
  l'inverse.
- **6 profils** par défaut (cpu via `StartCPUProfile`, heap, allocs, goroutine, block*, mutex*,
  threadcreate) ; `block`/`mutex` doivent être **armés**.
- **`runtime/pprof`** pour les programmes, **`net/http/pprof`** (`/debug/pprof/`, port **interne**) pour
  les services.
- **`go tool pprof`** : `top`, `list` (ligne à ligne), `peek` (contexte d'appel), `web`/flame graph.
  **`flat`** = temps dans la fonction, **`cum`** = avec ses appelés — optimisez le **`flat`**.
- Tas : **`alloc_space`** (pression GC) vs **`inuse_space`** (rétention/fuite). 🆕 1.26 : flame graph par
  défaut, profil `goroutineleak` (expérimental).

## 🔁 Pour aller plus loin

- [Ch. 38 — Traces & Flight Recorder](38-traces-flight-recorder.md) : la dimension temporelle fine.
- [Ch. 39 — Compilation & PGO](39-compilation-inlining-pgo.md) : un profil CPU **guide** la recompilation (PGO).
- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : lire un profil tas, c'est traquer les échappements.
- [Ch. 29 — Observabilité](29-observabilite-runtime.md) : des compteurs aux profils détaillés.
- Doc : `go tool pprof -h`, `go doc runtime/pprof`, `go doc net/http/pprof`.
