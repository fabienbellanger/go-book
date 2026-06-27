# Ch. 26 — Allocation mémoire & escape analysis

> **Objectif** — Comprendre **où vivent les données** : la **pile** (gratuite, par goroutine) ou le
> **tas** (allocation + travail pour le GC). Savoir ce qu'est l'**escape analysis**, lire ses décisions
> avec `-gcflags=-m`, **mesurer** les allocations (`allocs/op`), connaître l'**allocateur**
> (`mcache`/`mcentral`/`mheap`, size classes, tiny allocator) et **réduire** les allocations.
>
> **Prérequis** — [Ch. 5](05-fonctions.md), [Ch. 6](06-arrays-slices.md), [Ch. 19](19-goroutines.md)

---

## Introduction

Une variable Go vit à l'un de deux endroits. Sur la **pile** : créée et détruite avec l'appel de
fonction, **coût nul**, aucune intervention du GC. Sur le **tas** : elle survit à la fonction, mais
chaque allocation coûte du temps **et** alimente le travail du [garbage collector](27-garbage-collector.md).
La bonne nouvelle : **vous ne choisissez pas** — le compilateur décide, via l'**escape analysis**. La
meilleure nouvelle : vous pouvez **lire** et **influencer** ses décisions. Code dans
[`code/ch26-allocation-escape/`](../code/ch26-allocation-escape/).

---

## La pile d'une goroutine

Chaque goroutine démarre avec une **petite pile** (~2 Kio, [Ch. 19](19-goroutines.md)) qui **croît**
automatiquement : quand elle déborde, le runtime alloue une pile plus grande, **copie** le contenu et
ajuste les pointeurs. C'est transparent. Tout ce qui ne « s'échappe » pas de la fonction y est rangé,
puis **libéré gratuitement** au retour — sans GC.

```
  pile d'une goroutine          le tas (partagé, géré par le GC)
  +------------------+          +-----------------------------+
  | frame de main    |          |  objets qui SURVIVENT aux   |
  +------------------+          |  fonctions (échappés)       |
  | frame de f()     |          |  -> alloc + travail du GC   |
  |   buf [16]int    | <- pile  +-----------------------------+
  +------------------+
        ^ grossit/rétrécit avec les appels (gratuit)
```

## L'escape analysis décide

À la compilation, l'escape analysis détermine si la durée de vie d'une valeur **dépasse** la fonction.
Si oui, la valeur « **s'échappe** » et part sur le tas. Les déclencheurs classiques :

- **renvoyer un pointeur** vers une variable locale ;
- **stocker un pointeur** dans une structure qui survit (champ, slice, map, canal) ;
- passer une valeur à une **interface** consommée ailleurs (boxing) ;
- une taille **inconnue à la compilation** (`make([]T, n)` renvoyé).

```go
// code/ch26-allocation-escape/alloc.go
func sumLocalArray(n int) int { // [16]int reste local -> PILE, 0 alloc
	var buf [16]int
	/* ... */
	return s
}

func NewPoint(x, y int) *Point { // &p renvoyé -> p s'échappe -> TAS, 1 alloc
	p := Point{x, y}
	return &p
}
```

## Lire les décisions : `-gcflags=-m`

Le compilateur **explique** chaque choix avec `-m` :

```
$ go build -gcflags=-m ./ch26-allocation-escape
alloc.go:31:2:  moved to heap: p                      <- NewPoint : échappe
alloc.go:41:11: make([]int, 8) does not escape        <- sumSmallSlice : PILE
alloc.go:56:11: make([]int, n) escapes to heap        <- LeakSlice : renvoyé
```

`moved to heap` / `escapes to heap` = tas ; `does not escape` = pile. C'est l'outil **n°1** pour
chasser les allocations cachées. 🔁 [Ch. 39](39-compilation-inlining-pgo.md) pour les autres `-m`.

## Mesurer : `allocs/op`

`testing.AllocsPerRun` transforme l'analyse en **test** : il échoue si une allocation réapparaît.

```go
// code/ch26-allocation-escape/alloc_test.go
func TestSumLocalArrayNoAlloc(t *testing.T) {
	if got := testing.AllocsPerRun(100, func() { _ = sumLocalArray(3) }); got != 0 {
		t.Errorf("sumLocalArray = %.0f alloc/op ; attendu 0 (pile)", got)
	}
}
```

Résultats mesurés (go1.26.4) :

| Fonction        | alloc/op | Où ?                                         |
| --------------- | -------- | -------------------------------------------- |
| `sumLocalArray` | **0**    | tableau local sur la pile                    |
| `sumSmallSlice` | **0**    | backing de slice **sur la pile** (1.25/1.26) |
| `NewPoint`      | **1**    | pointeur renvoyé → tas                       |
| `LeakSlice`     | **1**    | slice renvoyé → tas                          |

## L'allocateur du tas

Quand une valeur s'échappe, le runtime ne fait **pas** un `malloc` système à chaque fois. Il gère un
**cache à trois étages**, optimisé pour le sans-contention :

```
  mcache   (par P, SANS verrou)     <- 99 % des allocs servies ici, ultra-rapide
    | à court de place ?
    v
  mcentral (par size class, verrou) <- recharge le mcache en spans
    | à court de spans ?
    v
  mheap    (global)  ---> OS (mmap) <- gros objets (>32 Kio) + approvisionnement

  Le tas est découpé en SPANS, chacun dédié à une « size class » :
     8, 16, 24, 32, 48, 64, ... jusqu'à 32 Kio  (~68 classes)
  Un objet de 17 o va dans la classe 24 o  (perte = fragmentation interne).
  Tiny allocator : objets < 16 o SANS pointeur regroupés dans un même bloc de 16 o.
```

Chaque P ([Ch. 28](28-ordonnanceur-gmp.md)) possède son **`mcache`** : allouer un petit objet est donc
**sans verrou** dans le cas courant. C'est ce qui rend l'allocation Go rapide malgré le GC.

## Réduire les allocations

Le levier le plus rentable : **préallouer** quand la taille est connue. `append` sur un slice de
capacité 0 **réalloue** plusieurs fois (le backing **double**, [Ch. 6](06-arrays-slices.md)) :

```go
// code/ch26-allocation-escape/alloc.go
func concatPrealloc(n int) []int {
	out := make([]int, 0, n) // capacité réservée d'emblée : 1 seule allocation
	for i := range n {
		out = append(out, i)
	}
	return out
}
```

Mesuré pour `n=1000` (`-benchmem`) :

| Variante           | ns/op    | B/op     | allocs/op |
| ------------------ | -------- | -------- | --------- |
| `concatNoPrealloc` | **2690** | 25152    | **9**     |
| `concatPrealloc`   | **1116** | **8192** | **1**     |

Une `cap` réservée : **9 → 1** allocation, **3×** moins de mémoire, **2,4×** plus rapide — sans changer
le résultat. Pour les objets **réutilisables** et de courte vie, `sync.Pool` ([Ch. 21](21-synchronisation.md))
recycle plutôt que réallouer.

---

## 🆕 Go 1.2x

- **1.25 / 1.26** — le backing d'un **slice** créé localement, de **taille bornée** et qui **ne
  s'échappe pas**, est de plus en plus souvent alloué **sur la pile** (vérifié : `sumSmallSlice` =
  **0 alloc/op**, `make([]int, 8) does not escape`). Du code idiomatique alloue donc **moins**, sans
  effort. 🔁 [Ch. 39](39-compilation-inlining-pgo.md).
- **1.26** — l'**adresse de base du tas** est randomisée (durcissement, [Ch. 24](24-runtime-bootstrap.md)) ;
  ne supposez **jamais** la stabilité des adresses entre exécutions.

## ⚠️ Pièges

- **Renvoyer `&local`** par réflexe — souvent ce qui fait fuir vers le tas. Si l'appelant n'a pas besoin
  d'un pointeur, renvoyez la **valeur** (les petits structs se copient à bas coût).
- **Interfaces sur le chemin chaud** — convertir une valeur en `any`/interface peut **boxer** (allouer).
  `-gcflags=-m` le révèle.
- **`append` sans `cap`** dans une boucle — réallocations répétées. Préallouez dès que la taille est
  connue ou estimable.
- **Micro-optimiser à l'aveugle** — `-m` et `-benchmem` **d'abord** ; n'allez pas contre l'escape
  analysis sans preuve qu'il y a un gain.

## ⚡ Performance

- Une alloc tas ne coûte pas que l'alloc : elle **augmente le travail du GC** ([Ch. 27](27-garbage-collector.md)).
  Réduire `allocs/op` réduit **deux** coûts à la fois.
- L'allocation sur le `mcache` est **sans verrou** : le vrai ennemi n'est pas le coût unitaire mais le
  **nombre** d'objets (pression GC) et leur **taille** (size class).
- 🔁 [Ch. 30](30-slices-profondeur.md) (stratégie de croissance d'`append`) et
  [Ch. 36](36-tests-benchmarks-fuzzing.md) (`benchstat`) pour mesurer rigoureusement.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch26-allocation-escape
go build -gcflags=-m ./ch26-allocation-escape 2>&1 | grep -E "escape|heap"
go test -bench=. -benchmem -run=^$ ./ch26-allocation-escape/...
```

À essayer :

1. Faites renvoyer une **valeur** au lieu d'un pointeur dans `NewPoint` : `-m` la garde-t-elle sur la pile ?
2. Passez `sumLocalArray` dans `fmt.Println` (interface) : observez l'allocation apparaître sous `-m`.
3. Estimez la `cap` initiale idéale de `concatNoPrealloc` et mesurez le gain avec `-benchmem`.

---

## 📌 À retenir

- **Pile** = gratuit, par goroutine, libéré au retour ; **tas** = alloc + pression GC. L'**escape
  analysis** (compilation) décide, pas vous.
- Une valeur **s'échappe** si sa durée de vie dépasse la fonction : pointeur renvoyé/stocké, interface,
  taille dynamique. **`-gcflags=-m`** explique chaque choix.
- **Mesurez** avec `allocs/op` (`testing.AllocsPerRun`, `-benchmem`) — et faites-en des **tests** de non-régression.
- L'allocateur a **trois étages** : `mcache` (par P, sans verrou) → `mcentral` → `mheap` → OS ; tas en
  **spans** par **size class** ; **tiny allocator** pour les objets < 16 o sans pointeur.
- **Préallouez** (`make([]T, 0, n)`), réutilisez (`sync.Pool`) : moins d'allocations = plus rapide **et**
  moins de GC.

## 🔁 Pour aller plus loin

- [Ch. 27 — Garbage collector](27-garbage-collector.md) : ce que coûtent les objets du tas.
- [Ch. 28 — L'ordonnanceur](28-ordonnanceur-gmp.md) : le P et son `mcache`.
- [Ch. 30 — Slices en profondeur](30-slices-profondeur.md) : croissance d'`append`, `slices.Clip`.
- [Ch. 39 — Compilation & inlining](39-compilation-inlining-pgo.md) : escape analysis et inlining ensemble.
- Doc : `go doc testing.AllocsPerRun` ; `go build -gcflags=-m` sur votre propre code.
