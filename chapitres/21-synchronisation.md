# Ch. 21 — Primitives de synchronisation

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

> ⚠️ `RWMutex` n'est **pas réentrant** : tenter de prendre le `Lock` en tenant déjà un `RLock`
> (« montée en grade ») **interbloque**. Il n'est gagnant que si les lectures sont **nombreuses et
> longues** ; sous écritures fréquentes, un `Mutex` simple est souvent plus rapide.

## `sync.WaitGroup` & `WaitGroup.Go` (1.25)

Un `WaitGroup` attend la fin d'un groupe de goroutines (déjà croisé au [Ch. 19](19-goroutines.md)).
Le schéma historique — `Add(1)` / `go` / `defer Done()` — est **verbeux et fragile** (un `Add` mal
placé casse tout). Go 1.25 ajoute **`WaitGroup.Go`** qui fait les trois d'un coup :

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
