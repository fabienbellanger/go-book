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
sous-jacent (le _backing array_). Comprendre ces trois mots — et ce qui, structurellement, distingue un
array d'un slice — explique d'un coup la réallocation d'`append`, les bugs d'aliasing et les fuites
mémoire silencieuses. Code dans [`code/ch30-slices-profondeur/`](../code/ch30-slices-profondeur/).

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

### Le vrai type : `runtime.slice`, et pourquoi un array n'en a pas besoin

Le schéma ci-dessus n'est pas qu'une métaphore : à la compilation, `[]T` devient le type interne
`runtime.slice` (vérifié dans les sources Go 1.26) :

```go
type slice struct {
	array unsafe.Pointer // le champ s'appelle bien "array", pas "ptr"
	len   int
	cap   int
}
```

Un `[N]T` (array), lui, **n'a pas** ce header : la variable **est** directement les `N` valeurs de
`T`, mises bout à bout en mémoire — il n'y a pas de champ séparé à lire pour la localiser. Deux
conséquences structurelles en découlent :

- **Adressage** — pour `arr[i]`, le compilateur connaît l'adresse de base de `arr` dès la
  compilation (un offset fixe sur la pile, ou dans le receveur) : il calcule directement
  `base + i*sizeof(T)`. Pour `s[i]`, cette base est une **valeur d'exécution** : il faut d'abord lire
  le champ `array` du header — un mot de plus à transporter (souvent déjà en registre pour un slice
  local, donc pas systématiquement un accès mémoire en plus, mais un degré d'indirection
  conceptuelle que l'array n'a pas).
- **Durée de vie** — un array local de taille connue qui ne s'échappe pas tient **entièrement** sur
  la pile, sans aucune indirection vers le tas (même logique que le backing d'un petit slice,
  [Ch. 26](26-allocation-escape.md)). Un array trop volumineux ou renvoyé par pointeur s'échappe
  comme n'importe quelle autre valeur.

🔁 Construire un slice depuis un pointeur brut (ou en extraire le pointeur) passe par
`unsafe.Slice`/`unsafe.SliceData` ([Ch. 35](35-unsafe-cgo.md)) — `reflect.SliceHeader`, l'ancien
moyen d'y accéder, est aujourd'hui marqué _deprecated_ par la doc standard au profit de ces deux
fonctions, justement parce que la mise en page de `runtime.slice` n'est garantie par aucune
spécification, seulement par l'implémentation actuelle.

## La stratégie de croissance d'`append`

Quand `append` doit agrandir, il ne prend pas pile la taille requise : il **sur-alloue** pour amortir
les futurs `append`. La règle, implémentée par `nextslicecap` (appelée depuis `growslice`, runtime
Go 1.18+) :

- **`cap < 256`** → on **double** ;
- **`cap >= 256`** → croissance **plus douce** : `newcap += (newcap + 3*256) / 4`, répété jusqu'à
  couvrir la taille demandée — le facteur ne tend vers **1,25×** que de façon **asymptotique**
  (voir le calcul ci-dessous) ;
- ce résultat, exprimé en **éléments**, est converti en octets puis **arrondi à une size class** de
  l'allocateur ([Ch. 26](26-allocation-escape.md)) — c'est `growslice`, pas `nextslicecap`, qui
  applique cet arrondi final, ce qui peut faire remonter la `cap` réelle au-dessus de la formule.

```go
// code/ch30-slices-profondeur/main.go
fmt.Printf("cap au fil des append : %v\n", CapGrowth(2000))
```

```
$ go run ./ch30-slices-profondeur
cap au fil des append (nil -> 2000 ints) : [4 8 16 32 64 128 256 512 848 1280 1792 2560]
```

On lit le doublement jusqu'à 256, puis le ralentissement — vérifions la formule à la main. À
`cap=256`, la croissance « douce » est déjà active, mais elle redonne _exactement_ 512
(`256 + (256 + 768) / 4 = 512`) : un doublement par coïncidence arithmétique, pas par règle. L'étape
suivante démasque la vraie pente : `512 + (512 + 768) / 4 = 832` éléments demandés, soit
`832 * 8 = 6656` octets pour `[]int` — mais la `cap` observée est **848**, pas 832. L'écart, c'est
l'arrondi à la size class : `growslice` ne peut allouer que des tailles prédéfinies, et la plus
petite qui couvre 6656 o vaut `848 * 8 = 6784` o. Le ratio `848/512 ≈ 1,66` reste loin de 1,25 : le
terme constant `+ 3*256` pèse encore lourd à cette échelle ; il ne devient négligeable que sur des
`cap` bien plus grandes, là où le ratio s'approche vraiment de **1,25×** (`1792 → 2560` donne déjà
≈ 1,43, toujours en décroissance). L'amortissement rend une suite de `n` `append` **O(n)** au total,
pas O(n²), quel que soit le ratio exact à chaque étape. 💡 Le premier `append` sur un slice **nil**
saute directement à `cap=4` (pour `[]int`) : l'`append` inliné arrondit à une petite size class. Ne
vous fiez pas aux valeurs exactes — fiez-vous à la **stratégie**.

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

L'invariant vérifié à l'exécution est `0 <= low <= high <= max <= cap(s)` (et `cap(arr) == len(arr)`
pour un array). Chaque indice doit rester dans les bornes du **précédent**, pas seulement dans celles
du backing physique. Le violer **panique**, immédiatement et sans condition — pas de lecture
hors-limites silencieuse à la C :

```go
s := make([]int, 5, 10)
_ = s[2:4:11] // panic: runtime error: slice bounds out of range [::11] with capacity 10
```

Cette vérification a un coût, mais le compilateur l'**élide** quand il peut prouver statiquement les
bornes respectées — le cas classique est l'indexation `s[i]` dans une boucle `for i := range s`. Pour
une expression de slicing à indices dynamiques comme ci-dessus, en revanche, elle reste presque
toujours systématique.

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
retient **tous**. La raison est **structurelle** : le GC de Go raisonne par **allocation entière**,
pas par octet utile — `big` et `big[:3]` pointent vers le **même** objet alloué (un seul appel à
l'allocateur), via le champ `array` de leurs headers respectifs. Tant qu'**un seul** pointeur vers cet
objet reste atteignable, rien n'est libéré : il n'existe pas de mécanisme pour découper un objet déjà
alloué en une partie vivante et une partie morte.

```go
// code/ch30-slices-profondeur/slicegrow.go
func TrimRetention(big []int, n int) []int {
	return slices.Clone(big[:n]) // copie -> backing de taille n ; l'ancien devient collectable
}
```

`slices.Clone` (ou `slices.Clip`, qui ramène `cap` à `len`) **détache** la vue en créant un **nouvel**
objet alloué, de taille minimale : l'ancien backing, lui, n'est alors plus référencé par personne et
devient collectable.

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
- **1.25 / 1.26** — l'**escape analysis** range de plus en plus de backings de taille **bornée et
  connue à la compilation** **sur la pile** ([Ch. 26](26-allocation-escape.md), mesuré :
  `sumSmallSlice` = 0 alloc/op) : un `make([]T, k)` local qui ne s'échappe pas n'alloue plus. La
  taille doit rester **statique** — un `make([]T, n)` avec `n` connu seulement à l'exécution continue,
  lui, de s'échapper sur le tas.

## ⚠️ Pièges

- **`append` qui écrase le parent** — un sous-slice avec `cap > len` partage le backing. Bornez la cap
  (`s[a:b:b]`) ou clonez avant de modifier.
- **Rétention d'un grand backing** — un petit slice d'un grand tableau le retient en entier.
  `slices.Clone`/`Clip` pour détacher.
- **Tronquer un slice de pointeurs sans vider la queue** — `s = s[:n]` (n < len) ne supprime rien :
  les éléments au-delà de `n` restent référencés par le backing, donc **vivants** pour le GC. Pour un
  `[]*T` ou un `[]any`, videz d'abord la queue (`clear(s[n:])`, 🆕 1.21) avant de tronquer.
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
- **Localité de cache** — le backing array est **contigu** : parcourir un `[]T` séquentiellement
  charge des lignes de cache entières utiles d'avance (_prefetching_ matériel), contrairement à une
  structure chaînée où chaque nœud peut vivre à une adresse arbitraire (même défaut de localité que
  celui évoqué pour le marquage GC au [Ch. 27](27-garbage-collector.md)). C'est une raison
  structurelle pour laquelle un `[]T` bat presque toujours une liste chaînée en Go, même à
  algorithmie équivalente.

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

- Un slice = **header de 3 mots** (`runtime.slice{array, len, cap}`, 24 o en 64 bits) **vue** sur un
  backing partagé ; copier un slice copie le header, pas les données. Un **array** n'a pas ce header :
  il **est** ses éléments, adressables directement par le compilateur dès la compilation.
- `append` **sur-alloue** via `nextslicecap` : **double** sous `cap=256`, puis croît par
  `(cap + 768) / 4` — un facteur qui ne s'approche de **1,25×** qu'**asymptotiquement** — le tout
  arrondi à une **size class** par `growslice`. Amortissement O(n) ; chaque réallocation **copie**.
- L'**expression à 3 indices** `s[a:b:c]` **borne la cap** — l'outil pour isoler un sous-slice et forcer
  la réallocation. La violer **panique** (`slice bounds out of range`), jamais de lecture hors-bornes
  silencieuse.
- Trois pièges : **aliasing** (`append` écrase le parent), **rétention** (un petit slice retient un
  grand backing — le GC libère par **objet alloué entier**, jamais une portion) et son inverse,
  **oublier de vider** un slice de pointeurs avant de le tronquer. Parade : borner la cap,
  `slices.Clone`/`Clip`, `clear()`.
- **Préallouer** et **réutiliser un buffer** (`buf[:0]`) éliminent les allocations sur le chemin
  chaud ; un backing **contigu** profite aussi du cache CPU, contrairement aux structures chaînées.

## 🔁 Pour aller plus loin

- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : size classes, pile vs tas, préallocation.
- [Ch. 27 — Garbage collector](27-garbage-collector.md) : pourquoi la rétention coûte cher.
- [Ch. 31 — Strings en profondeur](31-strings-profondeur.md) : le même genre de header, mais immuable.
- [Ch. 36 — Benchmarks & fuzzing](36-tests-benchmarks-fuzzing.md) : mesurer `allocs/op` rigoureusement.
- Doc : `go doc slices` ; `go doc builtin.append`.
