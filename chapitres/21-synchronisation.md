# 21 — Primitives de synchronisation

> **Objectif** — Protéger un **état partagé** quand les canaux ne suffisent pas : `sync.Mutex` /
> `RWMutex`, `WaitGroup` (+ `Go`, 1.25), `Once`/`OnceValue`, `sync/atomic` (compteurs et
> `atomic.Pointer[T]`), `sync.Pool`. Savoir **lequel** choisir et à **quel coût**.
>
> **Prérequis** — [Ch. 19 — Goroutines](19-goroutines.md), [Ch. 20 — Channels](20-channels-select.md), [Ch. 16 — `defer`](16-defer.md)

---

## Introduction

« Partager la mémoire en communiquant » ([Ch. 20](20-channels-select.md)) est l'idéal — mais parfois
plusieurs goroutines doivent **vraiment** lire et écrire **la même** donnée (un compteur, un cache,
une config). Il faut alors **sérialiser** les accès, sinon c'est une **course** (_data race_,
[Ch. 23](23-patterns-concurrence.md)) : comportement indéfini, bugs non reproductibles.

Le package `sync` et `sync/atomic` fournissent ces garde-fous. Règle de choix : **canal** pour
transférer/signaler, **mutex** pour protéger une section de code, **atomic** pour une seule valeur.
L'exemple est dans [`code/ch21-synchronisation/`](../code/ch21-synchronisation/).

Ce chapitre couvre en réalité **deux problèmes distincts**, à ne pas confondre : **l'exclusion
mutuelle** (combien de goroutines ont le droit d'être dans une section de code à la fois ? — réponse
de `Mutex`, `RWMutex`, `atomic`) et la **signalisation/attente** (une goroutine doit-elle patienter
qu'un événement survienne, ou que d'autres aient terminé ? — réponse de `WaitGroup`, `Once`, `Cond`).
Un `Mutex` ne dit jamais « attends que telle chose arrive » ; un `WaitGroup` ne protège aucune
donnée. Les deux familles se combinent souvent dans une même structure.

---

## `sync.Mutex` : exclusion mutuelle

Un `Mutex` garantit qu'**un seul** goroutine est dans la **section critique** à la fois. On verrouille
(`Lock`), on travaille, on déverrouille (`Unlock`) — toujours via `defer` pour ne jamais l'oublier,
même en cas de panique ([Ch. 16](16-defer.md)).

```go
// code/ch21-synchronisation/counters.go
type SafeCounter struct {
	mu sync.Mutex
	n  int64
}

func (c *SafeCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++ // protégé : un seul incrément à la fois
}
```

> ⚠️ Un `Mutex` ne doit **jamais être copié** après usage. Une méthode à **récepteur valeur** copie la
> struct (et son mutex) : utilisez un **récepteur pointeur**. `go vet` (analyzer `copylocks`) le
> détecte.

**Pourquoi copier casse tout.** Un `Mutex` n'est pas un simple booléen « verrouillé / libre » : il
contient un état interne (un compteur de goroutines en attente, un sémaphore pour les réveiller) qui
n'a de sens qu'à **une seule adresse mémoire**. Copier la struct qui le contient duplique cet état à
un instant donné — l'original et la copie deviennent deux verrous **indépendants**, qui ne se voient
plus l'un l'autre :

```go
func (c Counter) Inc() { // BUG : récepteur valeur -> copie c.mu à chaque appel
	c.mu.Lock()         // verrouille LA COPIE, pas le mutex partagé par les autres appelants
	defer c.mu.Unlock()
	c.n++                // et cet incrément est lui aussi perdu : c est jetée au retour
}
```

Deux goroutines qui croient protéger « la même » donnée via un `Mutex` copié peuvent toutes les deux
obtenir un `Lock` en même temps : l'exclusion disparaît silencieusement, **sans erreur de
compilation**. `go vet ./...` détecte ce cas précis (récepteur valeur), et aussi les cas plus
insidieux où une struct contenant un `Mutex` est passée par valeur à une fonction, ou stockée par
valeur dans une slice/map.

## `sync.RWMutex` : lecteurs multiples

Quand les **lectures** dominent, `RWMutex` laisse **plusieurs lecteurs** entrer en parallèle (`RLock`)
et ne réserve l'exclusivité que pour une **écriture** (`Lock`) :

```go
// code/ch21-synchronisation/state.go
func (r *Registry) Get(key string) (int, bool) {
	r.mu.RLock()         // plusieurs Get peuvent s'exécuter en même temps
	defer r.mu.RUnlock()
	v, ok := r.m[key]
	return v, ok
}
func (r *Registry) Set(key string, val int) {
	r.mu.Lock()          // exclusif : bloque lecteurs ET écrivains
	defer r.mu.Unlock()
	r.m[key] = val
}
```

```
   RLock  G1  [-----------lit-----------]
   RLock  G2       [-----------lit-----------]      G1, G2, G3 en parallele :
   RLock  G3            [-----------lit-----------]  RLock n'exclut pas RLock
                                                   |
   Lock   G4                                      [--------ecrit--------]
                                                   ^ attend la fin de TOUS les RLock en cours
                                                                             |
   RLock  G5                                                                [----lit----]
                                                                             ^ attend la fin du Lock
```

> ⚠️ `RWMutex` n'est **pas réentrant** : tenter de prendre le `Lock` en tenant déjà un `RLock`
> (« montée en grade ») **interbloque**. Il n'est gagnant que si les lectures sont **nombreuses et
> longues** ; sous écritures fréquentes, un `Mutex` simple est souvent plus rapide.

## `sync.WaitGroup` & `WaitGroup.Go` (1.25)

Un `WaitGroup` attend la fin d'un groupe de goroutines (déjà croisé au [Ch. 19](19-goroutines.md)).
Contrairement à un `Mutex`, il **ne protège aucune donnée** : c'est un compteur thread-safe dont la
méthode `Wait` bloque tant qu'il n'est pas revenu à zéro — l'outil de la **signalisation** (« tout le
monde a-t-il fini ? »), pas de l'exclusion mutuelle. Le schéma historique — `Add(1)` / `go` /
`defer Done()` — est **verbeux et fragile** (un `Add` mal placé casse tout). Go 1.25 ajoute
**`WaitGroup.Go`** qui fait les trois d'un coup :

```go
// code/ch21-synchronisation/counters.go
func runConcurrently(n int, fn func()) {
	var wg sync.WaitGroup
	for range n {
		wg.Go(fn) // = wg.Add(1); go func(){ defer wg.Done(); fn() }()
	}
	wg.Wait()
}
```

> 🆕 1.25 — l'analyzer **`go vet waitgroup`** signale le piège classique : `wg.Add(1)` appelé **depuis**
> la goroutine (`WaitGroup.Add called from inside new goroutine`), où il s'exécute trop tard. `Go`
> rend ce bug impossible.

## `sync.Once` & `OnceValue` (1.21)

Pour une **initialisation unique** (config, singleton), `sync.Once` garantit qu'une fonction ne
s'exécute **qu'une fois**, même sous accès concurrent. Go 1.21 ajoute les enveloppes **`OnceFunc`** et
**`OnceValue`**, plus directes :

```go
// code/ch21-synchronisation/state.go
var config = sync.OnceValue(expensiveInit) // expensiveInit ne tourne qu'une fois

// 100 appels concurrents -> 1 seule exécution réelle (vérifié : loadCount == 1)
runConcurrently(100, func() { _ = config() })
```

> ⚠️ Le comportement en cas de **panique** diffère entre les deux API. Avec `Once.Do(f)` : si `f`
> panique, le `Once` est quand même marqué « fait » — les appels suivants à `Do` renvoient
> silencieusement, **sans** ré-exécuter `f` ni repaniquer. Avec `OnceFunc`/`OnceValue`/`OnceValues` :
> si `f` panique, **chaque** appel suivant **repanique avec la même valeur**, indéfiniment. Une
> initialisation qui peut échouer doit en tenir compte : `OnceValue` ne donne jamais l'illusion d'un
> succès après un échec, mais ne retente pas non plus tout seul.

## `sync/atomic` : sans verrou

Pour **une seule valeur**, les types atomiques (`atomic.Int64`, `Bool`, `Uint64`...) effectuent
lecture/écriture/incrément de façon **indivisible**, sans verrou — plus simple et plus rapide qu'un
mutex :

```go
// code/ch21-synchronisation/counters.go
type AtomicCounter struct{ n atomic.Int64 }

func (c *AtomicCounter) Inc()         { c.n.Add(1) }   // incrément atomique
func (c *AtomicCounter) Value() int64 { return c.n.Load() }
```

### `atomic.Pointer[T]` : échanger un état entier

`atomic.Pointer[T]` publie un **pointeur** atomiquement : les lecteurs obtiennent la version courante
**sans jamais bloquer**, et un écrivain en **publie** une nouvelle d'un seul `Store`. Parfait pour une
configuration **rechargée à chaud** :

```go
// code/ch21-synchronisation/state.go
type Config struct{ current atomic.Pointer[Settings] }

func (c *Config) Load() *Settings   { return c.current.Load() }  // lecture sans verrou
func (c *Config) Store(s *Settings) { c.current.Store(s) }       // publication atomique
```

> ⚠️ Mélanger un accès **atomique** et un accès **ordinaire** à la même variable est une **course**.
> Si une donnée est atomique, **tous** ses accès doivent passer par les méthodes `atomic`. Les types
> `atomic.T` (1.19+) évitent aussi l'ancien piège d'alignement des fonctions `atomic.AddInt64`.

### Quel outil pour quel besoin ?

| Outil          | Protège                                                       | Coût relatif (sous contention)                                        | Choisir quand…                                   |
| -------------- | ------------------------------------------------------------- | --------------------------------------------------------------------- | ------------------------------------------------ |
| `sync/atomic`  | **une seule** valeur (mot machine, pointeur)                  | le plus bas (~50 ns)                                                  | compteur, drapeau, configuration publiée en bloc |
| `sync.Mutex`   | une section critique (plusieurs champs liés par un invariant) | moyen (~140 ns)                                                       | état composite à modifier de façon cohérente     |
| `sync.RWMutex` | une section critique à **lecture dominante**                  | lecture rapide (~65 ns), écriture plus chère qu'un `Mutex`            | cache/registre lu très souvent, écrit rarement   |
| canal (`chan`) | rien directement : **transfère la propriété** d'une valeur    | le plus élevé (~185 ns, rendez-vous, [Ch. 20](20-channels-select.md)) | coordination, pipeline, signal d'arrêt           |

La distinction `atomic`/`Mutex` vs **canal** n'est pas qu'une question de coût : un canal fait passer
une donnée d'un propriétaire à l'autre (jamais deux goroutines n'y accèdent **en même temps**), alors
qu'un mutex protège une donnée que plusieurs goroutines **partagent réellement**. Et `atomic` ne
protège qu'**une** valeur : dès que plusieurs champs doivent rester cohérents **entre eux** (ex. un
solde et son historique), un `Mutex` est obligatoire — deux `atomic` indépendants peuvent être lus à
des instants différents, donc incohérents l'un par rapport à l'autre.

## `sync.Pool` : recycler le jetable

`sync.Pool` met en cache des objets **temporaires** pour réduire les allocations et la pression GC.
`Get` renvoie un objet recyclé (ou en crée un via `New`), `Put` le rend :

```go
// code/ch21-synchronisation/pool.go
var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func joinInts(nums []int, sep string) string {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()            // un objet recyclé peut être sale
	defer bufPool.Put(b)
	// ... écrire dans b ...
}
```

> ⚠️ Le GC **vide** le pool sans préavis : n'y mettez **que** du jetable (jamais des connexions, des
> objets à état durable). Et `Get` peut renvoyer un objet **sale** : réinitialisez-le.

## `sync.Map` & `sync.Cond` (en bref)

- **`sync.Map`** n'est gagnante que dans deux cas : clés **écrites une fois puis lues souvent**, ou
  jeux de clés **disjoints** par goroutine. Sinon, une `map` + `RWMutex` est plus simple et souvent
  plus rapide. 🔁 Internals des maps au [Ch. 32](32-maps-hachage.md).
- **`sync.Cond`** (attendre qu'une condition devienne vraie) est **rarement** le bon outil en Go : un
  **canal** exprime presque toujours la même chose plus clairement. À réserver aux cas où l'on
  réveille un grand nombre d'attentes (`Broadcast`).

---

## 🆕 Go 1.2x

- **1.25** — **`WaitGroup.Go(f)`** (lance + compte + décompte) et l'analyzer **`go vet waitgroup`**
  (détecte `Add` appelé dans la goroutine). 🔁 [Ch. 13](13-tests-outillage.md).
- **1.21** — **`sync.OnceFunc`** / **`sync.OnceValue`** / `OnceValues` : initialisation paresseuse
  sans le boilerplate de `sync.Once`.
- Depuis **1.19**, les **types** atomiques (`atomic.Int64`, `atomic.Pointer[T]`...) remplacent
  avantageusement les fonctions `atomic.AddInt64` & co (typage, pas de souci d'alignement).

## ⚠️ Pièges

- **Copier un type `sync`** (`Mutex`, `WaitGroup`, `Once`...) après usage : récepteur **pointeur**
  obligatoire ; `go vet copylocks` veille.
- **Oublier `Unlock`** : interblocage. Toujours `defer mu.Unlock()` juste après `Lock`.
- **Mutex non réentrant** : appeler `Lock()` depuis une goroutine qui le détient déjà (appel
  récursif, ou méthode publique verrouillante qui en appelle une autre) bloque **pour toujours** — Go
  n'a pas de verrou réentrant. Isolez la logique interne dans des méthodes **non verrouillantes**,
  appelées par les méthodes publiques qui, elles, verrouillent.
- **Ordre de verrouillage incohérent** (verrou A puis B ici, B puis A là) : interblocage **circulaire**
  dès que les deux séquences s'exécutent en même temps (cycle classique « ABBA »). Imposez un **ordre
  total** dès qu'une opération touche plusieurs verrous (ex. trier par un identifiant stable).
- **Montée en grade `RLock` -> `Lock`** sur un `RWMutex` : interblocage (non réentrant).
- **Mélanger atomique et non-atomique** sur la même variable : course. Tout ou rien.
- **`sync.Pool` pour du durable** : le GC le vide. Réservé aux objets temporaires, à réinitialiser.

## ⚡ Performance

Mesuré (go1.26.4, Apple M3, `RunParallel` = sous contention) :

```
   BenchmarkAtomicInc     52.5 ns/op   0 allocs   (atomic.Int64.Add)
   BenchmarkMutexInc     138.4 ns/op   0 allocs   (Mutex autour d'un n++)
   BenchmarkRWMutexRead   63.5 ns/op   0 allocs   (lecteurs parallèles)
   BenchmarkMutexRead    120.4 ns/op   0 allocs   (exclusif même en lecture)
```

- Pour **une valeur**, `atomic` est ~**2,6×** plus rapide qu'un `mutex` (et plus simple).
- Ces chiffres mesurent le **pire cas** : plusieurs goroutines qui se disputent réellement le verrou
  (`RunParallel`). Un `Mutex` **non contesté** (jamais qu'une seule goroutine à la fois en pratique)
  coûte presque aussi peu qu'un `atomic` — `Lock`/`Unlock` se résout alors en une simple opération
  atomique réussie au premier essai, sans passer par le runtime. Le surcoût mesuré ici vient du
  **réveil des goroutines mises en sommeil** quand elles doivent attendre, pas du verrou lui-même.
- Pour de la **lecture pure**, `RWMutex` est ~**1,9×** plus rapide qu'un `Mutex` — utile seulement si
  les lectures dominent réellement.
- **Faux partage** (_false sharing_) : deux atomics chauds sur la **même ligne de cache** (64 o) se
  gênent comme s'ils partageaient la donnée. Pour un état très contendu, séparez-les (padding). 🔁
  [Ch. 26](26-allocation-escape.md).
- Le moins cher reste de **ne pas partager** : donnez à chaque goroutine sa donnée, agrégez à la fin.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch21-synchronisation
go test -race ./ch21-synchronisation/...
go test -bench=. -benchmem -cpu=8 ./ch21-synchronisation/...
```

À essayer :

1. Retirez le `Lock` de `SafeCounter.Inc` et lancez `go test -race` : observez la course signalée.
2. Donnez à `SafeCounter` un récepteur **valeur** et lancez `go vet` : lisez l'alerte `copylocks`.
3. Remplacez `RWMutex` par `Mutex` dans `Registry` et comparez les benchmarks de lecture.
4. Faites paniquer `expensiveInit` (`panic("boom")` avant le `return`) puis appelez `config()`
   plusieurs fois : chaque appel **repanique** avec `"boom"`, sans jamais ré-exécuter `expensiveInit`
   avec succès — la panique d'un `OnceValue` n'est jamais avalée.

---

## 📌 À retenir

- **Canal** pour transférer/signaler ; **mutex** pour protéger une section ; **atomic** pour une seule
  valeur. Choisir selon l'intention.
- `defer mu.Unlock()` **systématiquement** ; ne **copiez jamais** un type `sync` (récepteur pointeur).
- **`WaitGroup.Go`** (1.25) remplace `Add`/`go`/`Done` et évite le piège du `Add` mal placé.
- `OnceValue` (1.21) pour l'init unique ; `atomic.Pointer[T]` pour publier un état rechargeable sans
  verrou.
- `atomic` > `mutex` > `RWMutex` en lecture > `Mutex` en lecture, mais **ne pas partager** bat tout.

## 🔁 Pour aller plus loin

- [Ch. 22 — `context`](22-context.md) : annulation et délais propagés.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : _data races_ et `go test -race`.
- [Ch. 25 — Modèle mémoire](25-modele-memoire.md) : les garanties _happens-before_ de `sync`/`atomic`.
- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : faux partage, lignes de cache, `sync.Pool`.
