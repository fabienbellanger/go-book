# 6 — Arrays & slices (usage)

> **Objectif** — Manipuler les séquences et acquérir le **modèle mental du slice**
> (header `ptr`/`len`/`cap`), pour éviter ses pièges classiques.
>
> **Prérequis** — [Ch. 5 — Fonctions](05-fonctions.md)

---

## Introduction

Le **slice** est la structure de données la plus utilisée de Go — et la plus
mal comprise. Tout devient limpide une fois qu'on voit ce qu'il est vraiment : un petit
**descripteur à trois champs** qui pointe vers un tableau. Ce chapitre couvre l'usage ;
les internals (croissance, pile/tas) sont au [Ch. 30](30-slices-profondeur.md).

L'exemple complet est dans [`code/ch06-slices/`](../code/ch06-slices/).

## Arrays : taille fixe, valeur

Un **array** a une **taille fixe**, qui fait partie de son **type** (`[3]int` ≠ `[4]int`).
C'est un **type valeur** : l'affecter ou le passer à une fonction le **copie entièrement**.

```go
a := [3]int{1, 2, 3}
b := a   // COPIE intégrale
b[0] = 99
// a == [1 2 3], b == [99 2 3]   -> a est intact
```

- Taille déduite avec `...` : `[...]int{1, 2, 3}` (le compilateur compte → `[3]int`).
- Les arrays sont **comparables** avec `==` si leurs éléments le sont :
  `a == [3]int{1,2,3}` vaut `true`.

En pratique, on manipule rarement des arrays directement : on utilise des **slices**,
qui sont des **vues** sur des arrays.

## Le slice : un header sur un tableau

Un slice est un **descripteur de 3 mots machine** :

```
   s := arr[1:4]

   s  (slice header)
   +--------+--------+--------+
   |  ptr   | len=3  | cap=4  |
   +---+----+--------+--------+
       |
       v
   arr [0] [1] [2] [3] [4]
           ^----len----^
           ^-------cap-------^   (du début du slice à la fin du tableau)
```

- **`ptr`** : adresse du **premier élément** du slice dans le tableau sous-jacent.
- **`len`** : nombre d'éléments **accessibles** (`len(s)`).
- **`cap`** : nombre d'éléments **jusqu'à la fin** du tableau sous-jacent (`cap(s)`).

> 💡 Copier un slice (affectation, passage en argument) copie **le header**, pas les
> données : les deux copies **partagent** le même tableau (voir [Ch. 5](05-fonctions.md)).

## Créer un slice

```go
var s []int          // nil : len=0, cap=0, s == nil (mais utilisable !)
s = []int{1, 2, 3}   // littéral : len=3, cap=3
s = make([]int, 3)   // len=3, cap=3, rempli de zero values
s = make([]int, 3, 8) // len=3, cap=8 (réserve de la place)
```

> ⚠️ **`nil` slice vs slice vide** : `var s []int` est `nil` (`s == nil` vaut `true`),
> alors que `[]int{}` ne l'est pas. Les deux ont `len == 0` et fonctionnent avec `append`,
> `range`, `len` — préférez la forme `nil` par défaut.

## `append` & réallocation

`append` ajoute des éléments **à la fin** et **renvoie** le slice (potentiellement
nouveau) — il faut **toujours réaffecter** :

```go
s = append(s, 4)        // un élément
s = append(s, 5, 6, 7)  // plusieurs
s = append(s, autre...) // un autre slice, éclaté
```

Quand `len == cap`, il n'y a plus de place : `append` **alloue un nouveau tableau**, plus
grand, y **recopie** les éléments, et renvoie un header pointant dessus. La capacité croît
**géométriquement**. Progression réellement observée (Go 1.26) :

```
   cap : 0 -> 4 -> 8 -> 16 -> 32 -> 64 -> 128 -> 256 -> 512 -> 848 -> ...
         \________ ×2 tant que petit ________/   \__ ralentit (~×1.5) __/
```

```
   append qui réalloue (len == cap)
   --------------------------------
   avant : ptr --> [ 1 | 2 | 3 ]      len=3 cap=3   (plein)

   append(s, 4) :
   après : ptr --> [ 1 | 2 | 3 | 4 | _ | _ | _ | _ ]   len=4 cap=8
                   (NOUVEAU tableau ; l'ancien est abandonné au GC)
```

> ⚡ Si vous connaissez la taille finale, **préallouez** : `make([]T, 0, n)` évite les
> réallocations et copies successives (voir `filter` dans l'exemple).

## Slicing : `a[i:j]` et `a[i:j:k]`

```go
s := []int{0, 1, 2, 3, 4}
s[1:3]   // [1 2]        -> len=2, cap=4 (de l'index 1 à la fin)
s[:2]    // [0 1]        -> début omis = 0
s[2:]    // [2 3 4]      -> fin omise = len
s[:]     // tout
```

L'**expression à trois indices** `a[i:j:k]` fixe en plus la **capacité** (`cap = k - i`) :

```go
s[1:3:3] // [1 2] mais cap=2 -> tout append réallouera
```

C'est l'outil pour **isoler** un sous-slice et empêcher qu'un `append` ne déborde sur le
tableau partagé (voir le piège ci-dessous).

## `copy`

`copy(dst, src)` copie élément par élément, jusqu'à `min(len(dst), len(src))`, et renvoie
le nombre copié :

```go
dst := make([]int, 3)
n := copy(dst, []int{1, 2, 3, 4, 5}) // n=3, dst=[1 2 3]
```

## Aliasing : LE piège des slices

Un sous-slice **partage** le tableau du parent. Tant qu'il reste de la capacité, un
`append` sur le sous-slice **écrit dans le tableau du parent** :

```go
base := []int{1, 2, 3, 4, 5}
sub := base[1:3]      // [2 3], mais cap=4 !
sub = append(sub, 99) // place dispo -> écrit dans base[3]
// base == [1 2 3 99 5]   <-- base a été modifié à distance !
```

La parade : l'**expression à trois indices** borne la capacité, forçant `append` à
réallouer (donc à ne plus toucher au parent) :

```go
sub := base[1:3:3]    // cap=2
sub = append(sub, 99) // réalloue -> base intact
// base == [1 2 3 4 5]
```

## Slices de slices (2D)

Go n'a pas de tableau 2D natif : on compose des slices de slices.

```go
grid := [][]int{
	{1, 2, 3},
	{4, 5, 6},
}
grid[1][2] // 6
```

Chaque ligne est un slice indépendant (longueurs potentiellement différentes : tableau
« en dents de scie »).

## Le package `slices` (🆕 1.21)

La bibliothèque standard fournit des helpers génériques (voir [Ch. 11](11-genericite.md)) :

```go
slices.Sort(s)              // tri en place
slices.Contains(s, v)       // présence
slices.Index(s, v)          // position (ou -1)
slices.Equal(a, b)          // égalité élément par élément
clone := slices.Clone(s)    // copie indépendante (nouveau tableau)
s = slices.Clip(s)          // ramène cap à len (libère l'excédent)
```

> 💡 `slices.Clone` est la façon simple d'obtenir une copie **détachée** du tableau
> d'origine (utile contre les pièges d'aliasing et de rétention mémoire).

## ⚠️ Pièges

- **Aliasing après `append`** — un sous-slice avec du `cap` disponible écrit dans le
  parent. Utilisez `a[i:j:j]` ou `slices.Clone` pour isoler.
- **Rétention mémoire** — garder un **petit** sous-slice d'un **énorme** tableau empêche le
  GC de libérer ce dernier (le `ptr` le maintient vivant) :

  ```go
  func firstFew(huge []byte) []byte {
  	return huge[:3] // ⚠️ retient TOUT le tableau de huge
  }
  // Parade : renvoyer une copie -> slices.Clone(huge[:3])
  ```

- **Oublier de réaffecter `append`** — `append(s, x)` sans `s =` perd le résultat si une
  réallocation a lieu. Écrivez **toujours** `s = append(s, x)`.
- **`range` copie la valeur** — pour muter, indexez `s[i]` (rappel du [Ch. 4](04-flux-controle.md)).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch06-slices
go test ./ch06-slices/...
```

À essayer :

1. Dans `chunk`, remplacez `s[i:end:end]` par `s[i:end]` et observez l'échec du test
   `TestChunkNoAliasing` (aliasing réintroduit).
2. Mesurez l'effet d'une préallocation : comparez `var s []int` et `make([]int, 0, n)`
   dans une boucle d'`append` (benchmark, voir Ch. 36).
3. Écrivez `removeAt(s []int, i int) []int` sans fuite (attention au dernier élément).

---

## 📌 À retenir

- Un **array** a une taille fixe et est **copié** (type valeur, comparable).
- Un **slice** est un header **`ptr`/`len`/`cap`** : le copier partage le tableau sous-jacent.
- `append` peut **réallouer** (cap géométrique) → **toujours** `s = append(...)`.
- `a[i:j:k]` borne la **capacité** : l'outil anti-aliasing.
- Pièges majeurs : **aliasing** après append et **rétention** d'un gros tableau ;
  `slices.Clone` les neutralise.

## 🔁 Pour aller plus loin

- [Ch. 7 — Maps & strings](07-maps-strings.md).
- [Ch. 30 — Slices & arrays en profondeur](30-slices-profondeur.md) : stratégie de croissance, pile/tas, zéro-allocation.
- [Ch. 11 — Généricité](11-genericite.md) pour le package `slices`.
