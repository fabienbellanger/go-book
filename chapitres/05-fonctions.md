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

- Le **type suit l'identifiant** (`a int`, pas `int a`) : même convention que pour `var`
  (Ch. 3). On lit `a int` comme « a est un int » — et ça reste lisible pour des types
  composés (`f func(int) (int, error)`), là où la syntaxe préfixée de C devient vite
  cryptique pour un pointeur de fonction ou un tableau de pointeurs.
- Des paramètres consécutifs de même type se factorisent : `func add(a, b int) int`.
- Sans valeur de retour, on omet la partie droite : `func log(msg string) { … }`.

> ⚠️ Go n'a ni **surcharge de fonctions** (un seul `add` possible, quels que soient les
> types de ses paramètres) ni **valeurs par défaut**. Deux idiomes du langage comblent ce
> manque, tous deux détaillés plus bas dans ce chapitre : les **fonctions variadiques**
> pour un nombre variable d'arguments, et le **functional options pattern** pour des
> réglages optionnels nommés.

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

> 💡 Ces valeurs multiples **ne forment pas un type tuple** : impossible de les affecter à
> une seule variable (`x := divmod(17, 5)` ne compile pas) ni de les stocker dans un
> slice. La seule construction qui les manipule en bloc est l'appel direct d'une fonction
> sur le résultat d'une autre, quand les arités correspondent terme à terme :
> `sum(divmod(17, 5))` appelle en fait `sum(3, 2)` et vaut `5`, sans variable
> intermédiaire. Pratique pour enchaîner deux appels, mais à réserver aux cas évidents :
> ça masque ce qui est réellement passé.

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

> ⚠️ Piège plus sournois que le `return` nu : un `:=` dans un bloc imbriqué peut
> **masquer** une variable de retour nommée (même mécanisme que le _shadowing_ du
> [Ch. 3](03-variables-constantes-types.md), appliqué ici à `err`) :
>
> ```go
> func find(id int) (result string, err error) {
> 	if v, err := fetch(id); err != nil { // := crée un NOUVEL err, local au if/else
> 		return "", err                   // celui-ci est le bon
> 	} else {
> 		result = v // ... mais ICI, le err NOMMÉ n'a jamais été touché
> 	}
> 	return // renvoie (result, err) -> err nommé resté à nil : bug silencieux
> }
> ```
>
> N'utilisez `:=` dans un sous-bloc que si aucun des identifiants déclarés ne porte le nom
> d'un retour nommé de la fonction englobante ; sinon, utilisez `=` pour réaffecter.

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

> ⚠️ Impossible de **mélanger** valeurs individuelles et slice éclaté dans le même appel :
> `sum(1, xs...)` est refusé à la compilation (`too many arguments`). C'est l'un ou
> l'autre.

> 💡 Sans argument, le paramètre variadique vaut **`nil`**, pas seulement `len == 0` :
> `sum()` reçoit `nums == nil`, alors que `sum([]int{}...)` reçoit un slice non-nil de
> longueur 0. La nuance compte si le corps de la fonction teste `nums == nil` pour
> distinguer « aucun argument fourni » de « slice vide fourni explicitement ».

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

« Stockée » n'est pas qu'un mot en l'air : une `map[string]func(...) T` donne une **table
de dispatch**, une alternative à une cascade de `switch` quand les cas se multiplient ou
sont ajoutés dynamiquement :

```go
ops := map[string]func(int, int) int{
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
}
result := ops["add"](2, 3) // 5
```

> ⚠️ Les valeurs `func` ne sont **comparables qu'à `nil`** : `f == g` entre deux fonctions
> est refusé à la compilation (`func can only be compared to nil`). Conséquence
> pratique : impossible de s'en servir comme **clé de map**, et impossible de comparer
> deux closures « par identité » pour de la mémoïsation.

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

> ⚠️ « Pas infinie » a une limite précise : par défaut, une pile de goroutine peut croître
> jusqu'à **1 Go sur une architecture 64 bits** (250 Mo en 32 bits — voir
> `runtime/debug.SetMaxStack`). Au-delà, le programme s'arrête sur `fatal error: stack
overflow` — une **erreur fatale du runtime**, pas une `panic` ordinaire : un
> `recover()` placé dans un `defer` ne la rattrape **pas** (🔁 [Ch. 17](17-panic-recover.md)).

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

La même règle s'applique à un **tableau** (`[N]T`, taille fixée à la compilation) : le
passer en paramètre copie **tous ses éléments**, exactement comme un struct. C'est
précisément pour éviter ce coût sur de grandes collections que Go propose les **slices**
— un type distinct du tableau (🔁 [Ch. 6](06-arrays-slices.md)), dont seul un petit
descripteur est copié, comme détaillé ci-dessous.

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

Alternative courante : un seul paramètre `Config` struct (`NewServer(cfg Config)`).

| Aspect               | Functional options                                       | Struct de config                                                                   |
| -------------------- | -------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Ajout d'un réglage   | Toujours sans casse (nouvelle fonction `With...`)        | Sans casse si les appelants utilisent des champs **nommés** (`Config{Port: 9000}`) |
| Validation           | Possible option par option (`WithPort` peut rejeter < 0) | Faite après coup, sur l'ensemble du struct, dans `New...`                          |
| Coût                 | Une closure allouée par option utilisée                  | Aucune allocation supplémentaire                                                   |
| Lisibilité à l'appel | `NewServer(WithPort(9000), WithTLS())` — auto-documenté  | `NewServer(Config{Port: 9000, TLS: true})` — un seul bloc à lire                   |

> 💡 Repère pratique : _functional options_ pour une **bibliothèque publique** dont la
> liste de réglages grandira avec le temps ; struct de config pour du **code interne** à
> la liste de champs stable, où l'allocation supplémentaire des closures ne se justifie
> pas.

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
- Un appel **variadique** (`sum(1, 2, 3)`) construit un `[]T` pour porter les arguments :
  si l'analyse d'échappement prouve qu'il ne survit pas à l'appel, ce slice reste sur la
  **pile** (vérifiable avec `go build -gcflags=-m`, qui rapporte alors `... argument does
not escape`) ; sinon il est réalloué sur le **tas** à chaque appel.

## ⚠️ Pièges

- **Oublier de tester `err`** — le compilateur ne force pas à lire un retour ; `go vet` et
  les linters aident (`errcheck`).
- **Appeler une fonction `nil`** — panique. Vérifiez les callbacks optionnels.
- **Gros struct copié** — passé par valeur, il est entièrement recopié à chaque appel ;
  utilisez un pointeur si la taille le justifie.
- **`return` nu dans une longue fonction** — illisible ; renvoyez explicitement.
- **Retour nommé masqué par un `:=`** dans un sous-bloc — le bug est silencieux (la zero
  value est renvoyée). Voir l'exemple détaillé plus haut.
- **Mélanger valeurs et slice éclaté** dans un même appel variadique — refusé à la
  compilation.
- **Comparer deux fonctions** (`f == g`) — refusé à la compilation ; seule la comparaison
  à `nil` est permise.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch05-functions
go test ./ch05-functions/...
go build -gcflags=-m ./ch05-functions 2>&1 | grep inlin # décisions d'inlining
```

À essayer :

1. Le fichier réel propose déjà `WithHost` (voir `code/ch05-functions/options.go`) :
   ajoutez à votre tour une option `WithMaxConns(n int)` et vérifiez que les appels
   existants (`NewServer(WithPort(9000), WithTLS())`) compilent toujours sans
   modification (extensibilité).
2. Transformez `incVal` pour qu'elle modifie réellement le counter (passez `*counter`).
3. Écrivez `compose(f, g func(int) int) func(int) int` qui renvoie `x -> f(g(x))`.
4. Vérifiez que `sum(divmod(17, 5))` compile et vaut `5` ; expliquez pourquoi, à
   l'inverse, `q := divmod(17, 5)` ne compile pas.

---

## 📌 À retenir

- Les fonctions sont des **valeurs de première classe** (paramètres, retours, variables).
- **Retours multiples** : fondent les idiomes `v, ok` et `v, err`.
- **Variadique** (`...T`) : 0..N arguments ; éclatement d'un slice avec `xs...`.
- **Tout est passé par valeur** : pour muter l'appelant ou éviter une grosse copie, passer
  un **pointeur**.
- Slices/maps/channels partagent leurs données sous-jacentes même copiés par valeur.
- Erreur = **dernier retour**, testée par l'appelant.
- Pas de surcharge ni de valeurs par défaut : **variadique** et _functional options_
  comblent ce manque dans le langage.

## 🔁 Pour aller plus loin

- [Ch. 8 — Structs, méthodes & composition](08-structs-methodes.md) : récepteur valeur vs pointeur.
- [Ch. 15 — Closures](15-closures.md) : fonctions anonymes qui capturent leur environnement.
- [Ch. 10 — Gestion des erreurs](10-erreurs.md) et [Ch. 39 — inlining/PGO](39-compilation-inlining-pgo.md).
