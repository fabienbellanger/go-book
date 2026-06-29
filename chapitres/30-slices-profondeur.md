# 30 — Slices & arrays en profondeur

> **Objectif** — Ouvrir le **header de slice** (ptr/len/cap), comprendre la **stratégie de croissance**
> d'`append` (amortissement, facteur, arrondi aux size classes), maîtriser l'**expression à 3 indices**
> pour borner la capacité, et désamorcer les deux pièges majeurs — **aliasing** et **rétention
> mémoire** — avec des patterns **zéro allocation** et le package `slices`.
>
> **Prérequis** — [Ch. 6](06-arrays-slices.md), [Ch. 26](26-allocation-escape.md)

---

## Introduction

Le [Ch. 6](06-arrays-slices.md) a montré **comment utiliser** les slices ; ce chapitre montre **comment
ils marchent**. Un slice n'est pas un tableau : c'est une **vue** de trois mots machine sur un tableau
sous-jacent (le _backing array_). Comprendre ces trois mots, c'est expliquer d'un coup la réallocation
d'`append`, les bugs d'aliasing et les fuites mémoire silencieuses. Code dans
[`code/ch30-slices-profondeur/`](../code/ch30-slices-profondeur/).

---

## Le header de slice : 3 mots

Un `[]T` est une petite structure de **24 octets** sur une machine 64 bits (3 mots de 8 octets) :
un **pointeur** vers le backing, une **longueur** (`len`) et une **capacité** (`cap`). Copier un
slice copie **ce header**, pas les données — d'où le partage du backing.

```
  s := arr[1:4]

  s  (slice header = 3 mots machine = 24 octets)
  +---------+---------+---------+
  |  ptr    |  len=3  |  cap=4  |
  +----+----+---------+---------+
       |
       v
  arr [0] [1] [2] [3] [4]
           ^---- len ----^
           ^------- cap -------^
```

`len` borne la lecture/écriture ; `cap` borne la croissance **sans réallocation**. Tant que
`len < cap`, `append` écrit **dans le backing existant**. Quand `len == cap`, il en alloue un
**nouveau**, plus grand.

## La stratégie de croissance d'`append`

Quand `append` doit agrandir, il ne prend pas pile la taille requise : il **sur-alloue** pour amortir
les futurs `append`. La règle (runtime `growslice`, Go 1.18+) :

- **`cap < 256`** → on **double** ;
- **`cap >= 256`** → croissance **plus douce**, `newcap += (newcap + 3*256) / 4` (≈ **1,25×** lissé),
- puis on **arrondit à une size class** de l'allocateur ([Ch. 26](26-allocation-escape.md)).

```go
// code/ch30-slices-profondeur/main.go
fmt.Printf("cap au fil des append : %v\n", CapGrowth(2000))
```

```
$ go run ./ch30-slices-profondeur
cap au fil des append (nil -> 2000 ints) : [4 8 16 32 64 128 256 512 848 1280 1792 2560]
```

On lit le doublement jusqu'à 256, puis le ralentissement (512 → 848 → 1280…, ratio ≈ 1,3 puis moins).
L'amortissement rend une suite de `n` `append` **O(n)** au total, pas O(n²). 💡 Le premier `append` sur
un slice **nil** saute directement à `cap=4` (pour `[]int`) : l'`append` inliné arrondit à une petite
size class. Ne vous fiez pas aux premières valeurs exactes — fiez-vous à la **stratégie**.

## Borner la capacité : l'expression à 3 indices

`s[low:high]` donne une `cap` qui va **jusqu'au bout** du backing. `s[low:high:max]` (3 indices, _full
slice expression_) la **borne à `max-low`** :

```go
// code/ch30-slices-profondeur/slicegrow.go
func SubSliceCap(s []int, low, high, max int) int {
	return cap(s[low:high:max]) // vaut max-low, pas len(backing)-low
}
```

```
arr := [10]int{...}
arr[2:4]    -> len=2  cap=8   (jusqu'au bout)
arr[2:4:6]  -> len=2  cap=4   (borné par le 3e indice)
```

C'est l'outil pour **isoler** un sous-slice : avec `cap == len`, le prochain `append` est **forcé de
réallouer** au lieu d'écraser le voisin.

## Piège n°1 : l'aliasing par `append`

Un sous-slice qui a **encore de la capacité** partage le backing du parent. Un `append` y **écrit
par-dessus** les données du parent :

```go
// code/ch30-slices-profondeur/slicegrow.go
func AppendAliasing() (parent, modified []int) {
	parent = []int{1, 2, 3, 4}
	head := parent[:2]          // len=2, cap=4 : partage le backing
	modified = append(head, 99) // cap suffit -> écrit dans parent[2] !
	return                      // parent == [1 2 99 4]
}
```

La parade : **borner la cap** (`parent[:2:2]`) — l'`append` réalloue, le parent reste intact
(`SafeAppend`). Mesuré : `AppendAliasing` donne `parent=[1 2 99 4]` ; `SafeAppend` donne `[1 2 3 4]`.

## Piège n°2 : la rétention mémoire

Un petit sous-slice **retient tout le backing** : tant qu'il vit, le grand tableau n'est **pas
collecté** ([Ch. 27](27-garbage-collector.md)). Garder 3 éléments d'un tableau d'un million les
retient **tous**.

```go
// code/ch30-slices-profondeur/slicegrow.go
func TrimRetention(big []int, n int) []int {
	return slices.Clone(big[:n]) // copie -> backing de taille n ; l'ancien devient collectable
}
```

`slices.Clone` (ou `slices.Clip`, qui ramène `cap` à `len`) **détache** la vue de son grand backing.

## Pattern zéro allocation : filtrer en place

Comme `append` réutilise le backing tant qu'il y a de la `cap`, on peut **filtrer sans allouer** en
repartant de `src[:0]` :

```go
// code/ch30-slices-profondeur/slicegrow.go
func FilterInPlace(src []int, keep func(int) bool) []int {
	out := src[:0] // même backing, len remise à 0
	for _, v := range src {
		if keep(v) {
			out = append(out, v) // réécrit par-dessus src, jamais de réallocation
		}
	}
	return out
}
```

Mesuré (`-benchmem`, entrée de 1000 entiers) :

| Variante         | ns/op    | B/op  | allocs/op |
| ---------------- | -------- | ----- | --------- |
| `FilterInPlace`  | **2130** | **0** | **0**     |
| `filterNewSlice` | 2907     | 8184  | **10**    |

Réutiliser le backing : **0 allocation** contre 10. ⚠️ En contrepartie, `FilterInPlace` **détruit**
le contenu d'origine de `src` — à n'employer que si l'appelant n'en a plus besoin.

---

## 🆕 Go 1.2x

- **1.21** — le package **`slices`** entre dans la bibliothèque standard : `Clone`, `Clip`, `Grow`,
  `Delete`, `Insert`, `Contains`, `Index`, `Sort`, `Equal`, `Compact`… Beaucoup de manipulations
  manuelles (et leurs bugs d'aliasing) disparaissent. 🔁 [Ch. 11](11-genericite.md).
- **1.22+** — l'**escape analysis** range de plus en plus de backings de taille bornée **sur la pile**
  ([Ch. 26](26-allocation-escape.md)) : un `make([]T, k)` local qui ne s'échappe pas n'alloue plus.

## ⚠️ Pièges

- **`append` qui écrase le parent** — un sous-slice avec `cap > len` partage le backing. Bornez la cap
  (`s[a:b:b]`) ou clonez avant de modifier.
- **Rétention d'un grand backing** — un petit slice d'un grand tableau le retient en entier.
  `slices.Clone`/`Clip` pour détacher.
- **Supposer la valeur exacte des premières `cap`** — le premier `append` arrondit à une size class
  (cap 4 pour `[]int`). Comptez sur la **stratégie**, pas sur `4` ou `8`.
- **Réutiliser le résultat de `FilterInPlace`** en pensant que `src` est intact — il est **écrasé**.

## ⚡ Performance

- **Préallouez** dès que la taille est connue : `make([]T, 0, n)` supprime toutes les réallocations
  intermédiaires ([Ch. 26](26-allocation-escape.md) : 9 → 1 allocation).
- **`slices.Grow(s, n)`** garantit `n` places libres en **une** réallocation avant une rafale d'`append`.
- **Réutilisez un buffer** entre les appels (`buf = buf[:0]`) sur le chemin chaud : zéro allocation.
- L'**amortissement** rend `append` O(1) amorti — mais chaque réallocation **copie** ; sur de gros
  volumes, préallouer évite et les copies et la pression GC.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch30-slices-profondeur
go test ./ch30-slices-profondeur/...
go test -bench=. -benchmem -run=^$ ./ch30-slices-profondeur/...
```

À essayer :

1. Affichez `CapGrowth` pour un `[]byte` : la suite des `cap` diffère (size classes d'octets).
2. Remplacez `parent[:2]` par `parent[:2:2]` dans `AppendAliasing` et observez le parent redevenir intact.
3. Mesurez `slices.Grow(s, 1000)` + 1000 `append` vs 1000 `append` sur cap 0 (`-benchmem`).

---

## 📌 À retenir

- Un slice = **header de 3 mots** (ptr/len/cap, 24 o en 64 bits) **vue** sur un backing partagé ; copier
  un slice copie le header, pas les données.
- `append` **sur-alloue** : **double** sous 256, puis **~1,25×**, arrondi à une **size class**.
  Amortissement O(n) ; chaque réallocation **copie**.
- L'**expression à 3 indices** `s[a:b:c]` **borne la cap** — l'outil pour isoler un sous-slice et forcer
  la réallocation.
- Deux pièges : **aliasing** (`append` écrase le parent) et **rétention** (un petit slice retient un grand
  backing). Parade : borner la cap, `slices.Clone`/`Clip`.
- **Préallouer** et **réutiliser un buffer** (`buf[:0]`) éliminent les allocations sur le chemin chaud.

## 🔁 Pour aller plus loin

- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : size classes, pile vs tas, préallocation.
- [Ch. 27 — Garbage collector](27-garbage-collector.md) : pourquoi la rétention coûte cher.
- [Ch. 31 — Strings en profondeur](31-strings-profondeur.md) : le même genre de header, mais immuable.
- [Ch. 36 — Benchmarks & fuzzing](36-tests-benchmarks-fuzzing.md) : mesurer `allocs/op` rigoureusement.
- Doc : `go doc slices` ; `go doc builtin.append`.
