# Ch. 18 — Itérateurs par fonction (range-over-func)

> **Objectif** — Écrire et composer des **séquences paresseuses** avec les itérateurs introduits en
> Go 1.23 : `iter.Seq[V]` / `iter.Seq2[K,V]`, `for x := range monIterateur`, conversion _push_ ↔
> _pull_ (`iter.Pull`), combinateurs (`Map`/`Filter`/`Take`) et arrêt anticipé.
>
> **Prérequis** — [Ch. 15 — Closures](15-closures.md), [Ch. 11 — Généricité](11-genericite.md), [Ch. 4 — Flux de contrôle](04-flux-controle.md) (`range`)

---

## Introduction

Jusqu'en Go 1.22, `range` ne marchait que sur des types **prédéfinis** (slice, map, chaîne, canal,
entier). Depuis **Go 1.23**, on peut faire un `range` sur une **fonction** d'une forme particulière :
un **itérateur**. Cela permet d'exposer **n'importe quelle** séquence — les éléments d'un arbre, les
lignes d'un fichier, une suite infinie — derrière la **même** syntaxe `for range`, sans matérialiser
de slice.

Le package [`iter`](https://pkg.go.dev/iter) définit les types ; `slices` et `maps` fournissent des
itérateurs prêts à l'emploi. L'exemple est dans [`code/ch18-iterators/`](../code/ch18-iterators/).

---

## Les deux types d'itérateurs

Un itérateur est une fonction qui reçoit une fonction `yield` et lui **pousse** chaque élément :

```go
type (
	Seq[V any]     func(yield func(V) bool)        // une valeur par élément
	Seq2[K, V any] func(yield func(K, V) bool)     // deux valeurs (clé/valeur, index/valeur)
)
```

`yield` renvoie un **booléen** : `true` = continue, `false` = le consommateur veut **arrêter** (un
`break`). L'itérateur **doit** respecter ce signal.

## Consommer : `for x := range`

Quand on écrit `for v := range seq`, le **corps de boucle** devient le `yield`. Sortir de la boucle
(`break`, `return`) fait renvoyer `false` à `yield`, ce qui arrête proprement l'itérateur.

```go
for i := range Count(5) {   // Count est un iter.Seq[int]
	fmt.Print(i, " ")       // 0 1 2 3 4
}
```

## Écrire un itérateur (push)

On boucle sur la source et on **pousse** chaque valeur dans `yield`, en s'arrêtant s'il renvoie
`false` :

```go
// code/ch18-iterators/iterators.go
func Count(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			if !yield(i) {
				return // le consommateur a fait un break : on s'arrête
			}
		}
	}
}
```

Comme rien n'est calculé tant qu'on n'itère pas, un itérateur peut être **infini** — le consommateur
décide quand s'arrêter :

```go
func Naturals() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; ; i++ { // pas de borne : 0, 1, 2, ...
			if !yield(i) {
				return
			}
		}
	}
}
```

## Composer des itérateurs

Un itérateur **prend** un itérateur et en **renvoie** un autre : la composition est naturelle et
**paresseuse** (chaque combinateur est une closure, [Ch. 15](15-closures.md)). Rien n'est matérialisé
entre les étapes.

```go
func Map[A, B any](seq iter.Seq[A], f func(A) B) iter.Seq[B] {
	return func(yield func(B) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}
// Filter et Take suivent le même squelette.

// Carrés des nombres pairs, 3 premiers — sur une source INFINIE :
pipeline := Take(Map(Filter(Naturals(), even), square), 3)
slices.Collect(pipeline) // [0 4 16]
```

Seuls **3** éléments sont réellement calculés : `Take` arrête la chaîne dès qu'il en a assez, et
l'arrêt **se propage** jusqu'à `Naturals` via le `yield` qui renvoie `false`.

## Push vs pull : `iter.Pull`

Dans un itérateur **push**, c'est la **séquence** qui mène la danse (elle appelle `yield`). Parfois
on a besoin de l'inverse — **tirer** les éléments à la demande, par exemple pour avancer **deux**
séquences en parallèle. `iter.Pull` convertit un itérateur push en deux fonctions `next`/`stop` :

```
  PUSH (iter.Seq) : l'itérateur mène               PULL (iter.Pull) : le consommateur mène
     for v := range seq { ... }                       next, stop := iter.Pull(seq)

   +----------+   yield(v)   +-----------+         +--------------+   next()   +----------+
   |   seq    | -----------> |  corps    |         | consommateur | ---------> |   seq    |
   | (pousse) | <----------- | (bool)    |         |   (tire)     | <--------- | (fournit)|
   +----------+              +-----------+         +--------------+  v, ok     +----------+
```

```go
func Zip[A, B any](sa iter.Seq[A], sb iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		nextA, stopA := iter.Pull(sa)
		defer stopA() // IMPÉRATIF : libère la ressource sous-jacente
		nextB, stopB := iter.Pull(sb)
		defer stopB()
		for {
			a, okA := nextA()
			b, okB := nextB()
			if !okA || !okB {
				return
			}
			if !yield(a, b) {
				return
			}
		}
	}
}
```

> ⚠️ `iter.Pull` lance une **goroutine** pour exécuter l'itérateur push. Il **faut** appeler `stop()`
> (typiquement `defer stop()`) sinon cette goroutine **fuit**. C'est le seul cas où un itérateur
> coûte une goroutine.

## Les itérateurs de la bibliothèque standard (1.23)

`slices` et `maps` exposent des itérateurs et des collecteurs :

| Fonction              | Renvoie            | Rôle                           |
| --------------------- | ------------------ | ------------------------------ |
| `slices.Values(s)`    | `iter.Seq[E]`      | les valeurs d'une slice        |
| `slices.All(s)`       | `iter.Seq2[int,E]` | les paires (index, valeur)     |
| `slices.Collect(seq)` | `[]E`              | matérialise une `Seq` en slice |
| `slices.Sorted(seq)`  | `[]E`              | collecte **et trie**           |
| `maps.Keys(m)`        | `iter.Seq[K]`      | les clés (ordre non défini)    |
| `maps.Values(m)`      | `iter.Seq[V]`      | les valeurs                    |

```go
// Idiome déjà vu au Ch. 7 : trier les clés d'une map.
keys := slices.Sorted(maps.Keys(m))
```

---

## 🆕 Go 1.2x

- **1.23** — le package **`iter`** (`Seq`, `Seq2`, `Pull`, `Pull2`) et le **`range`-over-func**.
  `slices`/`maps` gagnent `Values`/`All`/`Collect`/`Sorted`/`Keys`/`Values`.
- C'est le prolongement direct de la **portée par itération** (1.22, [Ch. 15](15-closures.md)) et de
  `for range N` (1.22, [Ch. 4](04-flux-controle.md)).

## ⚠️ Pièges

- **Appeler `yield` après un `false`** : interdit, **panique**. Toujours `return` dès que `yield`
  renvoie `false`.
- **Oublier `stop()` après `iter.Pull`** : fuite de goroutine. Mettez `defer stop()` juste après.
- **Croire que c'est lazy partout** : `slices.Collect`/`Sorted` **matérialisent** (et trient) — ils
  consomment toute la séquence. N'enchaînez pas un `Collect` au milieu d'un pipeline.
- **Itérateur infini sans borne côté consommateur** : `slices.Collect(Naturals())` boucle sans fin.
  Bornez avec `Take` ou un `break`.
- **Effets de bord dans `yield`** : le corps de boucle peut modifier l'état capturé ; attention en
  concurrence ([Ch. 23](23-patterns-concurrence.md)).

## ⚡ Performance

Un itérateur simple est **inliné** : le `range`-over-func compile vers une boucle quasi équivalente à
la main, **sans allocation**. Le gain vient surtout d'**éviter la matérialisation**. Mesuré
(go1.26.4, somme de 0..999) :

```
   BenchmarkIterator        1981 ns/op       0 B/op   0 allocs   (aucune slice)
   BenchmarkMaterialized    2889 ns/op    8192 B/op   1 alloc    (slice de 1000 ints)
```

- Quand on construirait une slice **juste pour l'itérer une fois**, l'itérateur est à la fois plus
  rapide et **sans allocation**.
- Si la slice peut rester sur la **pile** (escape analysis, [Ch. 26](26-allocation-escape.md)), la
  matérialiser redevient compétitif : mesurez votre cas réel.
- Les **chaînes profondes** de combinateurs empilent des appels de `yield` : un poil de surcoût par
  niveau, à comparer au coût d'une slice intermédiaire par étape. 🔁 [Ch. 36](36-tests-benchmarks-fuzzing.md).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch18-iterators
go test ./ch18-iterators/...
go test -bench=. -benchmem ./ch18-iterators/...
```

À essayer :

1. Écrivez `Reduce[V, A any](seq iter.Seq[V], init A, f func(A, V) A) A` (repli paresseux).
2. Ajoutez un combinateur `TakeWhile(seq, pred)` qui s'arrête au premier élément refusé.
3. Mesurez `Zip` : combien de goroutines `iter.Pull` crée-t-il ? Que se passe-t-il sans `stop()` ?

---

## 📌 À retenir

- Un **itérateur** = une fonction `func(yield func(V) bool)` (`iter.Seq`) ; `range` la consomme, le
  **corps de boucle** est le `yield`.
- `yield` renvoie `false` quand le consommateur **arrête** (break) ; l'itérateur **doit** s'arrêter
  alors (et **jamais** rappeler `yield`).
- Les itérateurs se **composent** paresseusement (`Map`/`Filter`/`Take`) sans matérialiser ; ils
  peuvent être **infinis**.
- `iter.Pull` convertit **push → pull** (`next`/`stop`) ; il lance une goroutine, d'où le
  `defer stop()` **obligatoire**.
- `slices.Values/All/Collect/Sorted` et `maps.Keys/Values` (1.23) couvrent les cas courants.

## 🔁 Pour aller plus loin

- [Ch. 15 — Closures](15-closures.md) : un itérateur et ses combinateurs **sont** des closures.
- [Ch. 11 — Généricité](11-genericite.md) : `Map`/`Filter`/`Take` sont génériques.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : itérateurs et goroutines (Pull).
- [Ch. 36 — Benchmarks](36-tests-benchmarks-fuzzing.md) : mesurer itérateur vs slice matérialisée.
- [The Go Blog — Range Over Function Types](https://go.dev/blog/range-functions).
