# 15 — Fonctions anonymes & closures

> **Objectif** — Écrire des fonctions littérales, comprendre la **capture par référence**, la
> durée de vie des variables capturées, et appliquer les grands patterns à base de closures
> (décorateur, middleware, mémoïsation, _functional options_).
>
> **Prérequis** — [Ch. 5 — Fonctions](05-fonctions.md), [Ch. 4 — Flux de contrôle](04-flux-controle.md) (boucles `for`)

---

## Introduction

En Go, **les fonctions sont des valeurs** ([Ch. 5](05-fonctions.md)) : on les affecte, on les passe
en argument, on les renvoie. Une **fonction anonyme** (ou _littérale_) est une fonction écrite sur
place, sans nom. Quand elle **capture** des variables de la fonction englobante, elle devient une
**closure** : une fonction **+** l'environnement auquel elle continue d'accéder.

C'est le mécanisme derrière beaucoup d'idiomes Go : callbacks, décorateurs, middlewares,
configurateurs. L'exemple est dans [`code/ch15-closures/`](../code/ch15-closures/).

---

## Fonction anonyme

Une fonction littérale a le même corps qu'une fonction nommée, mais s'écrit comme une expression.
On peut l'appeler immédiatement, la stocker, ou la passer :

```go
greet := func(name string) string { return "Bonjour, " + name }
fmt.Println(greet("Go"))

// Appel immédiat (IIFE) : utile pour isoler une portée.
n := func() int { return 40 + 2 }()
```

Tant qu'elle ne référence rien de l'extérieur, ce n'est qu'une fonction sans nom. Elle devient
intéressante dès qu'elle **capture**.

## Capture par référence

Une closure capture les variables **par référence**, pas par copie : elle voit et modifie **la
même** variable que la fonction englobante. C'est ce qui permet une closure **à état** :

```go
// counter renvoie une closure dont l'état (n) survit entre les appels.
func counter() func() int {
	n := 0
	return func() int {
		n++ // modifie LA variable n, partagée d'un appel à l'autre
		return n
	}
}

c := counter()
fmt.Println(c(), c(), c()) // 1 2 3

// Chaque appel à counter() crée un NOUVEL n : les compteurs sont indépendants.
c2 := counter()
fmt.Println(c2()) // 1
```

> 💡 « Par référence » ne veut pas dire « pointeur explicite » : c'est le **compilateur** qui s'en
> charge. Si une variable locale est capturée par une closure qui lui survit, il la déplace sur le
> **tas** (voir le schéma plus bas).

Ce choix n'est pas arbitraire. Si Go capturait **par valeur** — une copie de `n` figée au moment de
la création de la closure — `counter` serait inutilisable : chaque appel travaillerait sur sa propre
copie repartant de 0, et aucun état ne pourrait s'accumuler d'un appel à l'autre. La capture par
référence est précisément ce qui permet à une closure de **partager** un état mutable avec sa fonction
englobante, ou avec d'autres closures qui capturent la même variable. Contrairement à C++, où chaque
lambda choisit explicitement `[=]` (capture par valeur) ou `[&]` (capture par référence), Go ne laisse
pas le choix : **toute** capture est par référence. Pour obtenir l'équivalent d'une copie, il faut la
faire soi-même — c'est tout l'objet de l'idiome `i := i` détaillé plus bas.

## Durée de vie & échappement sur le tas

Normalement, une variable locale meurt à la fin de sa fonction. Mais si une closure renvoyée y
accède encore, elle **ne peut pas** mourir : le compilateur la **promeut sur le tas** (_escape_).
L'**escape analysis** ([Ch. 26](26-allocation-escape.md)) décide cela à la compilation.

```
  func counter() func() int {
      n := 0                 n est capturée par la closure renvoyée :
      return func() int {    elle doit survivre à counter().
          n++                => le compilateur la PROMEUT sur le tas.
          return n
      }
  }

  PILE de counter (libérée au retour)        TAS (survit après le retour)
  +-----------------------+                  +---------+
  | (rien : n a échappé)  |                  |  n = 0  | <--+
  +-----------------------+                  +---------+    |
                                                            | capture
  closure renvoyée à l'appelant                             |
  +-----------------+                                       |
  |  code  |  *n ---|---------------------------------------+
  +-----------------+
```

On le vérifie avec `-gcflags=-m` :

```
$ go build -gcflags='-m' -o /dev/null ./ch15-closures
... moved to heap: n
... func literal escapes to heap
```

Ces deux lignes décrivent deux allocations **distinctes**. La valeur d'une closure (ce que contient
une variable de type `func() int`) n'est pas le code lui-même : c'est un **pointeur** vers une petite
structure interne au runtime (parfois appelée _funcval_) qui regroupe l'adresse du code et un
pointeur vers chaque variable capturée. `moved to heap: n` signale que la variable `n` elle-même est
promue sur le tas ; `func literal escapes to heap` signale que cette structure funcval l'est aussi.
Une fonction littérale qui ne capture rien n'a besoin ni de l'une ni de l'autre.

## Portée par itération (le piège historique, disparu en 1.22)

Avant Go 1.22, **toutes** les itérations d'une boucle partageaient **la même** variable de boucle.
Une closure capturant `i` voyait donc sa **valeur finale**, d'où le bug classique :

```go
// AVANT 1.22 : les trois closures partagent la même i.
var fns []func() int
for i := 0; i < 3; i++ {
	fns = append(fns, func() int { return i })
}
// fns[0]() == fns[1]() == fns[2]() == 3   (et non 0, 1, 2 !)
```

```
  AVANT 1.22 : une seule variable i, réutilisée          DEPUIS 1.22 : une variable par tour,
  partagée par les 3 closures                            indépendante pour chaque closure

  tour 0 : i=0 --+                                        tour 0 : i0=0  -> closure 0 capture &i0
  tour 1 : i=1   |--> closures 0,1,2 capturent TOUTES &i   tour 1 : i1=1  -> closure 1 capture &i1
  tour 2 : i=2 --+    (la MÊME adresse mémoire)            tour 2 : i2=2  -> closure 2 capture &i2

  fin de boucle : i vaut 3                                 chaque ik garde SA valeur, jamais
  les 3 closures lisent la valeur FINALE : 3, 3, 3          réécrite par le tour suivant : 0, 1, 2
```

> ⚠️ Avant 1.22, l'idiome correctif consistait à **recopier** la variable de boucle à chaque
> itération, en la masquant par une nouvelle variable locale de même nom :
>
> ```go
> for i := 0; i < 3; i++ {
> 	i := i // copie FRAÎCHE de i, locale à cette itération, capturée séparément
> 	fns = append(fns, func() int { return i })
> }
> // fns[0]() == 0, fns[1]() == 1, fns[2]() == 2
> ```
>
> Ce `i := i` crée, à chaque tour, une nouvelle variable qui meurt à la fin de l'itération — sauf
> si une closure la capture, auquel cas elle échappe sur le tas comme toute variable capturée (cf.
> section précédente). Depuis 1.22, le compilateur fait exactement cela **pour vous**, implicitement.

Depuis **Go 1.22**, chaque itération a **sa propre** variable de boucle. Le même code donne
désormais `0, 1, 2` — le piège a **disparu** :

```go
// code/ch15-closures/closures.go
func makeAdders() []func() int {
	var fns []func() int
	for i := range 3 {
		fns = append(fns, func() int { return i })
	}
	return fns // en 1.22+ : renvoie des closures qui valent 0, 1, 2
}
```

Vérifié sur go1.26.4 : `[0 1 2]`. Le même programme compilé avec `go 1.21` dans son `go.mod` donne
`[3 3 3]`. Cela valait aussi pour le piège le plus courant — lancer des goroutines dans une boucle :

```go
for i := range 3 {
	go func() { fmt.Println(i) }() // 1.22+ : 0,1,2 (ordre quelconque). Avant : souvent 3,3,3.
}
```

Avant 1.22, une seconde correction, équivalente à `i := i` mais plus visible, consistait à **passer
`i` en paramètre** de la fonction lancée : `go func(i int) { fmt.Println(i) }(i)`. L'argument est
**évalué et copié immédiatement**, au moment de l'appel — indépendamment de toute capture (🔁 [Ch.
19](19-goroutines.md) détaille ce même piège côté goroutines, ainsi que la fuite de goroutine quand
une closure lancée reste bloquée pour toujours).

> ⚠️ Ce changement est **silencieux** et dépend de la **version `go` du `go.mod`**. Un module
> déclarant `go 1.21` garde l'ancienne sémantique même compilé avec Go 1.26 — la compatibilité
> ascendante est préservée.

## Patterns à base de closures

### Décorateur

Envelopper une fonction pour lui **ajouter un comportement** sans la modifier :

```go
func logged(name string, fn func(int) int) func(int) int {
	return func(x int) int {
		out := fn(x)
		fmt.Printf("[trace] %s(%d) = %d\n", name, x, out)
		return out
	}
}

double := logged("double", func(x int) int { return x * 2 })
double(21) // calcule 42 et trace l'appel
```

### Mémoïsation

Capturer un **cache** pour éviter de recalculer :

```go
func memoize(fn func(int) int) func(int) int {
	cache := map[int]int{}
	return func(x int) int {
		if v, ok := cache[x]; ok {
			return v // déjà calculé
		}
		v := fn(x)
		cache[x] = v
		return v
	}
}
```

> ⚠️ **Cas limite : mémoïser une fonction récursive.** Une closure ne peut pas se référencer
> **elle-même** quand elle est déclarée avec `:=` : son propre nom n'existe pas encore pendant
> l'évaluation de son corps. Mémoïser un `fib` récursif demande donc de **séparer la déclaration
> de l'affectation** :
>
> ```go
> var fib func(int) int // déclarée d'abord, avec sa valeur zéro (nil)
> fib = memoize(func(n int) int {
> 	if n < 2 {
> 		return n
> 	}
> 	return fib(n-1) + fib(n-2) // capture fib : la variable existe déjà
> })
> ```
>
> `fib := memoize(func(n int) int { ...fib(n-1)... })` ne compile **pas** (`undefined: fib`) : avec
> `:=`, `fib` n'entre en portée qu'**après** l'évaluation complète du membre de droite, donc le corps
> de la fonction anonyme ne peut pas encore la voir.

### Middleware (chaînage)

Un middleware est une closure qui **enveloppe** un handler et se **compose** avec d'autres — le cœur
de `net/http` (projet 2) :

```go
type Handler func(req string) string
type Middleware func(Handler) Handler

h := chain(baseHandler, tagged("api"), upper) // tagged enveloppe upper enveloppe base
h("go") // "api:HELLO GO"
```

> 💡 `chain` parcourt les middlewares **à l'envers** (`mws[i](h)` en partant de la fin) : c'est ce qui
> fait de `tagged("api")` l'enveloppe la plus **externe** et de `upper` la plus **interne**. Mais
> l'**exécution**, elle, part du handler de base et **remonte** — l'ordre de construction et l'ordre
> d'exécution sont inverses :
>
> ```
>   construction (chain)                 exécution (appel de h("go"))
>   tagged("api")  <- le plus externe     1. baseHandler("go")  -> "hello go"
>      upper                              2. upper(...)         -> "HELLO GO"
>         baseHandler <- le plus interne  3. tagged("api")(...) -> "api:HELLO GO"
> ```
>
> Chaque middleware **doit appeler `next`** pour que la chaîne se poursuive ; celui qui s'en abstient
> **court-circuite** les suivants — utile pour une authentification qui refuse une requête sans
> jamais atteindre le handler de base.

### Functional options

Des closures qui **configurent** un objet à la construction — l'idiome compense l'absence, en Go, de
paramètres par défaut et de surcharge de fonctions ([Ch. 5](05-fonctions.md) détaille le compromis
face à un simple `struct` de configuration). L'API reste stable même quand on ajoute des champs :

```go
type Option func(*Server)

func WithPort(p int) Option { return func(s *Server) { s.port = p } } // capture p

srv := NewServer("localhost", WithPort(9090)) // le reste prend ses défauts
```

Chaque `WithXxx(...)` renvoie une closure qui capture son argument ; comme elle est passée à
`NewServer` puis appelée à l'intérieur de la boucle `for _, opt := range opts`, elle **échappe** sur
le tas (cf. plus haut) — une option « coûte » donc une allocation. Pour un constructeur appelé en
boucle chaude avec un grand nombre d'options, préférez un `struct` de configuration ([Ch. 5](05-fonctions.md)
compare les deux approches).

---

## 🆕 Go 1.2x

- **1.22** — **portée par itération** de la variable de boucle (`for`). Le piège historique
  `for … { go func(){ use(i) } }` **n'existe plus** ; le comportement dépend de la version `go` du
  `go.mod` (compatibilité préservée pour les anciens modules).
- **1.22** — `for range N` (boucle sur un entier, [Ch. 4](04-flux-controle.md)), pratique pour
  fabriquer des closures indexées.

## ⚠️ Pièges

- **Capturer la variable, pas la valeur** : une closure différée voit la valeur **au moment de
  l'exécution**, pas de la capture. C'est l'inverse des arguments de `defer` ([Ch. 16](16-defer.md)),
  qui sont évalués **immédiatement**, à l'enregistrement :

  ```go
  n := 0
  defer func() { fmt.Println("closure:", n) }() // lit n à la FIN de la fonction
  defer fmt.Println("valeur immédiate:", n)      // lit n MAINTENANT (figé)
  n++
  n++
  // au retour (ordre LIFO) : "valeur immédiate: 0" puis "closure: 2"
  ```

- **Supposer l'ancienne sémantique de boucle** sous Go ≥ 1.22 (ou l'inverse dans un vieux module) :
  vérifiez la ligne `go` du `go.mod`.
- **Référencer une closure récursive avant sa déclaration** : `fib := func(n int) int { ...fib...}`
  ne compile pas (`undefined: fib`) — il faut un `var` séparé (cas détaillé dans la mémoïsation
  ci-dessus).
- **Cache non borné** dans une mémoïsation : il grandit indéfiniment. En concurrence, une `map`
  capturée n'est **pas** sûre sans `sync.Mutex` ([Ch. 21](21-synchronisation.md)).
- **Fuite mémoire par capture** : une closure qui capture un gros objet le **maintient en vie** tant
  qu'elle existe — par exemple un handler enregistré une fois pour toute la durée du programme
  (routeur HTTP, callback global) qui capture par erreur toute une `struct` de requête au lieu du
  seul champ utile : cette `struct` ne sera jamais collectée. Ne capturez que ce qui est nécessaire.

## ⚡ Performance

- Une closure qui **échappe** force la variable capturée sur le **tas** (1 allocation). Vérifiable
  avec `-gcflags=-m` (`moved to heap`). Si la closure **ne s'échappe pas** (appelée puis jetée sur
  place), le compilateur peut tout garder sur la **pile** : coût quasi nul.
- Appeler une closure stockée dans une variable est un appel **indirect** (par pointeur de fonction),
  rarement _inliné_ — un poil plus cher qu'un appel direct, négligeable hors boucle très chaude.
- 🔁 Détails pile/tas et escape analysis au [Ch. 26](26-allocation-escape.md).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch15-closures
go test ./ch15-closures/...
# Voir quelles variables capturées échappent sur le tas :
go build -gcflags='-m' -o /dev/null ./ch15-closures 2>&1 | grep -E "heap|escapes"
```

À essayer :

1. Réécrivez `makeAdders` avec `go 1.21` dans un `go.mod` à part : observez `[3 3 3]`.
2. Ajoutez un middleware `timed` qui mesure la durée d'un handler.
3. Rendez `memoize` sûr en concurrence avec un `sync.Mutex` ([Ch. 21](21-synchronisation.md)).
4. Mémoïsez un `fib` récursif en suivant le patron `var fib func(int) int ; fib = memoize(...)`.
5. Écrivez un benchmark comparant un appel direct `fn(x)` à un appel via une closure stockée dans
   une variable : mesurez le surcoût de l'indirection ([Ch. 36](36-tests-benchmarks-fuzzing.md)
   détaille `go test -bench`).

---

## 📌 À retenir

- Une **closure** = une fonction **+** les variables qu'elle **capture par référence** (la même
  variable, pas une copie).
- Une variable capturée par une closure qui lui survit **échappe sur le tas** ; l'escape analysis le
  décide à la compilation.
- **Depuis 1.22**, chaque itération a sa propre variable de boucle : le vieux piège de capture dans
  une boucle a disparu (selon la version `go` du `go.mod`).
- Closures = base du **décorateur**, du **middleware**, de la **mémoïsation** et des **functional
  options**.
- Ne capturez que le nécessaire : une closure prolonge la **durée de vie** de ce qu'elle référence.

## 🔁 Pour aller plus loin

- [Ch. 16 — `defer`](16-defer.md) : les arguments d'un `defer` sont évalués **à l'enregistrement**,
  à l'inverse de la capture par référence d'une closure.
- [Ch. 18 — Itérateurs](18-iterateurs.md) : les itérateurs `range`-over-func sont des closures qui
  reçoivent un `yield`.
- [Ch. 26 — Allocation & escape analysis](26-allocation-escape.md) : quand une capture coûte une
  allocation.
- [Ch. 19 — Goroutines](19-goroutines.md) : capturer une variable de boucle dans une goroutine,
  avant/après 1.22.
