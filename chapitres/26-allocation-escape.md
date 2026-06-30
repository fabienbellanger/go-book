# 26 — Allocation mémoire & escape analysis

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

En C, le choix est **explicite dans la syntaxe** : une variable locale (`int x;`) va sur la pile, un
`malloc` va sur le tas — c'est le code source lui-même qui décide, et une erreur (renvoyer l'adresse
d'une variable locale) produit un pointeur invalide silencieux. En Go, `x := Point{1, 2}` s'écrit à
l'identique dans les deux cas : c'est l'**usage** qui en est fait (renvoyée ? stockée ? juste lue ?) qui
détermine l'emplacement, pas la syntaxe de la déclaration. D'où l'intérêt de savoir **lire** la
décision plutôt que de la deviner.

---

## La pile d'une goroutine

Chaque goroutine démarre avec une **petite pile** (~2 Kio, [Ch. 19](19-goroutines.md)) qui **croît**
automatiquement : quand elle déborde, le runtime alloue une pile plus grande, **copie** le contenu et
ajuste les pointeurs. C'est transparent. Tout ce qui ne « s'échappe » pas de la fonction y est rangé,
puis **libéré gratuitement** au retour — sans GC.

Pourquoi « gratuit » au juste ? Réserver une variable sur la pile, c'est seulement **avancer** le
pointeur de pile (`SP`) à l'entrée de la fonction, et le **reculer** au retour — quelques instructions,
aucune structure à tenir à jour, aucun verrou. Une allocation tas, elle, doit choisir une classe de
taille, consulter le `mcache` (détaillé plus loin dans ce chapitre) et rester **traçable** par le GC
tant qu'elle vit. C'est cette différence de comptabilité, pas la vitesse de la RAM, qui rend la pile
« gratuite » et le tas « coûteux ».

La **copie** lors d'une croissance de pile n'est possible que parce que le runtime Go possède une
information que le système d'exploitation n'a pas : il **connaît** chaque pointeur qui pointe vers
cette pile (le compilateur les a tous recensés à la compilation) et peut donc tous les **réécrire**
après déplacement ([Ch. 19](19-goroutines.md) détaille pourquoi cette propriété est impossible pour la
pile d'un thread OS). C'est l'inverse du tas : un objet tas, lui, ne **bouge jamais** une fois alloué
(le GC de Go est non compactant, [Ch. 27](27-garbage-collector.md)) — il n'a pas besoin de bouger
puisqu'il n'est pas redimensionné comme une pile. Par sécurité, cette croissance n'est pas infinie : au
delà d'un plafond (**1 Go** par défaut sur 64 bits, 250 Mo sur 32 bits, réglable via
`debug.SetMaxStack`), le runtime arrête le programme (`fatal error: stack overflow`) plutôt que de
laisser une récursion incontrôlée épuiser la mémoire.

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
Si oui, la valeur « **s'échappe** » et part sur le tas. C'est une analyse **statique et conservatrice** :
si le compilateur ne peut pas **prouver** qu'une valeur reste locale, il la fait échapper par prudence
(mieux vaut une allocation superflue qu'un pointeur invalide). Les déclencheurs classiques, et le
pourquoi de chacun :

- **renvoyer un pointeur** vers une variable locale — l'appelant doit pouvoir le déréférencer après le
  retour de la fonction, donc la variable ne peut pas mourir avec sa frame de pile ;
- **stocker un pointeur** dans une structure qui survit (champ, slice, map, canal) — sa durée de vie
  devient celle du conteneur, potentiellement bien plus longue que l'appel qui l'a écrit ;
- **capturer une variable dans une closure qui s'échappe elle-même** (renvoyée, stockée, ou passée à
  `go`) — la variable doit vivre aussi longtemps que la closure qui la référence (🔁 [Ch.
  15](15-closures.md), qui détaille la mécanique de capture) ;
- passer une valeur à une **interface** dont le receveur la retient au-delà de l'appel (« boxing ») —
  ce n'est pas la conversion en interface en elle-même qui coûte, c'est ce que le receveur **fait** de
  la valeur ensuite (exemple ci-dessous) ;
- une taille **inconnue à la compilation** (`make([]T, n)` renvoyé) — pour rester sur la pile, une
  variable a besoin d'une taille **fixée à la compilation** ; une taille qui dépend d'une variable
  d'exécution interdit ce calcul ;
- une valeur **trop grande**, même purement locale — passé un certain volume, le compilateur préfère
  l'allouer sur le tas plutôt que de gonfler la frame de pile et le coût de copie à chaque croissance de
  pile (mesuré : un tableau `[16384]int` de 128 Kio reste sur la pile, `[16385]int` (128 Kio + 8 o)
  bascule sur le tas — un seuil interne, non documenté ni garanti stable entre versions).

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

Le cas de l'interface mérite un exemple à part : il **surprend**, car on l'associe rarement à une fuite
quand la valeur est petite et passée **par valeur** (pas par pointeur).

```go
// code/ch26-allocation-escape/alloc.go
var sink any // simule un état qui survit à l'appel (cache, registre global, etc.)

func pointToInterface(p Point) {
	sink = p // p (16 o, par valeur) boxé puis retenu par sink -> TAS, 1 alloc
}
```

`Point` ne fait que 16 octets et `pointToInterface` le reçoit **par valeur**, sans aucun pointeur en
vue — et pourtant `p` s'échappe. La cause profonde est **la même** que pour `NewPoint` : sa durée de
vie dépasse l'appel, parce que `sink` la conserve. Seul le mécanisme diffère : un pointeur renvoyé d'un
côté, une interface retenue de l'autre. C'est le principe à retenir plutôt qu'une liste de cas
particuliers à mémoriser.

> 💡 Techniquement, la conversion en interface **seule** ne force rien : si la fonction qui reçoit la
> valeur ne fait que la **lire** sans la conserver, l'escape analysis peut prouver qu'elle reste sur la
> pile (essayez de remplacer `sink = p` par `_ = p` dans le fichier réel : l'échappement disparaît sous
> `-m`). En pratique, dès que le receveur est un autre paquet dont le compilateur ne peut pas garantir
> l'absence de rétention (`fmt`, `log`, `encoding/json`...) ou qu'il stocke effectivement la valeur,
> l'échappement est la règle — d'où le piège, redoutable parce qu'**invisible à la lecture** du site
> d'appel.

## Lire les décisions : `-gcflags=-m`

Le compilateur **explique** chaque choix avec `-m` :

```
$ go build -gcflags=-m ./ch26-allocation-escape
alloc.go:31:2:  moved to heap: p                      <- NewPoint : échappe
alloc.go:48:9:  p escapes to heap                     <- pointToInterface : boxé, retenu par sink
alloc.go:57:11: make([]int, 8) does not escape        <- sumSmallSlice : PILE
alloc.go:72:11: make([]int, n) escapes to heap        <- LeakSlice : renvoyé
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

| Fonction           | alloc/op | Où ?                                         |
| ------------------ | -------- | -------------------------------------------- |
| `sumLocalArray`    | **0**    | tableau local sur la pile                    |
| `sumSmallSlice`    | **0**    | backing de slice **sur la pile** (1.25/1.26) |
| `NewPoint`         | **1**    | pointeur renvoyé → tas                       |
| `LeakSlice`        | **1**    | slice renvoyé → tas                          |
| `pointToInterface` | **1**    | boxé dans une interface **retenue** → tas    |

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

Pourquoi des **classes de taille** plutôt qu'une réservation exacte ? Un allocateur à tailles exactes
doit gérer des blocs libres de toutes formes (fragmentation **externe** : assez de mémoire libre au
total, mais aucun trou assez grand et contigu) et chercher dans des listes chaînées pour en trouver un
qui convient. En bornant les tailles possibles à ~68 classes fixes, chaque taille a sa propre liste de
blocs **libres et interchangeables** : allouer ou libérer redevient une opération de liste quasi
gratuite. Le prix payé est la fragmentation **interne** illustrée ci-dessus (17 o facturés 24 o) — un
compromis volontaire, et largement plus rentable en pratique que la fragmentation externe qu'il évite.

> ⚠️ Ne confondez pas les **deux** seuils de taille rencontrés dans ce chapitre : les ~128 Kio évoqués
> plus haut concernent une variable qui **resterait sur la pile** sans cette limite (un choix de
> l'escape analysis, à la compilation) ; les 32 Kio ci-dessus délimitent, **côté tas**, les objets
> servis par `mcache`/`mcentral` (via les size classes) des « gros objets » alloués directement par
> `mheap`. Deux mécanismes indépendants, deux chiffres sans rapport entre eux.

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
le résultat. `make([]T, 0, n)` suppose un slice construit **de zéro** ; pour étendre un slice
**existant** (déjà partiellement rempli), `slices.Grow(s, n)` ([Ch. 30](30-slices-profondeur.md)) joue
le même rôle sans le recréer. Pour les objets **réutilisables** et de courte vie, `sync.Pool`
([Ch. 21](21-synchronisation.md)) recycle plutôt que réallouer.

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
- **Confondre « sur la pile » et « gratuit en tout point »** — la pile évite l'allocation et le GC, mais
  pas le **coût de copie** : passer (ou renvoyer) un **gros struct par valeur** dans une fonction
  appelée souvent reste un travail CPU et cache proportionnel à sa taille, même à **0 alloc/op**. `-m`
  ne le signale pas (ce n'est pas une fuite) ; seul un benchmark le révèle. C'est un arbitrage distinct
  de l'escape analysis : un pointeur supprime la copie mais peut introduire une allocation tas — les
  deux coûts se mesurent, ne se devinent pas.
- **Interfaces sur le chemin chaud** — convertir une valeur en `any`/interface peut **boxer** (allouer)
  dès que le receveur la retient, **même pour une valeur minuscule passée par valeur** (voir
  `pointToInterface` plus haut — le piège est d'autant plus sournois qu'il est invisible au site
  d'appel). `-gcflags=-m` le révèle.
- **`append` sans `cap`** dans une boucle — réallocations répétées. Préallouez dès que la taille est
  connue ou estimable.
- **Micro-optimiser à l'aveugle** — `-m` et `-benchmem` **d'abord** ; n'allez pas contre l'escape
  analysis sans preuve qu'il y a un gain.

## ⚡ Performance

- Une alloc tas ne coûte pas que l'alloc : elle **augmente le travail du GC** ([Ch. 27](27-garbage-collector.md)).
  Réduire `allocs/op` réduit **deux** coûts à la fois.
- L'allocation sur le `mcache` est **sans verrou** : le vrai ennemi n'est pas le coût unitaire mais le
  **nombre** d'objets (pression GC) et leur **taille** (size class).
- **`B/op` reflète la classe de taille, pas la taille logique demandée** : un `Point` de 16 o et un
  `NewPoint` qui en alloue un finissent dans la **même** classe que `pointToInterface` (16 o exactement
  ici, donc pas d'arrondi visible) — mais un struct de 17 o, lui, facture 24 o (`B/op`). Comparez
  toujours `B/op` mesuré à la taille réelle de votre type (`unsafe.Sizeof`), pas à vue d'œil.
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
2. Dans `pointToInterface`, remplacez `sink = p` par `_ = p` : l'échappement disparaît-il sous `-m` ?
   Cela confirme que c'est la **rétention**, pas la conversion en interface, qui coûte.
3. Estimez la `cap` initiale idéale de `concatNoPrealloc` et mesurez le gain avec `-benchmem`.
4. Augmentez la taille de `buf` dans `sumLocalArray` jusqu'à dépasser ~128 Kio : à quel moment `-m`
   bascule-t-il de `does not escape` à `moved to heap` sur votre machine ?

---

## 📌 À retenir

- **Pile** = gratuit, par goroutine, libéré au retour ; **tas** = alloc + pression GC. L'**escape
  analysis** (compilation) décide, pas vous — et tranche par prudence dès qu'elle ne peut pas
  **prouver** qu'une valeur reste locale.
- Une valeur **s'échappe** si sa durée de vie dépasse la fonction : pointeur renvoyé/stocké, closure
  échappée, interface **retenue**, taille dynamique ou trop grande. Peu importe le mécanisme, le
  principe est **unique** : la durée de vie dépasse l'appel. **`-gcflags=-m`** explique chaque choix.
- Une interface peut faire fuir une valeur **minuscule passée par valeur** — ce n'est pas la conversion
  qui coûte, c'est la **rétention** par le receveur (`pointToInterface`).
- **Mesurez** avec `allocs/op` (`testing.AllocsPerRun`, `-benchmem`) — et faites-en des **tests** de non-régression.
- L'allocateur a **trois étages** : `mcache` (par P, sans verrou) → `mcentral` → `mheap` → OS ; tas en
  **spans** par **size class** (pour échanger fragmentation interne contre listes libres O(1)) ;
  **tiny allocator** pour les objets < 16 o sans pointeur.
- **Préallouez** (`make([]T, 0, n)`, `slices.Grow`), réutilisez (`sync.Pool`) : moins d'allocations =
  plus rapide **et** moins de GC.

## 🔁 Pour aller plus loin

- [Ch. 15 — Closures](15-closures.md) : capture par référence, et pourquoi une closure échappée
  embarque ses variables capturées sur le tas.
- [Ch. 27 — Garbage collector](27-garbage-collector.md) : ce que coûtent les objets du tas.
- [Ch. 28 — L'ordonnanceur](28-ordonnanceur-gmp.md) : le P et son `mcache`.
- [Ch. 30 — Slices en profondeur](30-slices-profondeur.md) : croissance d'`append`, `slices.Grow`, `slices.Clip`.
- [Ch. 39 — Compilation & inlining](39-compilation-inlining-pgo.md) : escape analysis et inlining ensemble.
- Doc : `go doc testing.AllocsPerRun` ; `go build -gcflags=-m` sur votre propre code.
