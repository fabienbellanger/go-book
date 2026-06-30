# 18 — Itérateurs par fonction (range-over-func)

> **Objectif** — Écrire et composer des **séquences paresseuses** avec les itérateurs introduits en
> Go 1.23 : `iter.Seq[V]` / `iter.Seq2[K,V]`, `for x := range monIterateur`, conversion _push_ ↔
> _pull_ (`iter.Pull`), combinateurs (`Map`/`Filter`/`Take`) et arrêt anticipé.
>
> **Prérequis** — [Ch. 15 — Closures](15-closures.md), [Ch. 11 — Généricité](11-genericite.md), [Ch. 4 — Flux de contrôle](04-flux-controle.md) (`range`)

---

## Introduction

Jusqu'en Go 1.22, `range` ne marchait que sur des types **prédéfinis** (slice, map, chaîne, canal,
entier). Pour exposer une séquence « maison » — les éléments d'un arbre, les lignes d'un fichier,
une suite calculée à la volée — il fallait soit la **matérialiser** entièrement dans une slice
(coût mémoire, perte de la paresse), soit la pousser dans un **canal** alimenté par une goroutine
dédiée (coût d'une goroutine et de synchronisation pour quelque chose qui n'a souvent rien de
concurrent), soit définir une interface `Iterator` maison avec `Next()`/`Value()` (verbeux, et
chaque consommateur doit réinventer sa propre boucle `for it.Next() { ... }`).

Depuis **Go 1.23**, on peut faire un `range` sur une **fonction** d'une forme particulière : un
**itérateur**. Cela permet d'exposer **n'importe quelle** séquence derrière la **même** syntaxe
`for range`, sans matérialiser de slice et sans lancer de goroutine. Le mécanisme est défini par le
**spec du langage** lui-même (pas seulement par convention de bibliothèque) : le compilateur
reconnaît la forme `func(yield func(V) bool)` et sait la consommer avec `for range`, exactement
comme il sait déjà consommer un canal ou une slice.

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

`Seq`/`Seq2` sont des **alias de confort** (comme `http.HandlerFunc`) : `range` ne reconnaît pas le
nom du type mais sa **forme**. Toute fonction dont la signature est `func(yield func(V) bool)` ou
`func(yield func(K, V) bool)` est itérable, qu'elle utilise `iter.Seq` ou non — c'est du **typage
structurel**, pas une interface à implémenter. Le spec du langage définit même une **troisième**
forme, sans valeur, `func(yield func() bool)` : utile pour un `for range f { ... }` qui répète une
action tant que le consommateur ne dit pas stop, sans transporter de donnée à chaque tour.

## Consommer : `for x := range`

Quand on écrit `for v := range seq`, le **corps de boucle** devient le `yield`. Sortir de la boucle
(`break`, `return`) fait renvoyer `false` à `yield`, ce qui arrête proprement l'itérateur. C'est un
**flux de contrôle inversé** par rapport à un `range` classique : ce n'est plus la boucle qui pioche
dans une collection, c'est l'itérateur qui **appelle** le corps de boucle, autant de fois qu'il a de
valeurs à produire.

```go
for i := range Count(5) {   // Count est un iter.Seq[int]
	fmt.Print(i, " ")       // 0 1 2 3 4
}
```

Le compilateur réécrit ce `for` en un appel direct à `Count(5)`, avec le **corps de boucle promu en
fonction `yield`** :

```go
// Équivalent conceptuel de la boucle ci-dessus (pas du code à écrire soi-même) :
Count(5)(func(i int) bool {
	fmt.Print(i, " ") // le corps de la boucle, tel quel
	return true       // sous-entendu en fin de corps : "continue"
})
// un `break`/`return` dans le corps devient un `return false` explicite à cet endroit
```

```
   for i := range Count(5) { corps }          Count(5)(yield)

   +-------------+   appelle   +------------------------+
   |   Count(5)  | ----------> |  yield(i)  (= "corps")  |
   |  (produit)  |             |  exécute le corps       |
   |             | <---------- |  renvoie true (suite)   |
   +-------------+   bool      |  ou false (break/return)|
        |                      +------------------------+
        | si false : Count(5) s'arrête (plus jamais d'appel à yield)
        v
   fin de boucle
```

Détail non évident : bien que le corps soit mécaniquement **promu en fonction**, le compilateur fait
en sorte que `break`, `continue`, `return` et même `goto` s'y comportent **exactement comme dans une
boucle ordinaire**. Un `return` dans le corps ne se contente pas de faire sortir `yield` avec
`false` : il sort bien de la **fonction englobante** (celle qui contient le `for range`), pas
seulement du `yield` synthétisé — contrairement à ce qui se passerait dans une closure ordinaire
passée en callback, où `return` ne quitterait que la closure elle-même.

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

Pourquoi une goroutine ? Un itérateur push ne sait que **pousser** : une fois lancé, il ne rend la
main qu'à la fin ou sur `yield(...) == false`. Pour le transformer en `next()` qui rend la main
**après chaque valeur**, il faut pouvoir **suspendre** son exécution au milieu d'un `yield` puis la
**reprendre** plus tard — Go n'a pas de coroutines symétriques natives. `iter.Pull` simule cette
suspension avec une goroutine séparée qui bloque sur un canal interne entre deux appels à `next()`.

> ⚠️ `iter.Pull` lance une **goroutine** pour exécuter l'itérateur push. Il **faut** appeler `stop()`
> (typiquement `defer stop()`) sinon cette goroutine **fuit**. C'est le seul cas où un itérateur
> coûte une goroutine. Les fonctions `next`/`stop` renvoyées ne sont **pas** sûres pour des appels
> concurrents (documentation `iter.Pull`) : un seul goroutine doit les piloter à la fois — cohérent
> avec le reste du chapitre, où un itérateur reste pensé pour une consommation séquentielle.

## Itérateur, canal ou slice : que choisir ?

Trois façons d'exposer une séquence à un appelant ; avant 1.23, le canal servait souvent à simuler
la paresse de l'itérateur, au prix d'une goroutine et d'une synchronisation inutiles dès qu'aucun
vrai parallélisme n'était en jeu.

| Approche                     | Allocation                         | Concurrence                       | Arrêt anticipé                                    | Cas d'usage typique                                    |
| ---------------------------- | ---------------------------------- | --------------------------------- | ------------------------------------------------- | ------------------------------------------------------ |
| `iter.Seq` (range-over-func) | aucune si inliné (⚡ ci-dessous)   | même goroutine que l'appelant     | natif — `break` propage `yield(false)`            | séquence calculée à la volée, pipelines composables    |
| `chan T`                     | le canal + chaque valeur transitée | **goroutine séparée obligatoire** | nécessite un signal explicite (`close`, `select`) | producteurs/consommateurs réellement asynchrones       |
| `[]T` matérialisée           | tout le contenu d'un coup          | aucune (donnée passive)           | trivial (`break` ordinaire)                       | accès répété, indexé ou aléatoire ; taille raisonnable |

Le canal reste le bon choix quand la production est **réellement concurrente** (ex. plusieurs
goroutines qui alimentent le même flux). Pour une séquence calculée **séquentiellement** — la
grande majorité des générateurs, transformations et filtres — un `iter.Seq` offre la même paresse
sans le coût d'une goroutine ni la latence d'un canal.

## Les itérateurs de la bibliothèque standard (1.23-1.24)

`slices` et `maps` exposent des itérateurs et des collecteurs :

| Fonction              | Renvoie            | Rôle                                       |
| --------------------- | ------------------ | ------------------------------------------ |
| `slices.Values(s)`    | `iter.Seq[E]`      | les valeurs d'une slice                    |
| `slices.All(s)`       | `iter.Seq2[int,E]` | les paires (index, valeur)                 |
| `slices.Backward(s)`  | `iter.Seq2[int,E]` | les paires (index, valeur), à **l'envers** |
| `slices.Collect(seq)` | `[]E`              | matérialise une `Seq` en slice             |
| `slices.Sorted(seq)`  | `[]E`              | collecte **et trie**                       |
| `maps.Keys(m)`        | `iter.Seq[K]`      | les clés (ordre non défini)                |
| `maps.Values(m)`      | `iter.Seq[V]`      | les valeurs                                |
| `maps.All(m)`         | `iter.Seq2[K,V]`   | les paires (clé, valeur) — un vrai `Seq2`  |
| `maps.Collect(seq)`   | `map[K]V`          | construit une map depuis des paires        |

```go
// Idiome déjà vu au Ch. 7 : trier les clés d'une map.
keys := slices.Sorted(maps.Keys(m))

// Backward : parcourir une slice en sens inverse sans la copier ni la trier à l'envers.
for i, v := range slices.Backward(words) {
	fmt.Println(i, v) // dernier index d'abord
}
```

> 💡 `strings` propose aussi des variantes itérateur depuis **1.24** : `SplitSeq`, `FieldsSeq`,
> `Lines` — détaillées au [Ch. 7](07-maps-strings.md), elles suivent exactement le même principe
> que `slices.Values` : `for w := range strings.FieldsSeq(text)` ne matérialise jamais la tranche
> de mots intermédiaire.

---

## 🆕 Go 1.2x

- **1.23** — le package **`iter`** (`Seq`, `Seq2`, `Pull`, `Pull2`) et le **`range`-over-func**.
  `slices` gagne `Values`/`All`/`Backward`/`Collect`/`Sorted` ; `maps` gagne `Keys`/`Values`/`All`/
  `Collect`.
- **1.24** — `strings` gagne ses propres itérateurs (`SplitSeq`, `FieldsSeq`, `Lines`, 🔁
  [Ch. 7](07-maps-strings.md)) : même principe que `slices.Values`, appliqué au texte.
- C'est le prolongement direct de la **portée par itération** (1.22, [Ch. 15](15-closures.md)) et de
  `for range N` (1.22, [Ch. 4](04-flux-controle.md)).

## ⚠️ Pièges

- **Appeler `yield` après un `false`** : interdit, **panique**. Toujours `return` dès que `yield`
  renvoie `false`. Le runtime le détecte et arrête tout avec
  `runtime error: range function continued iteration after function for loop body returned false`
  — voir `BrokenAfterStop` et `TestYieldAfterStopPanics` dans
  [`code/ch18-iterators/iterators.go`](../code/ch18-iterators/iterators.go) pour un cas reproduit et
  capturé avec `recover`.
- **Oublier `stop()` après `iter.Pull`** : fuite de goroutine. Mettez `defer stop()` juste après.
- **Croire que c'est lazy partout** : `slices.Collect`/`Sorted` **matérialisent** (et trient) — ils
  consomment toute la séquence. N'enchaînez pas un `Collect` au milieu d'un pipeline.
- **Itérateur infini sans borne côté consommateur** : `slices.Collect(Naturals())` boucle sans fin.
  Bornez avec `Take` ou un `break`.
- **Effets de bord dans `yield`** : le corps de boucle peut modifier l'état capturé ; attention en
  concurrence ([Ch. 23](23-patterns-concurrence.md)).
- **Supposer qu'un itérateur est rejouable** : `Count(5)` peut être parcouru plusieurs fois (chaque
  appel repart de `i := 0`), mais un itérateur qui capture une **ressource à état** — un
  `bufio.Scanner`, un curseur SQL — ne l'est pas : le second `range` repart d'où le premier s'est
  arrêté (souvent : épuisé). Rejouable ou non dépend uniquement de ce que la closure **capture**, pas
  de la forme `iter.Seq` elle-même.

## ⚡ Performance

Un itérateur simple est **inliné** : quand le compilateur voit, au même endroit, l'appel `range seq`
et le corps de `seq`, il peut fusionner les deux et faire disparaître l'indirection de `yield` — le
`range`-over-func compile alors vers une boucle quasi équivalente à la main, **sans allocation**. La
closure `yield` elle-même n'échappe jamais sur le tas puisqu'elle ne sert qu'à cet appel direct
(escape analysis, [Ch. 26](26-allocation-escape.md)). Le gain vient surtout d'**éviter la
matérialisation**. Mesuré (go1.26.4, somme de 0..999) :

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
4. Lancez `go test -run TestYieldAfterStopPanics -v ./ch18-iterators/` et lisez le message de
   panique : retrouvez, dans `BrokenAfterStop`, la ligne exacte qui le déclenche.

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
- Face à un canal ou une slice, l'itérateur gagne dès qu'il n'y a **pas de vrai parallélisme** à
  exprimer et qu'on ne consomme la séquence **qu'une fois**.
- `slices.Values/All/Backward/Collect/Sorted` et `maps.Keys/Values/All/Collect` (1.23) couvrent les
  cas courants ; `strings.SplitSeq/FieldsSeq/Lines` (1.24) font de même pour le texte.

## 🔁 Pour aller plus loin

- [Ch. 15 — Closures](15-closures.md) : un itérateur et ses combinateurs **sont** des closures.
- [Ch. 11 — Généricité](11-genericite.md) : `Map`/`Filter`/`Take` sont génériques.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : itérateurs et goroutines (Pull).
- [Ch. 36 — Benchmarks](36-tests-benchmarks-fuzzing.md) : mesurer itérateur vs slice matérialisée.
- [The Go Blog — Range Over Function Types](https://go.dev/blog/range-functions).
