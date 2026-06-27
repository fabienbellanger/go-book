# Ch. 40 — Méthodologie de performance

> **Objectif** — Adopter un **processus** rigoureux plutôt que des micro-optimisations au hasard :
> définir un **budget/SLO**, **mesurer** d'abord, **localiser** (profil/trace), **corriger**, puis
> **re-vérifier** avec `benchstat` — en mesurant **tous les axes** (temps, mémoire, p99), car une
> « victoire » sur l'un peut être une **régression** sur l'autre.
>
> **Prérequis** — [Ch. 36](36-tests-benchmarks-fuzzing.md), [Ch. 37](37-profiling-pprof.md), [Ch. 39](39-compilation-inlining-pgo.md)

---

## Introduction

Ce chapitre **clôt la Partie VI** et relie tous ses outils en une **discipline**. La citation de Knuth
est mal comprise : « l'optimisation prématurée est la racine de tous les maux » ne dit pas _n'optimisez
jamais_, mais _n'optimisez pas **avant de mesurer**_. L'intuition sur les points chauds se trompe presque
toujours. La méthode, elle, ne se trompe pas : elle **mesure**. Code dans
[`code/ch40-methodologie/`](../code/ch40-methodologie/).

---

## La boucle

```
        +-----------------------------------------------------------+
        |                                                           |
        v                                                           |
   [0. budget/SLO] -> [1. mesurer] -> [2. localiser] -> [3. corriger]
        défini          benchmark        profil/trace      1 hypothèse
                        représentatif    (pprof/trace)      à la fois
                                                              |
        [5. garder/jeter] <- [4. re-mesurer] <----------------+
         selon le SLO          benchstat (p<0,05, TOUS les axes)
```

**Une hypothèse à la fois.** Changer trois choses puis re-mesurer ne dit pas **laquelle** a agi (ni
laquelle a **nui**).

## Étape 0 — Définir le budget (SLO)

On n'optimise pas « pour que ce soit plus rapide » mais **vers une cible** : « p99 < 50 ms », « < 100 Mo
de RSS », « 10 k req/s ». Sans cible, on ne sait pas **quand s'arrêter** — ni quel **axe** compte. La
latence ? Le débit ? La mémoire ? Le SLO **arbitre les compromis** de l'étape 4.

## Étape 1 — Mesurer d'abord

Un **benchmark représentatif** ([Ch. 36](36-tests-benchmarks-fuzzing.md)) sur des données réalistes.
Notre fil rouge : dédupliquer 4000 chaînes (beaucoup de doublons). La version « évidente » :

```go
// code/ch40-methodologie/dedup.go
func DedupNaive(items []string) []string {
	var out []string
	for _, it := range items {
		if !slices.Contains(out, it) { // appartenance O(n) DANS une boucle -> O(n^2)
			out = append(out, it)
		}
	}
	return out
}
```

```
BenchmarkDedupNaive-8   463   2472142 ns/op   18811 B/op   10 allocs/op
```

2,5 **ms** par appel. Est-ce un problème ? **Seul le SLO le dit.** Supposons que oui.

## Étape 2 — Localiser

On ne devine pas : on **profile** ([Ch. 37](37-profiling-pprof.md)). Un profil CPU de `DedupNaive`
désigne sans ambiguïté **`slices.Contains`** — le scan linéaire répété. Le coupable n'est pas la
fonction, c'est la **structure de données** : une appartenance en O(n) dans une boucle donne du O(n²).

> 💡 Pour une **latence** (et non un débit), le profil ne suffit pas : une **trace**
> ([Ch. 38](38-traces-flight-recorder.md)) montre l'**attente** (GC, scheduler, verrou) qu'un profil CPU
> ne voit pas.

## Étape 3 — Corriger (une hypothèse)

**Hypothèse** : remplacer l'appartenance linéaire par une **map** (O(1) amorti) ramène le tout à O(n).

```go
// code/ch40-methodologie/dedup.go
func Dedup(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items)) // préalloc : favorise la VITESSE
	for _, it := range items {
		if _, ok := seen[it]; !ok {
			seen[it] = struct{}{}
			out = append(out, it)
		}
	}
	return slices.Clip(out)
}
```

Un **test** garantit que la correction **ne change pas le résultat** (`slices.Equal(DedupNaive, Dedup)`) —
une optimisation qui casse le comportement n'en est pas une.

## Étape 4 — Re-mesurer (et mesurer TOUS les axes)

```bash
go test -bench=Dedup -benchmem -count=8 -run=^$ ./ch40-methodologie/... # avant ET après
benchstat old.txt new.txt
```

```
        │ old.txt (naïve) │      new.txt (map)          │
        │     sec/op      │   sec/op      vs base        │
Dedup-8     2464.43µ ± 4%    76.57µ ± 0%   -96.89% (p=0.000 n=8)
        │      B/op       │    B/op       vs base        │
Dedup-8      18.36Ki ± 0%   277.31Ki ± 0%  +1410.47% (p=0.000 n=8)
        │    allocs/op    │  allocs/op    vs base        │
Dedup-8       10.00 ± 0%     18.00 ± 0%     +80.00% (p=0.000 n=8)
```

**32× plus rapide** (−96,9 %, p significatif) — mais **+1410 % de mémoire** ! La préallocation à
`len(items)` (4000) alors que seuls **500** sont distincts gaspille de la mémoire. La méthode vient de
**révéler un compromis** que l'intuition aurait masqué.

## Étape 5 — Garder ou jeter, selon le SLO

C'est ici que le **budget de l'étape 0 tranche** :

| Si le SLO vise… | Choix                           | Mesure            |
| --------------- | ------------------------------- | ----------------- |
| la **latence**  | `Dedup` (préalloc)              | **76 µs**, 277 Ki |
| la **mémoire**  | sans préalloc (laisser grandir) | 87 µs, **73 Ki**  |

Sans préallocation, on mesure **87 µs / 73 Ki / 25 allocs** : un peu plus lent, **4× plus léger**. Aucune
des deux n'est « la bonne » dans l'absolu — **le SLO décide**. La leçon centrale : **mesurez tous les
axes**, un gain n'est validé que rapporté à la **cible**.

---

## 🆕 Go 1.2x

- **1.24** — **`b.Loop`** rend les benchmarks plus fiables (chrono, pas de hissage) :
  [Ch. 36](36-tests-benchmarks-fuzzing.md).
- **1.25** — **`testing/synctest`** ([Ch. 23](23-patterns-concurrence.md)) teste la latence à horloge
  **virtuelle** (pas de `time.Sleep` réel) ; **Flight Recorder** ([Ch. 38](38-traces-flight-recorder.md))
  capture l'avant-incident en production.
- **1.26** — **Green Tea GC** par défaut ([Ch. 27](27-garbage-collector.md)) réduit l'overhead GC ;
  **PGO** ([Ch. 39](39-compilation-inlining-pgo.md)) affine les chemins chauds. Le runtime optimise
  **pour** vous — encore faut-il **mesurer** l'effet.

## ⚠️ Pièges

- **Optimiser sans profil** — vous corrigez ce que vous **croyez** chaud, pas ce qui l'**est**. Profilez.
- **Micro-benchmark trompeur** — résultat hissé/éliminé ([Ch. 36](36-tests-benchmarks-fuzzing.md)), données
  irréalistes, cache chaud. Le `sink` et des entrées **représentatives** sont obligatoires.
- **Conclure sur la moyenne** — la **p99/p999** fait l'expérience utilisateur, pas la moyenne. Un GC mal
  réglé dégrade la **queue** sans bouger la moyenne.
- **Une seule mesure** — sans `-count` ni `benchstat`, on confond gain et bruit.
- **Changer plusieurs choses** à la fois — impossible d'attribuer l'effet. **Une hypothèse par tour.**
- **Ignorer un axe** — gagner 32× en temps en perdant 14× en mémoire **peut** violer le SLO.

## ⚡ Performance

- **`GOMEMLIMIT`** ([Ch. 27](27-garbage-collector.md)) borne la mémoire et lisse le GC sous pression —
  un levier de p99 sans changer une ligne de code.
- **Réduire les allocations** ([Ch. 26](26-allocation-escape.md)) est souvent le meilleur gain de
  latence : moins de travail pour le GC, moins de pauses.
- **Le profil de production** (via `net/http/pprof`, [Ch. 37](37-profiling-pprof.md)) prime sur le profil
  de laboratoire : données, charge et cache y diffèrent.
- L'ordre de rentabilité, en général : **algorithme/structure** (ici 32×) ≫ **allocations** ≫
  **micro-optimisations** (BCE, `unsafe`). Commencez par le haut.

## 🧪 À tester soi-même

```bash
cd code
go test -bench=. -benchmem -count=8 -run=^$ ./ch40-methodologie/...
# Comparez vous-même avec benchstat (cf. Ch. 36) ; profilez DedupNaive (cf. Ch. 37).
```

À essayer :

1. Retirez la préallocation de `Dedup` et re-mesurez : retrouvez-vous ~87 µs / 73 Ki ?
2. Profilez `DedupNaive` (`go test -cpuprofile`) et vérifiez que `slices.Contains` domine.
3. Définissez un SLO (« < 100 µs ET < 100 Ki ») : laquelle des trois versions le respecte ?

---

## 📌 À retenir

- **Mesure → hypothèse → changement → re-mesure**, en boucle, **une hypothèse à la fois**. L'intuition
  sur les points chauds se trompe ; la **méthode** non.
- **Définir le SLO d'abord** (latence p99 ? mémoire ? débit ?) : il dit **quand s'arrêter** et **arbitre
  les compromis**.
- **Localiser avec un profil/trace**, jamais en devinant : ici, `slices.Contains` (O(n²)) → map (O(n)),
  **32×**.
- **Mesurer TOUS les axes** : le même changement gagne **96,9 % de temps** mais **+1410 % de mémoire** —
  le SLO tranche.
- Rentabilité décroissante : **algorithme/structure** ≫ **allocations** ≫ **micro-optimisations**.
  Valider chaque gain avec **`benchstat`** (`p < 0,05`).

## 🔁 Pour aller plus loin

- [Ch. 36 — Benchmarks & fuzzing](36-tests-benchmarks-fuzzing.md) : mesurer et comparer (benchstat).
- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : localiser le coût.
- [Ch. 38 — Traces & Flight Recorder](38-traces-flight-recorder.md) : diagnostiquer la latence.
- [Ch. 39 — Compilation & PGO](39-compilation-inlining-pgo.md) : laisser le compilateur optimiser.
- [Ch. 27 — GC](27-garbage-collector.md) : `GOGC`/`GOMEMLIMIT` pour la mémoire et la p99.
- **Projet 7 — Profiling d'un service réel** : la méthode de bout en bout sur un cas complet.
