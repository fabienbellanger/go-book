# 39 — Compilation, inlining, PGO & optimisations

> **Objectif** — Comprendre ce que fait le **compilateur Go** et comment l'**aider** : le pipeline
> (parse → types → SSA → asm), l'**inlining** (budget, `-gcflags=-m`), l'**escape analysis**,
> l'**élimination des contrôles de borne** (BCE), et la **PGO** (optimisation guidée par profil) qui
> recompile votre code à partir d'un profil de production.
>
> **Prérequis** — [Ch. 26](26-allocation-escape.md), [Ch. 37](37-profiling-pprof.md), [Ch. 33](33-interfaces-profondeur.md)

---

## Introduction

Le compilateur Go privilégie la **compilation rapide** et un code **prévisible** plutôt que les
optimisations les plus agressives (à la LLVM). Mais il fait beaucoup : inlining, analyse
d'échappement, élimination de contrôles, et — depuis 1.21 — **recompilation guidée par profil**. Bien
le comprendre, c'est écrire du code qu'il peut **optimiser**, et savoir **lire** ses décisions plutôt
que de les deviner. Code dans [`code/ch39-compilation-pgo/`](../code/ch39-compilation-pgo/).

---

## Le pipeline de compilation

```
  .go ──parse──> AST ──types──> IR typé ──SSA──> optimisations ──> asm ──> .o ──link──> binaire
                                          (inlining, escape, BCE,
                                           devirt, PGO...)
```

Tout passe par une **forme SSA** (Static Single Assignment) sur laquelle s'enchaînent des dizaines de
passes. On n'agit pas dessus directement, mais on **observe** ses décisions via `-gcflags`.

## L'inlining

**Inliner**, c'est remplacer un appel par le **corps** de la fonction. Le gain n'est pas tant l'appel
évité que les **optimisations débloquées** ensuite (constantes propagées, escape analysis affinée,
contrôles éliminés). Le compilateur inline tant qu'une fonction tient dans un **budget de coût** (~80
« nœuds »). On lit ses décisions avec **`-gcflags=-m`** :

```
$ go build -gcflags=-m -o /dev/null ./ch39-compilation-pgo
compile.go:5:6:  can inline add
compile.go:8:38: inlining call to add        <- l'appel a bien été inliné
compile.go:50:6: can inline Square.Area
```

```go
// code/ch39-compilation-pgo/compile.go
func add(a, b int) int { return a + b }     // minuscule -> inlinable
func AddTwice(n int) int { return add(n, n) } // `inlining call to add`
```

Ce qui **empêche** l'inlining : corps trop gros, **boucles** coûteuses, `defer` dans certains cas, appels
récursifs, et la directive explicite **`//go:noinline`**. Dans notre code, `TotalArea` la porte : elle
**n'apparaît pas** dans les « can inline ».

> 💡 N'écrivez pas des fonctions artificiellement petites « pour l'inlining » : le compilateur s'en
> charge. Mesurez ([Ch. 36](36-tests-benchmarks-fuzzing.md)) **avant** de vous en soucier.

## Escape analysis (rappel)

La même passe décide si une valeur vit sur la **pile** (gratuit) ou sur le **tas** (coût + GC). `-m` le
dit aussi ([Ch. 26](26-allocation-escape.md)) :

```
compile.go:12:15: xs does not escape         <- reste sur la pile
... escapes to heap                          <- alloué sur le tas
```

Une fonction inlinée permet souvent à l'escape analysis de prouver qu'une valeur **ne s'échappe pas** —
les deux optimisations se renforcent.

## Élimination des contrôles de borne (BCE)

Chaque accès `xs[i]` impose, en théorie, un **contrôle** : `0 <= i < len(xs)`, sinon `panic`. Le
compilateur **élimine** ce contrôle quand il peut **prouver** la sûreté. On l'observe avec
`-gcflags=-d=ssa/check_bce` (qui liste les contrôles **conservés**) :

```
$ go build -gcflags=-d=ssa/check_bce -o /dev/null ./ch39-compilation-pgo
compile.go:26:14: Found IsInBounds           <- contrôle CONSERVÉ (xs[i], i externe)
```

- **`SumRange`** (parcours `for range`) : aucun contrôle — le compilateur sait que l'indice est valide.
- **`SumGather`** (`xs[i]` où `i` vient d'un autre slice) : contrôle **conservé** (ligne 26).
- **`SumHinted`** : un accès **témoin** `_ = xs[3]` en tête prouve la sûreté des accès `xs[0..3]` suivants.

Le coût est réel et mesurable :

| Fonction    | ns/op     | Contrôle de borne |
| ----------- | --------- | ----------------- |
| `SumRange`  | **504,7** | éliminé           |
| `SumGather` | **595,6** | conservé          |

**~15 % d'écart** (0 allocation des deux côtés), uniquement dû au contrôle. La leçon : préférez le
**`range`**, et ne « micro-optimisez » à coups d'`unsafe` ([Ch. 35](35-unsafe-cgo.md)) que si un profil
le justifie.

## PGO : optimiser d'après un profil

La **Profile-Guided Optimization** (GA en **1.21**) recompile votre programme en s'appuyant sur un
**profil CPU de production** : le compilateur sait alors quelles fonctions sont **chaudes** et les
optimise plus agressivement. Le workflow tient en trois étapes :

```
  1. profiler la prod ───> cpu.prof
  2. déposer le profil ──> default.pgo  (à la racine du package main)
  3. go build  ──────────> -pgo=auto (défaut) le détecte et l'applique
```

```bash
go run ./ch39-compilation-pgo profile   # écrit default.pgo (profil CPU)
go build -pgo=auto .                     # auto : utilise default.pgo s'il existe
```

Deux optimisations principales, **ciblées sur les chemins chauds** :

- **Inlining plus agressif** des fonctions chaudes (budget relevé là où ça compte).
- **Dévirtualisation** des appels d'interface chauds quand un **type domine** : `s.Area()` devient un
  appel direct (puis inlinable). On le voit avec `-d=pgodebug=2` :

```
$ go build -pgo=auto -gcflags=-d=pgodebug=2 -o /dev/null .
compile.go:62:18: PGO devirtualize considering call s.Area()
hot-callsite-thres-from-CDF=0.6968...
```

Le compilateur **lit** le profil, calcule un **seuil de chaleur**, et **considère** la dévirtualisation
de l'appel polymorphe. Les gains typiques rapportés par l'équipe Go sont de **2 à 14 %** sur des services
réels — modestes mais **gratuits** (aucun changement de code), et **cumulatifs** d'une version à l'autre.

> 💡 **Versionnez `default.pgo`** avec le code : le build PGO devient reproductible et la CI applique la
> même optimisation. Rafraîchissez le profil périodiquement à partir de la production.

## Architecture cible : `GOAMD64`, FMA, DWARF5

- **`GOAMD64=v1..v4`** sélectionne un **niveau d'instructions** x86-64 (v2 = SSE3+, v3 = AVX2, v4 =
  AVX-512). En **v3+**, le compilateur émet des **FMA** (`a*b+c` en une instruction, plus précise et plus
  rapide). _(La machine de référence du livre est **arm64** : `GOAMD64` n'y a pas d'effet ; le FMA y est
  disponible nativement.)_
- **DWARF5** (par défaut en **1.25**) : infos de debug plus **compactes**, binaires plus petits, liens
  plus rapides — transparent pour delve/gdb.

---

## 🆕 Go 1.2x

- **1.21** — **PGO** en disponibilité générale : `-pgo=auto` détecte `default.pgo`. Dévirtualisation +
  inlining guidés par profil.
- **1.25** — **DWARF5** par défaut (debug plus compact) ; **FMA** émis en **`GOAMD64=v3`** et plus.
- **1.26** — **davantage de backing stores de slices alloués sur la pile** : l'escape analysis progresse,
  réduisant les allocations sans changer le code (vérifié en [Ch. 26](26-allocation-escape.md)). La PGO
  continue de s'affiner (meilleure dévirtualisation).

## ⚠️ Pièges

- **Réécrire pour « forcer » l'inlining** sans mesure : le compilateur décide mieux que l'intuition.
  Lisez `-m`, ne devinez pas.
- **`//go:noinline` par superstition** : c'est un outil de **diagnostic/benchmark**, pas une
  optimisation. En production, il **dégrade**.
- **Oublier `default.pgo`** à la racine du **package `main`** : `-pgo=auto` ne le trouve pas ailleurs.
- **Profil PGO périmé** : un profil qui ne reflète plus la charge optimise les mauvais chemins.
  Rafraîchissez-le.
- **`unsafe` pour éviter un bounds check** : presque toujours prématuré. Préférez `range` ou un témoin,
  et **mesurez** ([Ch. 35](35-unsafe-cgo.md)).

## ⚡ Performance

- **L'inlining débloque** les autres passes : c'est souvent le gain le plus important, et il est
  **automatique**.
- **BCE** : le `range` est gratuit ; l'indexation externe coûte un contrôle (~15 % ici). Aidez le
  compilateur par un **accès témoin** quand la boucle l'autorise.
- **PGO** : 2-14 % « gratuits » sur du code chaud, surtout sur les appels d'interface
  ([Ch. 33](33-interfaces-profondeur.md)) et les fonctions chaudes.
- 🔁 Le profil qui nourrit la PGO est celui du [Ch. 37](37-profiling-pprof.md) ; on **vérifie** le gain
  par benchmark + `benchstat` ([Ch. 36](36-tests-benchmarks-fuzzing.md)).

## 🧪 À tester soi-même

```bash
cd code
go build -gcflags=-m -o /dev/null ./ch39-compilation-pgo          # décisions d'inlining
go build -gcflags=-d=ssa/check_bce -o /dev/null ./ch39-compilation-pgo  # contrôles conservés
go test -bench='SumRange|SumGather' -benchmem -run=^$ ./ch39-compilation-pgo/...
( cd ch39-compilation-pgo && go run . profile && go build -pgo=auto -gcflags=-d=pgodebug=2 -o /dev/null . )
```

À essayer :

1. Ajoutez du code à `add` pour dépasser le budget : à quel moment `can inline add` disparaît-il ?
2. Récrivez `SumGather` avec un `range` sur `idx` **et** un accès témoin : le contrôle disparaît-il ?
3. Générez `default.pgo`, reconstruisez avec `-pgo=auto` puis `-pgo=off` et comparez `pgodebug`.

---

## 📌 À retenir

- Le compilateur passe par une **forme SSA** ; on observe ses décisions avec **`-gcflags=-m`** (inlining,
  escape) et **`-d=ssa/check_bce`** (contrôles de borne).
- **Inliner** débloque les autres optimisations ; budget ~80, `//go:noinline` pour le diagnostic
  seulement. Ne réécrivez pas « pour l'inlining ».
- **BCE** : `range` = aucun contrôle ; indexation externe = contrôle conservé (**~15 %** ici). Un accès
  **témoin** aide.
- **PGO** (🆕 1.21) : `default.pgo` + **`-pgo=auto`** → inlining agressif + **dévirtualisation** des
  appels chauds ; **2-14 %** gratuits, à **versionner** et rafraîchir.
- **`GOAMD64=v3+`** active les **FMA** (x86) ; **DWARF5** (1.25) allège le debug ; 1.26 met **plus de
  slices sur la pile**.

## 🔁 Pour aller plus loin

- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : produire le profil qui nourrit la PGO.
- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : l'analyse d'échappement en détail.
- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : ce que la dévirtualisation supprime.
- [Ch. 40 — Méthodologie de performance](40-methodologie-performance.md) : intégrer tout cela en un processus.
- Doc : `go doc cmd/compile`, `go help build` (flag `-pgo`), `go.dev/doc/pgo`.
