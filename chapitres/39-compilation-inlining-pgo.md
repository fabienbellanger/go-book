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
optimisations les plus agressives (à la LLVM) — ce n'est pas accidentel, c'est la même philosophie de
conception qu'au [Ch. 0](00-pourquoi-go.md). Mais il fait beaucoup : inlining, analyse d'échappement,
élimination de contrôles, et — depuis 1.21 — **recompilation guidée par profil**. Bien le comprendre,
c'est écrire du code qu'il peut **optimiser**, et savoir **lire** ses décisions plutôt que de les
deviner. Code dans [`code/ch39-compilation-pgo/`](../code/ch39-compilation-pgo/).

---

## Le pipeline de compilation

```
  .go --parse--> AST --types--> IR typé --SSA--> optimisations --> asm --> .o --link--> binaire
                                          (inlining, escape, BCE,
                                           devirt, PGO...)
```

Tout passe par une **forme SSA** (Static Single Assignment) : chaque valeur intermédiaire n'est
assignée **qu'une seule fois**, ce qui simplifie radicalement le raisonnement sur le flot de données —
une passe qui se demande « d'où vient cette valeur ? » n'a qu'un seul endroit à regarder, jamais une
réaffectation à retrouver en remontant le code. C'est ce qui permet d'empiler des dizaines de passes
indépendantes (inlining, escape analysis, BCE, dévirtualisation, PGO...) sans qu'elles interfèrent entre
elles. On n'agit pas sur cette forme directement, mais on **observe** ses décisions via `-gcflags`.

## L'inlining

**Inliner**, c'est remplacer un appel par le **corps** de la fonction. Le gain n'est pas tant l'appel
évité (quelques nanosecondes sur du matériel moderne) que les **optimisations débloquées** ensuite : une
fois le corps de l'appelée **inséré tel quel** dans l'appelante, les deux corps n'en forment plus qu'un
seul, et chaque passe suivante (constantes, escape analysis, BCE) raisonne sur ce tout — avec une vision
qu'elle n'avait pas en analysant les deux fonctions séparément. Le compilateur inline tant qu'une
fonction tient dans un **budget de coût** (un score interne sans unité physique : ~80 « nœuds » de
l'arbre, où une affectation, un appel ou une boucle coûtent plus cher qu'une opération simple). On lit
ses décisions avec **`-gcflags=-m`** :

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

L'inlining est aussi **transitif**, et c'est là qu'il devient intéressant : quand `main` appelle
`AddTwice`, qui appelle `add`, le compilateur inline d'abord `AddTwice` dans `main`, puis, dans cette
copie fraîchement insérée, retente l'inlining de l'appel à `add` qui s'y trouve maintenant. La sortie de
`-m` sur `main.go` montre bien les **deux** décisions séparément (`inlining call to AddTwice` puis
`inlining call to add`) : c'est cette **cascade** qui élimine entièrement la chaîne d'appels
`main -> AddTwice -> add` au profit d'un simple calcul inséré sur place.

Pour voir le **coût exact** retenu par le compilateur, et pas seulement le verdict « can inline »,
augmentez la verbosité avec **`-gcflags=-m=2`** :

```
$ go build -gcflags=-m=2 -o /dev/null ./ch39-compilation-pgo
compile.go:5:6: can inline add with cost 4 as: func(int, int) int { return a + b }
compile.go:8:6: can inline AddTwice with cost 9 as: func(int) int { return add(n, n) }
compile.go:59:6: cannot inline TotalArea: marked go:noinline
```

`add` (coût 4) et `AddTwice` (coût 9) sont loin du plafond de 80. Une fonction qui le **dépasse**
l'annonce tout aussi explicitement. Par exemple, ce corps avec boucle et plusieurs opérations :

```go
func sumSquares(n int) int {
    total := 0
    for i := 0; i < n; i++ {
        total += i*i - i/2 + i%7
        total += i*i - i/2 + i%7
        total += i*i - i/2 + i%7
        total += i*i - i/2 + i%7
        total += i*i - i/2 + i%7
    }
    return total
}
```

produit, sous `-m=2` : `cannot inline sumSquares: function too complex: cost 84 exceeds budget 80`.

Ce qui **empêche** l'inlining : corps trop gros (au-delà du budget), **boucles** coûteuses, `defer` dans
certains cas, appels récursifs, et la directive explicite **`//go:noinline`**. Dans notre code,
`TotalArea` la porte délibérément : si elle était inlinée, ses appels chauds `s.Area()` se
retrouveraient dispersés sur autant de sites d'appel différents qu'il y a d'endroits où `TotalArea` est
invoquée, diluant le profil PGO au lieu de le concentrer sur **un seul** site stable — exactement ce
qu'on veut éviter pour la démonstration de dévirtualisation plus bas.

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
les deux optimisations se renforcent. La raison est mécanique : l'escape analysis raisonne **par
fonction**. Tant qu'un appel reste un appel, le compilateur doit traiter conservativement tout ce que la
fonction appelée **pourrait** faire d'un pointeur qu'elle reçoit (le pire cas : le stocker quelque part
de durable). Une fois le corps **inséré** dans l'appelante, ce pointeur et son usage réel sont visibles
dans le **même** corps de fonction : l'analyse n'a plus besoin de supposer le pire, elle peut **suivre**
la valeur jusqu'à son dernier usage et conclure qu'elle ne sort jamais sur le tas — un raisonnement
impossible tant que la frontière d'appel existait.

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

La différence tient à ce que le compilateur peut **prouver statiquement**. Dans `SumRange`, l'indice
implicite du `for range` est **par construction** compris entre `0` et `len(xs)-1` : c'est la définition
même du `range` sur un slice, aucune analyse supplémentaire n'est nécessaire. Dans `SumGather`, `i` est
une valeur lue dans un **autre** slice (`idx`) : rien ne relie syntaxiquement sa plage de valeurs à
`len(xs)`, même si vous, en tant qu'humain, savez que les indices sont valides. Le **témoin** de
`SumHinted` comble cet écart : `_ = xs[3]` force un unique contrôle explicite, dont la passe `prove` du
compilateur déduit que `len(xs) > 3` — donc que `xs[0]`, `xs[1]` et `xs[2]` sont également sûrs. Un seul
contrôle remplace les quatre qu'un accès naïf aurait exigés.

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
  1. profiler la prod -----> cpu.prof
  2. déposer le profil ----> default.pgo  (à la racine du package main)
  3. go build  -------------> -pgo=auto (défaut) le détecte et l'applique
```

```bash
go run ./ch39-compilation-pgo profile   # écrit default.pgo (profil CPU)
go build -pgo=auto .                     # auto : utilise default.pgo s'il existe
```

`-pgo` accepte aussi un **chemin explicite** (`-pgo=chemin/vers/profil.pprof`) ; `auto` n'est qu'un
raccourci qui cherche `default.pgo` à côté du `main`. Une fois trouvé, le profil influence **tout le
graphe d'appels atteignable depuis `main`**, y compris les paquets importés — pas seulement le code du
package `main` lui-même.

Deux optimisations principales, **ciblées sur les chemins chauds** :

- **Inlining plus agressif** des fonctions chaudes. Mécanisme : le compilateur construit un **graphe
  d'appels pondéré** à partir des échantillons du profil — chaque arête appelant → appelé porte un poids
  proportionnel au temps CPU mesuré à cet endroit —, trie les sites d'appel par poids, et retient les
  plus chauds jusqu'à couvrir **99 %** du poids total cumulé : c'est ce seuil que `-d=pgodebug` affiche
  sous le nom `hot-callsite-thres-from-CDF`. Pour ces sites-là seulement, le budget d'inlining passe de
  **80** à **jusqu'à 2000** nœuds — une fonction normalement trop grosse pour être inlinée le devient, si
  et seulement si le profil prouve qu'elle est appelée depuis un chemin chaud.
- **Dévirtualisation** des appels d'interface chauds quand un **type domine** : `s.Area()` devient un
  appel direct (puis inlinable). Ici le compilateur regarde, pour **ce site d'appel précis**, quelle
  fonction concrète a accumulé le plus de poids dans le profil (le « callee » candidat) ; si cette arête
  a un poids **non nul**, elle devient candidate. On le voit avec `-d=pgodebug=2` :

```
$ go build -pgo=auto -gcflags=-d=pgodebug=2 -o /dev/null .
compile.go:62:18: PGO devirtualize considering call s.Area()
hot-callsite-thres-from-CDF=0.6968...
```

Le compilateur **lit** le profil, calcule un **seuil de chaleur**, et **considère** la dévirtualisation
de l'appel polymorphe. Les gains typiques rapportés par l'équipe Go sont de **2 à 14 %** sur des services
réels — modestes mais **gratuits** (aucun changement de code), et **cumulatifs** d'une version à l'autre.

> ⚠️ **Reproduisez l'exemple vous-même** : il arrive de voir apparaître, juste après la ligne
> `considering`, `compile.go:62:18: call main.TotalArea:3: no hot callee` — alors même que 99 % des
> formes générées sont des `Square`. Ce n'est pas un bug : le profil CPU est **statistique**
> (échantillonné par défaut à ~100 Hz), et `Area()` est si bon marché (une multiplication) qu'un run de
> quelques secondes peut n'enregistrer **aucun** échantillon précisément à l'intérieur de cet appel — la
> dominance d'un **type** dans le code n'implique pas la dominance de ce **site d'appel** dans un profil
> de _temps_. Un profil de production, avec un volume et une diversité de charge réels, donne des arêtes
> nettement mieux établies que ce micro-exemple synthétique.

> 💡 **Versionnez `default.pgo`** avec le code : le build PGO devient reproductible et la CI applique la
> même optimisation. Rafraîchissez le profil périodiquement à partir de la production.

## Architecture cible : `GOAMD64`, FMA, DWARF5

- **`GOAMD64=v1..v4`** sélectionne un **niveau d'instructions** x86-64 (v2 = SSE3+, v3 = AVX2, v4 =
  AVX-512). En **v3+**, le compilateur émet des **FMA** (_fused multiply-add_) : `a*b+c` devient une
  **seule** instruction matérielle au lieu de deux (`mul` puis `add`) — donc plus **rapide**, et plus
  **précise** : un seul arrondi final, là où deux opérations séparées arrondissent chacune leur résultat
  intermédiaire et accumulent l'erreur. _(La machine de référence du livre est **arm64** : `GOAMD64` n'y
  a pas d'effet ; le FMA y est disponible nativement.)_
- **DWARF5** (par défaut en **1.25**) : infos de debug plus **compactes**, binaires plus petits, liens
  plus rapides — transparent pour delve/gdb.

---

## 🆕 Go 1.2x

- **1.21** — **PGO** en disponibilité générale : `-pgo=auto` détecte `default.pgo`. Dévirtualisation +
  inlining guidés par profil.
- **1.25** — **DWARF5** par défaut (debug plus compact) ; **FMA** émis en **`GOAMD64=v3`** et plus.
- **1.26** — **davantage de backing stores de slices alloués sur la pile** : l'escape analysis progresse,
  réduisant les allocations sans changer le code (vérifié en [Ch. 26](26-allocation-escape.md)) ;
  **`b.Loop` n'empêche plus l'inlining** du corps de boucle ([Ch. 36](36-tests-benchmarks-fuzzing.md)).

## ⚠️ Pièges

- **Réécrire pour « forcer » l'inlining** sans mesure : le compilateur décide mieux que l'intuition.
  Lisez `-m`, ne devinez pas.
- **`//go:noinline` par superstition** : c'est un outil de **diagnostic/benchmark**, pas une
  optimisation. En production, il **dégrade** — et la directive est **absolue** : même un site jugé
  brûlant par la PGO ne sera jamais inliné si elle est présente.
- **Oublier `default.pgo`** à la racine du **package `main`** : `-pgo=auto` ne le trouve pas ailleurs.
- **Profil PGO périmé** : un profil qui ne reflète plus la charge optimise les mauvais chemins.
  Rafraîchissez-le.
- **Profil PGO trop petit ou synthétique** : un site d'appel peut ne révéler **aucune** fonction chaude
  même si un type domine numériquement (voir l'exemple `no hot callee` ci-dessus) — le profil mesure du
  **temps échantillonné**, pas des comptages d'appels. Préférez toujours un profil de production.
- **`-gcflags=-m` bruyant** sur un package qui importe beaucoup : les décisions de la bibliothèque
  standard (`os`, `fmt`, `sync/atomic`...) apparaissent mêlées aux vôtres. Filtrez avec
  `grep compile.go` ou `grep main.go` pour isoler votre code.
- **`unsafe` pour éviter un bounds check** : presque toujours prématuré. Préférez `range` ou un témoin,
  et **mesurez** ([Ch. 35](35-unsafe-cgo.md)).

## ⚡ Performance

- **L'inlining débloque** les autres passes : c'est souvent le gain le plus important, et il est
  **automatique**.
- **BCE** : le `range` est gratuit ; l'indexation externe coûte un contrôle (~15 % ici). Aidez le
  compilateur par un **accès témoin** quand la boucle l'autorise.
- **PGO** : 2-14 % « gratuits » sur du code chaud, surtout sur les appels d'interface
  ([Ch. 33](33-interfaces-profondeur.md)) et les fonctions chaudes. Le levier inlining va jusqu'à
  **×25** de budget (80 → 2000 nœuds), mais seulement sur les sites identifiés chauds par le profil.
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
   (Indice : `-gcflags=-m=2` affiche le coût exact, plus utile que le simple verdict.)
2. Récrivez `SumGather` avec un `range` sur `idx` **et** un accès témoin : le contrôle disparaît-il ?
3. Générez `default.pgo`, reconstruisez avec `-pgo=auto` puis `-pgo=off` et comparez `pgodebug`.
4. Relancez plusieurs fois `go run . profile && go build -pgo=auto -gcflags=-d=pgodebug=2 -o /dev/null .` :
   obtenez-vous parfois `no hot callee` pour `s.Area()`, parfois une callee retenue ? Augmentez la durée
   du profil dans `main.go` (`3 * time.Second`) : la stabilité du résultat s'améliore-t-elle ?

---

## 📌 À retenir

- Le compilateur passe par une **forme SSA** ; on observe ses décisions avec **`-gcflags=-m`** (inlining,
  escape) et **`-d=ssa/check_bce`** (contrôles de borne).
- **Inliner** débloque les autres optimisations ; budget ~80, `//go:noinline` pour le diagnostic
  seulement. Ne réécrivez pas « pour l'inlining ».
- **BCE** : `range` = aucun contrôle ; indexation externe = contrôle conservé (**~15 %** ici). Un accès
  **témoin** aide.
- **PGO** (🆕 1.21) : `default.pgo` + **`-pgo=auto`** → inlining agressif (budget jusqu'à **2000** sur
  les sites chauds) + **dévirtualisation** des appels chauds ; **2-14 %** gratuits, à **versionner** et
  rafraîchir avec un profil représentatif.
- **`GOAMD64=v3+`** active les **FMA** (x86) ; **DWARF5** (1.25) allège le debug ; 1.26 met **plus de
  slices sur la pile**.

## 🔁 Pour aller plus loin

- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : produire le profil qui nourrit la PGO.
- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : l'analyse d'échappement en détail.
- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : ce que la dévirtualisation supprime.
- [Ch. 40 — Méthodologie de performance](40-methodologie-performance.md) : intégrer tout cela en un processus.
- Doc : `go doc cmd/compile`, `go help build` (flag `-pgo`), `go.dev/doc/pgo`.
