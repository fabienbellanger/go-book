# 5 — Fonctions

> **Objectif** — Écrire et composer des fonctions : retours multiples, variadiques,
> fonctions comme valeurs, et comprendre le passage **par valeur**.
>
> **Prérequis** — [Ch. 4 — Flux de contrôle](04-flux-controle.md)

---

## Introduction

Les fonctions sont au cœur de Go : ce sont des **valeurs de première classe** (on les
passe en argument, on les renvoie, on les stocke). Deux traits les distinguent d'autres
langages : les **retours multiples** (qui fondent la gestion d'erreur idiomatique) et le
fait que tout est **passé par valeur**.

L'exemple complet est dans [`code/ch05-functions/`](../code/ch05-functions/).

## Signature & déclaration

```go
func name(param1 Type1, param2 Type2) ReturnType {
	// corps
}
```

- Le **type suit l'identifiant** (`a int`, pas `int a`).
- Des paramètres consécutifs de même type se factorisent : `func add(a, b int) int`.
- Sans valeur de retour, on omet la partie droite : `func log(msg string) { … }`.

## Retours multiples

Une fonction peut renvoyer **plusieurs valeurs** — pas besoin de structure ou de
paramètre de sortie :

```go
func divmod(a, b int) (int, int) {
	return a / b, a % b
}

q, r := divmod(17, 5) // q=3, r=2
```

C'est la base de deux idiomes omniprésents :

```go
v, ok := m[key]   // map : valeur + présence
n, err := f()     // résultat + erreur
```

## Retours nommés

On peut **nommer** les valeurs de retour dans la signature. Elles sont alors déclarées
(à leur zero value) et un `return` « nu » les renvoie :

```go
func minMax(nums ...int) (lo, hi int) {
	if len(nums) == 0 {
		return // renvoie lo=0, hi=0
	}
	lo, hi = nums[0], nums[0]
	// …
	return lo, hi // explicite (recommandé)
}
```

Avantages : ils **documentent** la signature et permettent à un `defer` de modifier le
résultat (utile pour la gestion d'erreur, voir [Ch. 16](16-defer.md)).

> ⚠️ Le `return` **nu** nuit à la lisibilité dans une fonction longue (on ne voit pas ce
> qui est renvoyé). Réservez-le aux fonctions courtes ; sinon, renvoyez explicitement.

## Fonctions variadiques

Le **dernier** paramètre peut être variadique (`...T`) : il accepte 0, 1 ou N valeurs,
reçues comme un `[]T` :

```go
func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

sum()           // 0
sum(1, 2, 3)    // 6
xs := []int{4, 5, 6}
sum(xs...)      // 15  -- « éclatement » d'un slice avec ...
```

> 💡 `fmt.Println(args ...any)` est variadique : c'est pourquoi il accepte un nombre
> quelconque d'arguments de types variés.

## Fonctions = valeurs de première classe

Une fonction est une **valeur** : elle a un type (`func(int) int`), peut être stockée,
passée en argument ou renvoyée.

```go
func apply(nums []int, f func(int) int) []int {
	out := make([]int, len(nums))
	for i, n := range nums {
		out[i] = f(n)
	}
	return out
}

squared := apply([]int{1, 2, 3, 4}, func(n int) int { return n * n })
// squared == [1 4 9 16]
```

Ici `func(n int) int { … }` est une **fonction anonyme** (littéral de fonction). Quand
elle capture des variables de son environnement, on parle de **closure** — détaillé au
[Ch. 15](15-closures.md).

> ⚠️ La zero value d'un type `func` est `nil`. **Appeler une fonction `nil` panique.**
> Testez `if f != nil` si la fonction est optionnelle.

## Récursivité

Go autorise la récursivité, sans mot-clé particulier :

```go
func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n-1)
}
```

La pile d'une goroutine **croît à la demande** (voir Ch. 19/26), donc la récursion
profonde est moins risquée qu'en C — mais pas infinie. Go n'optimise **pas** la récursion
terminale (_tail call_) : pour de très grandes profondeurs, préférez une boucle.

## Tout est passé par valeur

> **Règle d'or** — En Go, un argument est **toujours copié**. Une fonction reçoit une
> copie de la valeur, jamais la variable de l'appelant.

```go
type counter struct{ n int }

func incVal(c counter)  { c.n++ } // modifie une COPIE -> aucun effet dehors
func incPtr(c *counter) { c.n++ } // modifie via un POINTEUR -> effet visible
```

```go
c := counter{n: 0}
incVal(c)  // c.n vaut toujours 0
incPtr(&c) // c.n vaut maintenant 1
```

Pour **modifier** la valeur de l'appelant (ou éviter de copier un gros struct), on passe
un **pointeur** (`&c` à l'appel, `*counter` en paramètre).

### Le cas des slices, maps et channels

Ces types sont **eux aussi** passés par valeur — mais leur valeur est un petit
**descripteur** (header) qui pointe vers des données partagées. Copier le descripteur
**partage** donc le tableau / la table sous-jacents :

```go
func scale(nums []int, factor int) {
	for i := range nums {
		nums[i] *= factor // visible par l'appelant : même tableau sous-jacent
	}
}

nums := []int{1, 2, 3}
scale(nums, 10) // nums == [10 20 30]
```

```
   Appel scale(nums, 10)
   ---------------------
   appelant  nums ----+        header copié (ptr/len/cap),
                      |        mais MÊME tableau pointé
   fonction  nums ----+---> [ 1 | 2 | 3 ]  --(scale)-->  [ 10 | 20 | 30 ]
```

> ⚠️ Modifier un **élément** (`nums[i] = …`) est visible ; faire grandir le slice avec
> `append` ne l'est **pas** forcément (le header local change, pas celui de l'appelant).
> Détail au [Ch. 6](06-arrays-slices.md).

## La convention d'erreur

Par convention, l'**erreur est la dernière valeur de retour**, et l'appelant la teste
immédiatement :

```go
func safeDivide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division par zéro")
	}
	return a / b, nil
}

q, err := safeDivide(10, 0)
if err != nil {
	// traiter l'erreur
}
```

> 🔁 Le modèle d'erreur complet (`%w`, `errors.Is`/`As`, sentinelles) est au
> [Ch. 10](10-erreurs.md).

## 💡 Idiome : functional options

Pour une fonction de construction avec beaucoup de réglages optionnels, l'idiome Go est
le **functional options pattern** : une fonction variadique d'« options », chacune étant
une fonction qui modifie l'objet.

```go
type Option func(*Server)

func WithPort(p int) Option { return func(s *Server) { s.port = p } }

func NewServer(opts ...Option) *Server {
	s := &Server{host: "localhost", port: 8080} // défauts
	for _, opt := range opts {
		opt(s)
	}
	return s
}

s := NewServer(WithPort(9000), WithTLS()) // on ne précise que ce qu'on change
```

C'est extensible (ajouter une option ne casse pas les appels existants) et lisible.
Les options **capturent** leur argument : ce sont des closures ([Ch. 15](15-closures.md)).

## ⚡ Performance

- Un appel de fonction a un **coût** (mise en place de la pile, copie des arguments),
  généralement négligeable.
- Le compilateur **inline** automatiquement les petites fonctions (il recopie leur corps
  sur le site d'appel, supprimant le coût d'appel). On inspecte ses décisions avec
  `go build -gcflags=-m`.
- **Ne micro-optimisez pas à l'aveugle** : écrivez des fonctions claires, mesurez ensuite
  (voir Ch. 36/40). L'inlining et l'_escape analysis_ sont détaillés au
  [Ch. 39](39-compilation-inlining-pgo.md).
- Passer un **pointeur** évite de copier un gros struct, mais peut le faire **fuir sur le
  tas** (escape, Ch. 26). Ce n'est pas toujours un gain : à mesurer.

## ⚠️ Pièges

- **Oublier de tester `err`** — le compilateur ne force pas à lire un retour ; `go vet` et
  les linters aident (`errcheck`).
- **Appeler une fonction `nil`** — panique. Vérifiez les callbacks optionnels.
- **Gros struct copié** — passé par valeur, il est entièrement recopié à chaque appel ;
  utilisez un pointeur si la taille le justifie.
- **`return` nu dans une longue fonction** — illisible ; renvoyez explicitement.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch05-functions
go test ./ch05-functions/...
go build -gcflags=-m ./ch05-functions 2>&1 | grep inlin # décisions d'inlining
```

À essayer :

1. Ajoutez une option `WithHost` à `NewServer` et vérifiez que les anciens appels
   compilent toujours (extensibilité).
2. Transformez `incVal` pour qu'elle modifie réellement le counter (passez `*counter`).
3. Écrivez `compose(f, g func(int) int) func(int) int` qui renvoie `x -> f(g(x))`.

---

## 📌 À retenir

- Les fonctions sont des **valeurs de première classe** (paramètres, retours, variables).
- **Retours multiples** : fondent les idiomes `v, ok` et `v, err`.
- **Variadique** (`...T`) : 0..N arguments ; éclatement d'un slice avec `xs...`.
- **Tout est passé par valeur** : pour muter l'appelant ou éviter une grosse copie, passer
  un **pointeur**.
- Slices/maps/channels partagent leurs données sous-jacentes même copiés par valeur.
- Erreur = **dernier retour**, testée par l'appelant.

## 🔁 Pour aller plus loin

- [Ch. 8 — Structs, méthodes & composition](08-structs-methodes.md) : récepteur valeur vs pointeur.
- [Ch. 15 — Closures](15-closures.md) : fonctions anonymes qui capturent leur environnement.
- [Ch. 10 — Gestion des erreurs](10-erreurs.md) et [Ch. 39 — inlining/PGO](39-compilation-inlining-pgo.md).
