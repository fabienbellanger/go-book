# 36 — Tests avancés, benchmarks & fuzzing

> **Objectif** — **Mesurer** et **sécuriser** correctement : écrire des **benchmarks** fiables
> (`testing.B`, `b.Loop`, `b.ReportAllocs`), éviter l'**élimination par le compilateur**, comparer
> deux versions avec **`benchstat`** (significativité statistique), et **fuzzer** une fonction
> (`testing.F`, corpus) pour débusquer les cas limites.
>
> **Prérequis** — [Ch. 13](13-tests-outillage.md), [Ch. 26](26-allocation-escape.md), [Ch. 31](31-strings-profondeur.md)

---

## Introduction

Le [Ch. 13](13-tests-outillage.md) a posé la **culture du test**. Cette Partie VI **ouvre la boîte à
outils de la performance**, et tout commence ici : on ne peut pas **optimiser ce qu'on ne mesure pas**.
Le même package `testing` gère trois familles — tests, **benchmarks**, **fuzzing** — sans dépendance
externe. Ce chapitre se concentre sur les deux dernières, et sur la **rigueur statistique** qui sépare
une vraie amélioration d'un bruit de mesure. Code dans [`code/ch36-benchmarks-fuzzing/`](../code/ch36-benchmarks-fuzzing/).

Fil rouge : `FormatThousands(n)` insère une espace tous les trois chiffres (`1234567` → `"1 234 567"`),
en deux implémentations — une **naïve** (concaténation, [Ch. 31](31-strings-profondeur.md)) et une via
**`strings.Builder`**. On va prouver laquelle est meilleure.

---

## Un benchmark fiable : `b.Loop`

Un benchmark est une fonction **`BenchmarkXxx(b *testing.B)`**. Depuis Go **1.24**, on cadence la mesure
avec **`b.Loop()`** (et non plus `for i := 0; i < b.N; i++`) : il **réinitialise le chrono** au premier
tour (le setup ne compte pas) et l'**arrête** au dernier (le cleanup non plus).

```go
// code/ch36-benchmarks-fuzzing/bench_test.go
var sink string // voir « anti-élimination » ci-dessous

func BenchmarkBuilder(b *testing.B) {
	b.ReportAllocs()           // ajoute B/op et allocs/op au rapport
	for b.Loop() {
		sink = FormatThousands(1234567)
	}
}
```

```
$ go test -bench=. -benchmem -run=^$ ./ch36-benchmarks-fuzzing/...
BenchmarkBuilder-8   17174445   69.30 ns/op   24 B/op   2 allocs/op
BenchmarkNaive-8     11294998  106.6  ns/op   24 B/op   3 allocs/op
```

- `-bench=.` sélectionne les benchmarks (regexp) ; `-run=^$` **désactive les tests** classiques.
- `-benchmem` (ou `b.ReportAllocs()`) ajoute **`B/op`** et **`allocs/op`** — souvent plus parlants que
  les nanosecondes. `-8` = `GOMAXPROCS`.
- Le nombre de tours (`17174445`) est **choisi par `go test`** pour atteindre une durée stable (~1 s par
  défaut, réglable par `-benchtime`).

## ⚠️ L'élimination par le compilateur (dead-code elimination)

Le piège n°1 du micro-benchmark : si le **résultat n'est pas utilisé**, le compilateur peut **supprimer
l'appel** que vous croyez mesurer. La parade : affecter le résultat à une **variable de package**
(le `sink`), que le compilateur ne peut pas prouver inutile.

```
  SANS sink (FAUX)                  AVEC sink (correct)
  for b.Loop() {                    for b.Loop() {
      FormatThousands(123)              sink = FormatThousands(123)
  }   ^                                 ^
      +-- résultat ignoré                   +-- résultat publié hors de la
          -> appel potentiellement              fonction -> l'appel DOIT
             éliminé -> 0.3 ns/op                 s'exécuter -> mesure réelle
```

> 💡 **`b.Loop` aide aussi** : contrairement à l'ancienne boucle `b.N`, il garantit que le corps est
> exécuté **au moins une fois par itération** et que les arguments **ne sont pas hissés** hors de la
> boucle. Le `sink` reste néanmoins la garantie pour la **valeur de retour**.

## Sous-benchmarks : `b.Run`

Comme `t.Run`, **`b.Run`** crée des **sous-benchmarks** nommés — idéal pour balayer des tailles d'entrée
et voir le coût **croître** :

```go
func BenchmarkSizes(b *testing.B) {
	for _, in := range []struct{ name string; n int }{
		{"digits=3", 999}, {"digits=7", 1234567}, {"digits=13", 1234567890123},
	} {
		b.Run(in.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink = FormatThousands(in.n)
			}
		})
	}
}
```

```
BenchmarkSizes/digits=3-8    52.52 ns/op   16 B/op   2 allocs/op
BenchmarkSizes/digits=7-8    69.58 ns/op   24 B/op   2 allocs/op
BenchmarkSizes/digits=13-8   92.41 ns/op   40 B/op   2 allocs/op
```

## Comparer deux versions : `benchstat`

Un benchmark isolé ne dit rien : **107 ns, est-ce mieux ou moins bien ?** La réponse exige une
**comparaison statistique**. L'outil officiel est **`benchstat`** (package `golang.org/x/perf`) :

```bash
go install golang.org/x/perf/cmd/benchstat@latest   # une fois

# On mesure 10 fois chaque version (le bruit l'exige) dans deux fichiers.
go test -bench=Naive   -count=10 -run=^$ ./ch36-benchmarks-fuzzing/... > old.txt
go test -bench=Builder -count=10 -run=^$ ./ch36-benchmarks-fuzzing/... > new.txt
benchstat old.txt new.txt
```

```
            │   old.txt   │             new.txt              │
            │   sec/op    │   sec/op     vs base             │
Format-8      106.90n ± 8%   69.60n ± 1%  -34.89% (p=0.000 n=10)
            │  allocs/op  │  allocs/op   vs base             │
Format-8       3.000 ± 0%    2.000 ± 0%  -33.33% (p=0.000 n=10)
```

- **`± 8%`** est la **variation** (médiane des écarts) : une mesure à ±20 % est trop bruitée, augmentez
  `-count` ou stabilisez la machine.
- **`p=0.000`** : la différence est **statistiquement significative** (p < 0,05). Un **`~`** signifierait
  « indistinguable du bruit » — ne jamais conclure dans ce cas.
- Verdict : `strings.Builder` est **−35 % de temps** et **−33 % d'allocations**. Réel, mesuré, prouvé.

> ⚠️ **Toujours `-count=10`** (ou plus) avant de conclure. Une seule mesure A et une seule mesure B ne
> permettent **aucune** inférence : le bruit (fréquence CPU, GC, voisins) domine souvent l'écart.

## Fuzzing : casser les invariants

Un test vérifie les cas **auxquels vous avez pensé** ; le **fuzzing** génère des entrées pour trouver
**ceux que vous avez oubliés**. Une fonction **`FuzzXxx(f *testing.F)`** sème un **corpus** (`f.Add`)
puis confie à `f.Fuzz` une propriété qui doit **toujours** tenir :

```go
// code/ch36-benchmarks-fuzzing/format_test.go
func FuzzFormatThousands(f *testing.F) {
	for _, seed := range []int{0, 1, -1, 999, 1000, math.MaxInt, math.MinInt} {
		f.Add(seed) // corpus de seeds : les cas limites connus
	}
	f.Fuzz(func(t *testing.T, n int) {
		got := FormatThousands(n)
		// Invariant : retirer les séparateurs redonne les chiffres d'origine.
		if strings.ReplaceAll(got, " ", "") != strconv.Itoa(n) {
			t.Errorf("FormatThousands(%d) = %q : invariant violé", n, got)
		}
	})
}
```

```bash
go test -run=^$ ./ch36-benchmarks-fuzzing/...                 # n'exécute QUE le corpus de seeds
go test -fuzz=FuzzFormatThousands -fuzztime=6s ./ch36-...     # fuzzing actif : génère des entrées
```

```
fuzz: elapsed: 6s, execs: 1445640 (248577/sec), new interesting: 1 (total: 9)
PASS
```

1,4 **million** d'entrées en 6 s, sur 8 workers, sans violer l'invariant. Un bon invariant capture une
**vérité métier** : ici, _le formatage ne perd ni n'invente de chiffre_. C'est ainsi que le fuzzing
trouve les classiques : débordement sur `math.MinInt`, indices hors bornes, paniques.

> 💡 **Deux corpus**. Les seeds `f.Add` et les entrées « intéressantes » découvertes vivent dans un
> **cache** (`$GOCACHE/fuzz`). Quand le fuzzer trouve un **crash**, il écrit l'entrée fautive dans
> **`testdata/fuzz/FuzzXxx/`** : ce fichier, **versionné**, devient un cas de **non-régression** rejoué
> à chaque `go test`. Le bug ne pourra plus jamais repasser inaperçu.

## Couverture (rappel)

```bash
go test -cover ./ch36-benchmarks-fuzzing/...                       # % de lignes exécutées
go test -coverprofile=cover.out ./ch36-benchmarks-fuzzing/...
go tool cover -html=cover.out                                      # visualisation ligne à ligne
```

La couverture **mesure l'exécution, pas la pertinence** ([Ch. 13](13-tests-outillage.md)) : c'est un
indicateur, pas un objectif. Le fuzzing, lui, **augmente** mécaniquement la couverture en explorant des
branches oubliées.

---

## 🆕 Go 1.2x

- **1.24** — **`b.Loop()`** : la façon moderne de cadencer un benchmark. Plus sûr que `b.N` (chrono géré,
  pas de hissage d'arguments), il devient l'idiome recommandé.
- **1.26** — **`b.Loop` n'empêche plus l'inlining** du corps mesuré (d'après les notes de version) :
  auparavant, l'appel à `b.Loop` formait une frontière qui pouvait empêcher le compilateur d'**inliner**
  la fonction testée, faussant les micro-benchmarks ; le code mesuré est désormais optimisé **comme en
  production**.
- **1.24** — le **fuzzing** continue de mûrir (corpus partagé, intégration `go test`). Fuzzer fait
  partie de la CI sérieuse, pas seulement du debug ponctuel.

## ⚠️ Pièges

- **Résultat ignoré** → appel éliminé → benchmark qui mesure du vide (0,3 ns/op suspect). Publiez via un
  **`sink`** de package.
- **Une seule mesure** par version → conclusion invalide. **`-count=10`** + **`benchstat`** + vérifier
  **`p < 0,05`**.
- **Setup dans la boucle** → vous mesurez le setup. Sortez-le avant `for b.Loop()` (le chrono démarre au
  premier tour).
- **Machine instable** (turbo, thermique, autres process) → `± 20 %`. Fermez les voisins, fixez la
  fréquence si possible, augmentez `-count`.
- **Invariant de fuzz trop faible** (`!= ""`) → ne trouve rien. Visez une **propriété métier** forte
  (round-trip, idempotence, conservation).

## ⚡ Performance

- **`B/op` et `allocs/op` d'abord** : une allocation en moins vaut souvent plus que quelques
  nanosecondes ([Ch. 26](26-allocation-escape.md)). Ici, `strings.Builder` économise une allocation
  **et** du temps.
- **`-benchtime=10000x`** fixe un **nombre de tours** (au lieu d'une durée) pour comparer à charge
  strictement égale.
- 🔁 Les chiffres ne disent pas **où** part le temps : c'est le rôle du **profiling**
  ([Ch. 37](37-profiling-pprof.md)) et des **traces** ([Ch. 38](38-traces-flight-recorder.md)).

## 🧪 À tester soi-même

```bash
cd code
go test -bench=. -benchmem -run=^$ ./ch36-benchmarks-fuzzing/...
go test -fuzz=FuzzFormatThousands -fuzztime=10s ./ch36-benchmarks-fuzzing/...
```

À essayer :

1. Retirez le `sink` de `BenchmarkBuilder` et observez le temps tomber à ~0,3 ns/op (appel éliminé).
2. Lancez les deux versions avec `-count=10` et comparez-les vous-même avec `benchstat`.
3. Introduisez un bug (oubliez le signe `-`) et relancez le fuzzer : il dépose un crasher dans
   `testdata/fuzz/`.

---

## 📌 À retenir

- Benchmark = **`BenchmarkXxx(b *testing.B)`** + **`for b.Loop()`** (🆕 1.24) + **`b.ReportAllocs()`** ;
  `-benchmem` donne `B/op` / `allocs/op`.
- **Toujours publier le résultat** dans un `sink` de package, sinon le compilateur **élimine** l'appel
  mesuré.
- Une mesure isolée ne conclut pas : **`-count=10`** + **`benchstat`** + **`p < 0,05`** distinguent le
  gain réel du bruit.
- **Fuzzing** = **`FuzzXxx(f *testing.F)`** + seeds **`f.Add`** + **invariant** dans `f.Fuzz` ; un crash
  est figé dans **`testdata/fuzz/`** comme test de non-régression.
- La **couverture** mesure l'exécution, pas la pertinence — un indicateur, jamais un objectif.

## 🔁 Pour aller plus loin

- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : trouver **où** le temps et la mémoire partent.
- [Ch. 39 — Compilation & inlining](39-compilation-inlining-pgo.md) : pourquoi un appel est inliné (ou non).
- [Ch. 40 — Méthodologie de performance](40-methodologie-performance.md) : la boucle mesure → hypothèse → re-mesure.
- Doc : `go help test`, `go help testflag`, `go doc testing.B`, `pkg.go.dev/golang.org/x/perf/cmd/benchstat`.
