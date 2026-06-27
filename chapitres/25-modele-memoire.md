# Ch. 25 — Le modèle mémoire de Go

> **Objectif** — Savoir **quand** une écriture faite par une goroutine est **garantie visible** par une
> autre. Comprendre la relation **happens-before**, les arêtes qu'établissent `go`, les **canaux**,
> `sync` et `sync/atomic`, pourquoi **aucune** garantie n'existe sans synchronisation, et pourquoi
> **`-race`** est indispensable.
>
> **Prérequis** — [Ch. 19](19-goroutines.md), [Ch. 21](21-synchronisation.md), [Ch. 23](23-patterns-concurrence.md) (data race)

---

## Introduction

Le [Ch. 23](23-patterns-concurrence.md) a défini la **data race** ; ce chapitre répond à la question
de fond : **quand** une goroutine est-elle **sûre** de voir ce qu'une autre a écrit ? L'intuition
« elle a écrit avant, donc je le vois » est **fausse**. Le compilateur réordonne, le processeur met en
cache, chaque cœur a sa vision de la mémoire. La **seule** garantie est celle qu'offre une
**synchronisation explicite**. Le **modèle mémoire** de Go formalise précisément ces garanties. Code
dans [`code/ch25-modele-memoire/`](../code/ch25-modele-memoire/).

> 📌 Maxime de la spec officielle : « Si vous devez lire le modèle mémoire pour comprendre le
> comportement de votre programme, vous êtes trop malin. Ne soyez pas malin. » Le bon réflexe n'est pas
> de raisonner sur les réordonnancements, mais de **toujours synchroniser** les accès partagés.

---

## Sans synchronisation, aucune garantie

```go
// COURSE : deux goroutines, aucune synchronisation.
var ready bool
var data int

go func() { data = 42; ready = true }() // écrit data PUIS ready
go func() {
	if ready {       // peut voir ready=true...
		print(data)  // ...tout en lisant data=0 (!) : ordre non garanti
	}
}()
```

Sans arête de synchronisation, **rien** ne garantit que la goroutine de droite voie `data=42` même si
elle voit `ready=true`. Le résultat n'est pas « une vieille valeur » : il est **indéfini**. Pire, pour
une valeur plus large qu'un mot machine, un lecteur peut voir un état **déchiré** (moitié ancienne,
moitié nouvelle).

## `happens-before` : la seule garantie

Le modèle mémoire définit un ordre partiel, **happens-before**. La règle fondatrice :

> Si l'écriture _W_ d'une variable **happens-before** la lecture _R_, et qu'aucune autre écriture
> n'intervient entre les deux, alors _R_ **voit** _W_. Sans cette relation, _R_ peut voir **n'importe
> quelle** écriture (ou un état déchiré).

Dans **une même goroutine**, happens-before = l'ordre du programme. **Entre goroutines**, il faut une
**arête** explicite, créée par l'un de ces mécanismes :

| Mécanisme          | Arête happens-before garantie                                                    |
| ------------------ | -------------------------------------------------------------------------------- |
| **`go f()`**       | l'instruction `go` _happens-before_ le début de `f`                              |
| **Canal (envoi)**  | un **envoi** _happens-before_ la **réception** correspondante qui se termine     |
| **Canal (close)**  | le **`close`** _happens-before_ une réception qui renvoie zéro (canal fermé)     |
| **Canal non buf.** | une **réception** _happens-before_ la fin de l'**envoi** correspondant           |
| **`sync.Mutex`**   | le `n`-ième `Unlock` _happens-before_ le `m`-ième `Lock` (pour `n < m`)          |
| **`sync.Once`**    | le retour de `f` dans `Do(f)` _happens-before_ le retour de **tout** `Do`        |
| **`WaitGroup`**    | les `Done` _happens-before_ le retour de `Wait`                                  |
| **`sync/atomic`**  | un `Store` _happens-before_ le `Load` qui l'**observe** (cohérence séquentielle) |

```
  goroutine A                          goroutine B
  -----------                          -----------
  c.Addr = "..."   (1)
  c.Port = 8080    (2)
  c.Ready = true   (3)
  ch <- c          (4) ----+
                           |  l'envoi happens-before la reception
                           +---> v := <-ch   (5)
                                 use v.Addr   (6)   voit (1)(2)(3) EN BLOC
  ordre : (1)(2)(3) -> (4) -> (5) -> (6)
```

L'arête du canal (4)→(5) « transporte » toutes les écritures qui la précèdent : B voit forcément la
config **entièrement** construite.

## Publier une valeur sans risque

Trois façons **correctes** de partager une valeur construite par une goroutine — toutes vertes sous
`-race` :

```go
// code/ch25-modele-memoire/memmodel.go
// 1) Par canal : la réception voit tout ce qui précède l'envoi.
func PublishViaChannel() *Config {
	ch := make(chan *Config)
	go func() { ch <- buildConfig() }()
	return <-ch
}

// 2) Initialisation paresseuse : sync.Once garantit la visibilité du constructeur.
var once sync.Once
var cfg *Config

func GetConfig() *Config {
	once.Do(func() { cfg = buildConfig() })
	return cfg
}

// 3) Sans verrou : atomic.Pointer fournit la barrière (Store happens-before Load).
var current atomic.Pointer[Config]

func SwapConfig(c *Config) { current.Store(c) }
func LoadConfig() *Config  { return current.Load() }
```

## Le piège : double-checked locking

Tenté d'« optimiser » en lisant un drapeau **sans verrou** avant de verrouiller ? C'est le
**double-checked locking**, et il est **buggé** en Go :

```go
// BUGGÉ — ne JAMAIS écrire ceci.
func GetBad() *Config {
	if cfg == nil { // lecture NON synchronisée : course avec l'écriture ci-dessous
		mu.Lock()
		if cfg == nil {
			cfg = buildConfig() // l'écriture peut être vue « à moitié » par la 1re ligne
		}
		mu.Unlock()
	}
	return cfg // risque : *Config partiellement initialisé
}
```

La première lecture `cfg == nil` n'a **aucune** arête happens-before avec l'écriture `cfg = ...`. Un
appelant peut récupérer un pointeur **non nil mais pointant sur un `Config` à demi construit**. La
solution n'est pas d'ajouter un `atomic` à la main : c'est **`sync.Once`**, conçu exactement pour ça.

## Pourquoi `-race` est indispensable

Le modèle mémoire dit qu'un programme **avec** data race a un comportement **indéfini** — donc on ne
veut **aucune** race, pas « des races bénignes ». Mais les races sont **invisibles** à l'œil et
souvent à l'exécution normale. Le **détecteur** (`-race`, [Ch. 23](23-patterns-concurrence.md))
instrumente les accès et signale toute paire non ordonnée par un happens-before :

```bash
go test -race ./...   # en CI, systématiquement
```

> 📌 Une « race bénigne » n'existe pas dans le modèle Go. Si `-race` la signale, **corrigez-la** par une
> vraie synchronisation — n'essayez pas de la justifier.

---

## 🆕 Go 1.2x

- Depuis **Go 1.19**, le modèle mémoire est **aligné sur C/C++** : les opérations `sync/atomic` sont
  **séquentiellement cohérentes** et formellement spécifiées (types `atomic.Int64`, `atomic.Pointer[T]`…
  du [Ch. 21](21-synchronisation.md)). Avant, on devait s'appuyer sur la doc informelle du package.
- Le modèle est **stable** : il fait partie de la **Go 1 promise**. Ce que vous apprenez ici ne change pas.
- 🔁 Le détecteur de courses (`-race`) et `testing/synctest` (1.25) du [Ch. 23](23-patterns-concurrence.md)
  sont les outils pratiques qui complètent ce modèle théorique.

## ⚠️ Pièges

- **« Ça marche sur ma machine »** — une race peut passer 1000 fois puis casser sous charge, sur un
  autre CPU, ou après une optimisation du compilateur. L'absence de bug observé ≠ correction.
- **Double-checked locking** maison — toujours buggé ; utilisez `sync.Once` / `sync.OnceValue`.
- **Croire que `int`/`bool` est « atomique »** — même une écriture de mot machine n'offre **aucune**
  garantie de **visibilité** sans synchronisation. Utilisez `sync/atomic` ou un verrou.
- **Confondre atomicité et happens-before** — un accès atomique isolé ne protège pas une **séquence**
  d'opérations ; pour un invariant multi-champs, il faut un verrou ou une publication par pointeur.

## ⚡ Performance

- Une arête happens-before a un **coût** (barrière mémoire) : c'est le prix de la correction. Ne
  l'évitez pas, **réduisez** le partage (chaque goroutine travaille sur ses propres données).
- `atomic` < `Mutex` < `RWMutex` en lecture pure côté coût, mais seul le **profil** tranche
  ([Ch. 21](21-synchronisation.md) pour les chiffres : atomic 52 ns / mutex 138 ns).
- 🔁 [Ch. 27](27-garbage-collector.md) : les **write barriers** du GC sont un autre usage des barrières
  mémoire, à un niveau plus bas.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch25-modele-memoire
go test -race ./ch25-modele-memoire/...   # vert : tout partage est synchronisé
```

À essayer :

1. Écrivez la version `ready/data` sans synchronisation, lancez-la en boucle sous `-race` : observez le
   `WARNING: DATA RACE`.
2. « Réparez-la » avec un canal, puis avec `atomic.Bool` + `atomic.Int64` : `-race` redevient vert.
3. Implémentez `GetBad` (double-checked locking) et confirmez que `-race` le signale.

---

## 📌 À retenir

- Sans **happens-before**, une lecture peut voir **n'importe quelle** écriture (ou un état **déchiré**) :
  comportement **indéfini**, pas « valeur périmée ».
- Les arêtes happens-before viennent de **`go`**, des **canaux**, de **`sync`** (Mutex, Once, WaitGroup)
  et de **`sync/atomic`** — **jamais** du simple passage du temps.
- Pour **publier** une valeur : canal, `sync.Once`, ou `atomic.Pointer`. Pour l'**init paresseuse**,
  `sync.Once` (jamais le double-checked locking maison).
- Une **data race** = bug, point. Pas de « race bénigne ».
- **`-race`** est le seul moyen pratique de les traquer : activez-le en **CI**.

## 🔁 Pour aller plus loin

- [Ch. 21 — Synchronisation](21-synchronisation.md) : les primitives qui créent ces arêtes, et leur coût.
- [Ch. 23 — Patterns & data races](23-patterns-concurrence.md) : le détecteur `-race` en pratique.
- [Ch. 27 — Garbage collector](27-garbage-collector.md) : write barriers et cohérence mémoire bas niveau.
- Doc officielle : **The Go Memory Model** (`go.dev/ref/mem`) — la référence formelle.
