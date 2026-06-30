# 23 — Patterns de concurrence, data races & tests concurrents

> **Objectif** — Composer des systèmes concurrents **corrects** et **testables** : pipelines,
> fan-out/fan-in, **worker pools** (parallélisme borné), `errgroup`, _rate limiting_ ; définir une
> **data race** et la traquer avec `-race` ; tester le **temps** et la concurrence avec
> `testing/synctest` ; détecter les **fuites** avec `goroutineleak`.
>
> **Prérequis** — [Ch. 19](19-goroutines.md), [Ch. 20](20-channels-select.md), [Ch. 21](21-synchronisation.md), [Ch. 22](22-context.md)

---

## Introduction

Les chapitres précédents ont posé les **briques** : goroutines, canaux, `sync`, `context`. Ce
chapitre les **assemble** en patterns éprouvés, puis montre comment **prouver** qu'un programme
concurrent est correct — car à l'œil nu, c'est impossible. Deux outils sont décisifs : le **détecteur
de courses** (`-race`) et, depuis Go 1.25, **`testing/synctest`** pour tester le temps de façon
déterministe. L'exemple est dans [`code/ch23-patterns-concurrence/`](../code/ch23-patterns-concurrence/).

---

## Pipelines

Un **pipeline** enchaîne des étapes reliées par des canaux : chaque étape lit en amont, transforme,
écrit en aval. Chaque maillon est **une goroutine**, et tout circule **en flux** — rien n'est
matérialisé entre les étapes ([Ch. 18](18-iterateurs.md) pour la version paresseuse mono-goroutine).

Pourquoi une goroutine par étape plutôt qu'une seule fonction qui enchaîne les transformations en
séquence ? Parce que les étapes se **chevauchent** : pendant que `stage(+1)` traite l'élément 2,
`stage(x2)` prépare déjà l'élément 3 — comme une chaîne de montage. Le **débit**, une fois le pipeline
« plein », dépasse celui d'un traitement séquentiel, même si la **latence** d'un élément isolé (le
temps pour traverser tous les maillons) ne change pas.

```
  source ----> stage(x2) ----> stage(+1) ----> sink
   <-chan        <-chan          <-chan        range/collect
  [1 2 3 4]     [2 4 6 8]       [3 5 7 9]
  (chaque maillon = 1 goroutine ; les étapes s'exécutent en parallèle, en flux)
```

```go
// code/ch23-patterns-concurrence/pipeline.go
func stage[A, B any](ctx context.Context, in <-chan A, f func(A) B) <-chan B {
	out := make(chan B)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- f(v):
			case <-ctx.Done():
				return // l'aval abandonne : on s'arrête sans fuir
			}
		}
	}()
	return out
}
```

Le `select` sur `ctx.Done()` est **vital** : sans lui, si le consommateur arrête de lire, la goroutine
se bloque sur `out <-` et **fuit** ([Ch. 19](19-goroutines.md)). Ce garde-fou doit être répété à
**chaque** maillon, pas seulement au dernier : un seul `stage` qui s'en passe suffit à fuir dès que
l'aval s'arrête plus haut dans la chaîne.

```go
// ❌ Sans select sur ctx.Done() : si plus personne ne lit out, la goroutine reste bloquée sur
// `out <- f(v)` pour toujours. Elle n'est pas « plantée », juste bloquée à jamais — donc jamais
// libérée par le GC : c'est exactement la fuite que GOEXPERIMENT=goroutineleakprofile détecte
// (plus bas dans ce chapitre).
func leakyStage[A, B any](in <-chan A, f func(A) B) <-chan B {
	out := make(chan B)
	go func() {
		defer close(out)
		for v := range in {
			out <- f(v) // bloque indéfiniment si le consommateur a cessé de lire
		}
	}()
	return out
}
```

## Fan-out / fan-in & worker pool

Quand une étape est **lente**, on la **parallélise** : plusieurs workers se partagent l'entrée
(**fan-out**), et on **fusionne** leurs sorties (**fan-in**, [Ch. 20](20-channels-select.md)).

```
              +--> worker --+
   source --->+--> worker --+---> fusion ---> sink
              +--> worker --+
   fan-out : N workers tirent du MÊME canal   fan-in : on fusionne leurs sorties
```

Mais « une goroutine par tâche » ne **borne** rien : un million de tâches = un million de goroutines
**simultanées**. Le coût n'est pas que la pile de chacune (~2 Ko, [Ch. 19](19-goroutines.md) — un
million de goroutines, c'est « seulement » ~2 Go) : c'est surtout la pression sur l'**ordonnanceur**
(un million de G à répartir sur quelques P, [Ch. 28](28-ordonnanceur-gmp.md)), sur le **GC** (qui doit
scanner chaque pile vivante), et — le plus souvent décisif — sur la **ressource avale** que ces
goroutines sollicitent toutes en même temps (connexions à une base de données, descripteurs de
fichiers, quota d'une API tierce) : sans plafond, rien ne l'empêche d'être saturée. Le **worker pool**
fixe un plafond explicite — `n` workers, pas plus — et c'est ce plafond qui borne le **parallélisme
effectif**, indépendamment du nombre de tâches :

```go
// code/ch23-patterns-concurrence/pipeline.go
func workerPool[T, U any](n int, items []T, f func(T) U) []U {
	out := make([]U, len(items))
	idx := make(chan int)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			for i := range idx { // chaque worker prend le prochain index libre
				out[i] = f(items[i]) // index distinct : aucune course
			}
		})
	}
	for i := range items {
		idx <- i
	}
	close(idx)
	wg.Wait()
	return out
}
```

> 💡 C'est le **parallélisme borné** : on dimensionne `n` selon les ressources (cœurs, connexions
> réseau), au lieu de submerger la machine. Le projet 3 en fait son cœur.

Le code ne comporte **aucune étape de fusion explicite** — c'est un fan-in « gratuit ». Chaque worker
écrit `out[i] = f(items[i])` à un **indice qui lui est propre** (`idx` ne distribue jamais deux fois le
même indice) ; comme `out` est pré-dimensionné, l'ordre du résultat final est celui de `items`, sans
avoir à fusionner des flux qui arriveraient dans le désordre :

```
   idx <- 0 1 2 3 4 5 6 7        un seul canal `chan int`, alimenté par la boucle principale
            |   |   |
        +---+   |   +---+
        v       v       v
     worker1  worker2  worker3   n workers tirent CHACUN le prochain indice libre
        |       |       |
        v       v       v
   out[i] = f(items[i])          écriture à un indice distinct : pas de course, pas de fusion à coder
```

Un vrai **fan-in en flux** (résultats consommés au fil de l'eau, ordre non garanti) demanderait à
l'inverse un canal de sortie unique, fermé par une goroutine de fusion dédiée une fois tous les workers
terminés — utile quand l'entrée n'est pas une slice connue d'avance mais un flux continu.

Deux risques à anticiper pour les tâches d'un pool :

- **Panique non isolée** : une `panic` dans `f` n'est **pas** confinée au worker qui l'a déclenchée —
  elle fait planter **tout le programme** ([Ch. 17](17-panic-recover.md)). Chaque worker doit poser sa
  propre frontière de `recover`, qui convertit la panique en erreur au lieu de tout faire tomber :
  ```go
  for i := range idx {
      func() {
          defer func() {
              if r := recover(); r != nil {
                  errs[i] = fmt.Errorf("tâche %d paniquée : %v", i, r) // isolée, pas de crash global
              }
          }()
          out[i] = f(items[i])
      }()
  }
  ```
- **Erreur silencieuse** : `workerPool` suppose ici que `f` ne peut pas échouer (`U` seul, pas
  `(U, error)`). Pour des tâches faillibles, on fait porter l'erreur par `U` (un type `Result[T]` avec
  un champ `Err`), ou on bascule sur `errgroup` (section suivante), qui annule le reste du pool dès la
  première erreur.

## `errgroup` : annuler à la première erreur

Lancer N tâches et **abandonner toutes** dès que l'une échoue est si courant qu'il existe un outil
dédié : `golang.org/x/sync/errgroup`. En voici une version **minimale en stdlib pure**, qui capture la
première erreur et **annule le contexte** partagé :

```go
// code/ch23-patterns-concurrence/group.go
func (g *Group) Go(f func() error) {
	g.wg.Go(func() {
		if err := f(); err != nil {
			g.once.Do(func() {
				g.err = err
				g.cancel() // 1re erreur : on annule, les autres tâches voient ctx.Done()
			})
		}
	})
}
```

Les tâches **coopératives** (qui surveillent `ctx.Done()`, [Ch. 22](22-context.md)) s'arrêtent alors
au plus tôt — inutile de finir un travail dont le résultat sera jeté.

La version réelle de `x/sync/errgroup` va plus loin : `(*Group).SetLimit(n)` borne le nombre de tâches
**actives en même temps** — `g.Go` se bloque alors jusqu'à ce qu'une place se libère. C'est un worker
pool et un errgroup combinés en un seul outil : parallélisme **borné** et annulation à la **première
erreur**.

## Rate limiting

Pour ne pas saturer un service en aval, on **cadence** : au plus une action par intervalle. Un
`time.Ticker` fournit un « jeton » périodique.

```go
// code/ch23-patterns-concurrence/ratelimit.go
func rateLimited(ctx context.Context, items []int, every time.Duration, f func(int)) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for _, it := range items {
		select {
		case <-tick.C:
			f(it)
		case <-ctx.Done():
			return
		}
	}
}
```

Le canal `tick.C` est **bufferisé à 1** : si `f` met plus longtemps que `every` à s'exécuter, les tops
manqués sont **perdus**, pas mis en file — le rythme ne s'accélère jamais pour « rattraper » un retard
(toujours `tick.Stop()`, comme ici en `defer` ; détail des timers au [Ch. 44](44-temps.md)). Ce ticker
cadence à **intervalle fixe**, mais n'autorise aucune **rafale** : impossible de consommer trois jetons
d'un coup même s'ils se sont accumulés. Pour un débit qui tolère des pics ponctuels, la référence est
`golang.org/x/time/rate` — algorithme du **seau de jetons** (_token bucket_) — avec une limite **et**
une capacité de rafale configurables séparément.

## Data races & le détecteur de courses

Une **data race** survient quand **deux goroutines accèdent à la même mémoire en même temps**, qu'**au
moins un** accès est une **écriture**, et qu'**aucune synchronisation** ne les ordonne. Le résultat est
**indéfini** ([Ch. 25](25-modele-memoire.md)) : pas « une vieille valeur », mais un comportement
**imprévisible** — et souvent invisible en test.

```go
// COURSE : n++ est lecture + écriture ; deux goroutines y accèdent sans synchronisation.
var n int
go func() { n++ }()
go func() { n++ }()
```

Cela reste une course même avec `GOMAXPROCS=1` : la **préemption** peut interrompre une goroutine entre
la lecture et l'écriture de `n`, exactement comme sur plusieurs cœurs. Limiter le parallélisme **masque**
la probabilité d'observer la course, il ne la **supprime** pas ([Ch. 28](28-ordonnanceur-gmp.md)).

On ne les repère pas à l'œil : on les **instrumente**. Le **détecteur de courses** (`-race`) signale
tout accès concurrent non synchronisé **observé à l'exécution** :

```
$ go test -race ./...
==================
WARNING: DATA RACE
Write at 0x00c0000b4010 by goroutine 8:
  ...
Previous write at 0x00c0000b4010 by goroutine 7:
  ...
==================
```

> 📌 Lancez **toujours** vos tests concurrents avec `-race` (en CI aussi). Il ne trouve que les
> courses **réellement exécutées** — d'où l'importance de tests qui **exercent** la concurrence. Coût :
> 2–10× plus lent, mémoire accrue ; réservé aux tests, pas à la production.

## Deadlocks : les causes & la détection

Le pendant de la course, c'est l'**interblocage** : des goroutines s'attendent en cercle, plus rien
n'avance. Quatre causes couvrent l'essentiel :

1. **Verrous en ordres opposés** (« AB-BA ») : g1 prend `A` puis `B`, g2 prend `B` puis `A`. Correctif
   universel : **un ordre de verrouillage global** (toujours verrouiller dans le même ordre).
2. **Mutex non réentrant** : reverrouiller dans la **même** goroutine bloque (séparez méthode publique
   qui verrouille et logique privée « déjà verrouillée »).
3. **Canal sans issue** : envoi non bufferisé sans receveur, `range` sur un canal **jamais fermé**, ou
   canal `nil` (bloque à jamais). Le **producteur ferme** ; bornez par `select` + `ctx.Done()`.
4. **`WaitGroup` mal orchestré** : `Add` **dans** la goroutine au lieu d'avant (préférez `wg.Go`, 1.25).

```go
// ✅ Ordre global : on verrouille toujours le plus petit id d'abord -> pas d'AB-BA.
func Transfer(from, to *Account, amount int64) {
	if from == to { return }
	first, second := from, to
	if first.id > second.id { first, second = second, first }
	first.mu.Lock();  defer first.mu.Unlock()
	second.mu.Lock(); defer second.mu.Unlock()
	from.balance -= amount; to.balance += amount
}
```

> ⚠️ Le runtime n'affiche `fatal error: all goroutines are asleep - deadlock!` que si **TOUTES** les
> goroutines sont bloquées. Un deadlock **partiel** (le `main` tourne) **fige sans message** : on le
> diagnostique en **vidant les piles** des goroutines (`SIGQUIT` / `kill -QUIT`, ou
> `/debug/pprof/goroutine?debug=2`).

> 🔁 Le catalogue complet (data races + deadlocks), les correctifs côte à côte et la **checklist
> pre-merge** sont dans l'[Annexe H — Concurrence sûre](../annexes/H-concurrence-sure.md).

## Tester le temps : `testing/synctest` (1.25)

Tester un timeout, un _rate limiter_, un _retry_ avec backoff... implique d'**attendre** — tests
**lents** et **instables**. `testing/synctest` (GA en 1.25) exécute le test dans une **bulle isolée**
dotée d'une **horloge virtuelle** : le temps n'avance que lorsque **toutes** les goroutines de la bulle
sont **durablement bloquées**, et il avance alors **instantanément**. Aucun `GOEXPERIMENT` n'est requis
depuis la GA — contrairement à `goroutineleak` un peu plus loin, qui en réclame toujours un.

```go
// code/ch23-patterns-concurrence/synctest_test.go
func TestRateLimitedVirtualTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var count atomic.Int64
		start := time.Now()
		rateLimited(context.Background(), []int{1, 2, 3, 4, 5}, 100*time.Millisecond,
			func(int) { count.Add(1) })

		if elapsed := time.Since(start); elapsed != 500*time.Millisecond {
			t.Errorf("temps virtuel = %v ; attendu 500ms", elapsed) // EXACT, en 0 s réel
		}
	})
}
```

Cinq appels espacés de 100 ms « prennent » **500 ms** de temps **virtuel** — vérifié **exactement**,
en **0 s réel**. `synctest.Wait()` complète l'outil : il bloque jusqu'à ce que toutes les autres
goroutines de la bulle soient bloquées, pour observer un état stable sans `time.Sleep`.

## Détecter les fuites : `goroutineleak` (1.26)

Une goroutine **bloquée à jamais** fuit en silence ([Ch. 19](19-goroutines.md)). Go 1.26 ajoute un
profil **expérimental** qui les **liste** — celles qui sont bloquées **et inaccessibles** (plus aucun
moyen de les réveiller) :

```go
import "runtime/pprof"

p := pprof.Lookup("goroutineleak")     // nil sans l'expérience
_ = p.WriteTo(os.Stdout, 1)
```

```
$ GOEXPERIMENT=goroutineleakprofile go run .
goroutineleak profile: total 1
1 @ ...
#  main.leak.func1+0x23  .../main.go:14   (<-ch : bloqué pour toujours)
```

Vérifié sur go1.26.4 : il pointe la **ligne exacte** du blocage. Sans `GOEXPERIMENT=goroutineleakprofile`,
`pprof.Lookup("goroutineleak")` renvoie `nil`. 🔁 [Ch. 29](29-observabilite-runtime.md).

---

## 🆕 Go 1.2x

- **1.25** — **`testing/synctest`** passe en **GA** : `synctest.Test(t, f)` (bulle + horloge virtuelle)
  et `synctest.Wait()` (attendre que la bulle se stabilise). Fin des tests temporels lents et instables.
- **1.26** — profil **`goroutineleak`** (expérimental, `GOEXPERIMENT=goroutineleakprofile`) :
  identifie les goroutines bloquées à jamais. Aussi exposé via `/debug/pprof/goroutineleak`.
- **1.25** — l'analyzer **`go vet waitgroup`** ([Ch. 21](21-synchronisation.md)) attrape un `Add` mal
  placé, source classique de bug de pool.

## ⚠️ Pièges

- **Fuite de pipeline** : un maillon bloqué sur `out <-` parce que l'aval a cessé de lire. Toujours un
  `select` avec `ctx.Done()` (ou un canal `done`).
- **Worker pool non borné** : « une goroutine par tâche » n'est pas un pool. Fixez `n`.
- **Panique non isolée dans un worker** : une tâche qui panique sans `recover` **propre à elle** ne
  fait pas perdre qu'elle-même — elle arrête **tout le programme** ([Ch. 17](17-panic-recover.md)).
- **Oublier `-race`** : une course peut passer 1000 tests verts puis casser en prod. `-race` en CI.
- **Tester avec de vrais `time.Sleep`** : lent et instable. Préférez `synctest` (temps virtuel).
- **`errgroup` sans coopération** : si les tâches n'écoutent pas `ctx.Done()`, l'annulation ne les
  arrête pas — elle ne fait que **signaler**.

## ⚡ Performance

- Le bon `n` d'un **worker pool** dépend de la nature des tâches : ~`GOMAXPROCS` pour du **CPU**, bien
  **plus** pour de l'**I/O** (les goroutines en attente réseau ne consomment pas de cœur,
  [Ch. 28](28-ordonnanceur-gmp.md)). Mesurez.
- Un **pipeline** non bufferisé synchronise à chaque valeur (~185 ns/maillon, [Ch. 20](20-channels-select.md)).
  Pour de gros volumes, **lotissez** (envoyez des slices) ou bufferisez les canaux.
- `workerPool` pré-dimensionne `out` (`make([]U, len(items))`) plutôt que d'`append`-er sous mutex :
  zéro réallocation, zéro contention d'écriture. La seule façon d'`append`-er en parallèle sans course
  serait de protéger l'opération entière, ce qui **sérialiserait** ce que le pool cherche justement à
  paralléliser.
- **`-race`** ralentit l'exécution **2–10×** : outil de test, pas de production.
- 🔁 Benchmarks et `benchstat` au [Ch. 36](36-tests-benchmarks-fuzzing.md).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch23-patterns-concurrence
go test -race ./ch23-patterns-concurrence/...          # courses + tests synctest
go test -run TestRateLimitedVirtualTime -v ./ch23-patterns-concurrence/...
```

À essayer :

1. Écrivez le programme « course » ci-dessus, lancez `go test -race`, lisez le rapport.
2. Ajoutez à `workerPool` la prise en charge d'un `context` pour l'annulation à mi-parcours.
3. Compilez un programme qui fuit une goroutine avec `GOEXPERIMENT=goroutineleakprofile` et dumpez
   `goroutineleak`.

---

## 📌 À retenir

- **Pipeline** = maillons reliés par canaux, en flux ; **fan-out/fan-in** parallélise une étape lente ;
  un **worker pool** **borne** le parallélisme (`n` workers) — chaque tâche doit isoler ses propres
  erreurs et paniques.
- `errgroup` (ou son équivalent) **annule** tout à la première erreur — à condition que les tâches
  **écoutent** `ctx.Done()`.
- Une **data race** = accès concurrent non synchronisé, au moins une écriture = comportement
  **indéfini**. **`-race`** la traque ; activez-le en CI.
- **`testing/synctest`** (1.25) teste le **temps** de façon **déterministe et instantanée** (horloge
  virtuelle).
- **`goroutineleak`** (1.26, expérimental) liste les goroutines **bloquées à jamais**.

## 🔁 Pour aller plus loin

- [Ch. 17 — Panic & recover](17-panic-recover.md) : isoler la panique d'une tâche de pool, sans faire
  tomber le programme entier.
- [Ch. 25 — Modèle mémoire](25-modele-memoire.md) : ce que « non synchronisé » veut dire formellement.
- [Ch. 28 — L'ordonnanceur](28-ordonnanceur-gmp.md) : pourquoi l'I/O permet bien plus de goroutines.
- [Ch. 29 — Observabilité](29-observabilite-runtime.md) : profils `goroutine`/`goroutineleak`, métriques.
- [Ch. 36 — Benchmarks](36-tests-benchmarks-fuzzing.md) : mesurer un système concurrent.
- [Annexe H — Concurrence sûre](../annexes/H-concurrence-sure.md) : règles d'or, catalogue races &
  deadlocks avec correctifs, mode opératoire de détection, checklist pre-merge.
- Projet 3 (pipeline / worker pool) : la synthèse de tout ce chapitre.
