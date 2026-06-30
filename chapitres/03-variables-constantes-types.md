# 3 — Variables, constantes & types de base

> **Objectif** — Déclarer et typer des valeurs : variables, constantes, types
> numériques, et comprendre les conversions explicites de Go.
>
> **Prérequis** — [Ch. 2 — Structure d'un programme](02-structure-programme.md)

---

## Introduction

Go est **statiquement typé** mais **peu bavard** : grâce à l'inférence, on déclare
rarement les types explicitement. En contrepartie, il **refuse toute conversion
implicite** entre types — un parti pris qui élimine une classe entière de bugs. Ce
chapitre couvre les briques de base : déclarer, typer, convertir.

L'exemple complet est dans [`code/ch03-basics/`](../code/ch03-basics/).

## Déclarer une variable

Trois formes, du plus explicite au plus concis :

```go
var age int = 30 // 1. type + valeur explicites
var age = 30     // 2. type inféré depuis la valeur (-> int)
age := 30        // 3. déclaration courte := (type inféré)
```

- `var` fonctionne **partout** (niveau package ou fonction).
- `:=` ne fonctionne **que dans une fonction** (jamais au niveau package), et exige
  qu'**au moins une** variable à gauche soit nouvelle.

`:=` est une **instruction exécutable** : il n'a de sens que dans un flux séquentiel.
Au niveau package, les déclarations ne s'exécutent pas dans l'ordre du fichier — le
compilateur résout d'abord les **dépendances entre elles** (une variable peut être
initialisée à partir d'une autre déclarée plus bas) ; `var` est donc la seule forme
valide à cet endroit.

La règle « au moins une nouvelle variable » permet de **mélanger** réaffectation et
déclaration sur une même ligne :

```go
x := 1
x, y := 2, 3 // x est RÉAFFECTÉE (pas redéclarée), y est nouvelle -> OK
x, y := 4, 5 // ERREUR : no new variables on left side of := (aucune des deux n'est neuve)
```

On **regroupe** les déclarations dans un bloc `var ( … )` :

```go
var (
	i int
	f float64
	s string
)
```

> ⚠️ Une variable **locale déclarée mais non utilisée** est une **erreur de
> compilation** (comme les imports inutilisés). Au niveau package, en revanche, une
> variable non utilisée est tolérée.

## Zero values : pas de variable « non initialisée »

Toute variable déclarée sans valeur reçoit la **valeur nulle** (_zero value_) de son
type. Il n'existe pas de mémoire « indéterminée » comme en C.

| Type                          | Zero value  |
| ----------------------------- | ----------- |
| Numériques (`int`, `float64`) | `0`         |
| `bool`                        | `false`     |
| `string`                      | `""` (vide) |
| Pointeurs, slices, maps, …    | `nil`       |

```go
var (
	i int
	f float64
	b bool
	s string
)
// Affiche : i=0 f=0 b=false s=""
fmt.Printf("i=%d f=%g b=%t s=%q\n", i, f, b, s)
```

> 💡 Concevez vos types pour que **la zero value soit utile**. Par exemple, un
> `bytes.Buffer` vide est directement utilisable, un `sync.Mutex` à zéro est déverrouillé.

Ce choix élimine toute une classe de bugs présents en C : là où une variable locale non
initialisée contient des **octets indéterminés** (ce qui reste de la pile à cet endroit —
un comportement non défini, parfois exploitable comme faille de sécurité), Go garantit
qu'une variable se lit **toujours** de façon prévisible, dès sa déclaration.

> ⚠️ La zero value n'est utile **qu'en lecture**. Un slice ou une map valant `nil` se
> comporte comme vide en lecture (`len(s) == 0`, une recherche dans une map nil renvoie la
> zero value de l'élément sans paniquer), mais **écrire** dans une map `nil` panique :
> `m["k"] = 1` sur une map non initialisée provoque `panic: assignment to entry in nil
map`. Il faut l'initialiser avec `make` au préalable (détail au [Ch. 7](07-maps-strings.md)).

## Les types de base

### Entiers

| Type               | Taille               | Signé | Intervalle                                             |
| ------------------ | -------------------- | ----- | ------------------------------------------------------ |
| `int8`             | 8 bits               | oui   | −128 … 127                                             |
| `int16`            | 16 bits              | oui   | −32 768 … 32 767                                       |
| `int32`            | 32 bits              | oui   | ≈ ±2,1 × 10⁹                                           |
| `int64`            | 64 bits              | oui   | ≈ ±9,2 × 10¹⁸                                          |
| `uint8` … `uint64` | 8 … 64 bits          | non   | 0 … 2ⁿ−1                                               |
| `int` / `uint`     | **32 ou 64 bits**    | —     | dépend de la plateforme (64 sur les machines modernes) |
| `uintptr`          | taille d'un pointeur | non   | pour l'arithmétique d'adresses (rare, voir Ch. 35)     |

- **`byte`** est un **alias** de `uint8` (octet brut).
- **`rune`** est un **alias** de `int32` (un **point de code Unicode**, voir Ch. 7).

```go
r := 'é'       // rune (int32) = 233
b := byte('A') // byte (uint8) = 65
```

> ⚠️ **`int` n'a pas une taille fixe** : 64 bits sur un Mac/Linux 64 bits, mais 32 bits
> ailleurs. Pour un format de fichier, un protocole réseau ou une sérialisation, utilisez
> un type **de taille explicite** (`int32`, `int64`), jamais `int`.

> 💡 Cela dit, dans le code « courant » (compteurs, indices, boucles), utilisez `int` par
> défaut : c'est le type **idiomatique**, et c'est ce que renvoient `len`, `cap` et les
> index de slice/array — y mélanger un `int32` ou `uint` obligerait à convertir à chaque
> usage. Réservez les types de taille fixe aux cas où la taille **doit** être garantie.

### Flottants & complexes

- `float64` (par défaut) et `float32` — norme IEEE 754.
- `complex128` et `complex64` — nombres complexes natifs (`complex`, `real`, `imag`).

`float64` offre environ **15 à 17 chiffres décimaux significatifs** contre **6 à 9** pour
`float32` — c'est pourquoi c'est le type **par défaut** dès qu'un littéral flottant n'est
pas explicitement typé. Préférez `float32` seulement quand la mémoire compte (gros
volumes de données, interopérabilité avec du C ou du matériel graphique) et que la perte
de précision est acceptable.

```go
var x float64 = 3.14
c := complex(1, 2) // 1+2i (complex128)
```

> ⚠️ Les flottants sont **approximatifs** : `0.1 + 0.2 != 0.3`. Ce n'est pas un défaut de
> Go : la norme IEEE 754 représente les nombres en **base 2**, et la plupart des fractions
> décimales (0,1 = 1/10) n'ont pas de représentation binaire exacte — le même résultat
> apparaît en C, Java, Python ou JavaScript. Ne comparez jamais deux flottants avec `==` ;
> comparez `math.Abs(a-b) < epsilon`.

### Booléens & chaînes

- `bool` : `true` / `false` (pas de conversion entier ↔ booléen).
- `string` : suite **immuable** d'octets (UTF-8 en pratique). Détail au [Ch. 7](07-maps-strings.md).

## Conversions : explicites, toujours

Go **n'effectue aucune conversion implicite** entre types numériques, même « élargissante ».
Toute conversion est **écrite à la main** avec la syntaxe `T(v)` :

```go
var big int64 = 9_000
small := int32(big)          // OK : conversion explicite
half := float64(small) / 2.0 // int32 -> float64 avant la division
```

Sans la conversion, c'est une **erreur de compilation** :

```go
var a int32 = 1
var b int64 = a // ERREUR : cannot use a (int32) as int64 value
```

C'est verbeux, mais **intentionnel** : aucune perte ou promotion silencieuse ne se
glisse dans votre dos. Go évite ainsi une classe de bugs bien connue en C, où comparer un
`int` signé et un `unsigned int` **convertit implicitement** le signé vers le non signé :

```go
// En C : if (i < u) où i est int (-1) et u est unsigned int (1)
// convertit -1 en unsigned -> une très grande valeur positive -> comparaison fausse.
var i int = -1
var u uint = 1
// if i < u {}              // ERREUR de compilation en Go : type mismatch
if int64(i) < int64(u) {}   // il faut convertir explicitement, dans un type commun
```

En Go, ce mélange est **rejeté à la compilation** : la conversion doit être écrite, donc
relue et assumée par l'auteur du code.

> 💡 Le séparateur `_` est autorisé dans les littéraux numériques pour la lisibilité :
> `1_000_000`, `0xFF_FF`, `0b1010_0101`.

## Constantes : typées vs non typées

Une **constante** est figée à la compilation (`const`). Sa subtilité : elle peut être
**non typée**.

```go
const Pi = 3.14159     // NON typée : s'adapte au contexte
const Max int = 32_767 // typée : c'est un int, point.
```

Une constante **non typée** possède une **précision arbitraire** et un **type par
défaut**, qui ne s'applique que si le contexte n'impose rien d'autre — par exemple un
`fmt.Println(Pi)` sans variable cible affiche `Pi` avec son type par défaut :

| Genre de littéral non typé | Type par défaut  |
| -------------------------- | ---------------- |
| Entier (`42`)              | `int`            |
| Flottant (`3.14`)          | `float64`        |
| Caractère (`'é'`)          | `rune` (`int32`) |
| Chaîne (`"go"`)            | `string`         |
| Booléen (`true`)           | `bool`           |

Quand le contexte impose un type différent — une affectation, un paramètre de
fonction —, c'est ce type-là qui s'applique, à condition que la valeur y tienne :

```go
const big = 1 << 62  // non typée : aucun débordement à la compilation
var n int64 = big    // OK : 4 611 686 018 427 387 904
var f float64 = Pi   // la même constante Pi sert ici de float64
var g float32 = Pi   // … et là de float32, sans nouvelle déclaration
```

> ⚠️ Une constante **hors limites** du type cible est rejetée **à la compilation** (elle
> ne « déborde » jamais silencieusement) :
> `var x int8 = 128` → _cannot use 128 … as int8 value (overflows)_.

> ⚠️ La précision arbitraire ne change pas les **règles d'opérateurs** : diviser deux
> constantes entières reste une **division entière**, même non typées :
>
> ```go
> const third = 1 / 3   // 0 : les deux opérandes sont des constantes ENTIÈRES
> const thirdF = 1.0 / 3 // 0.333333333333333333333... (précision arbitraire, un seul
>                         // opérande suffit à rendre l'expression flottante)
> ```

### `iota` : compteur de constantes

`iota` est un compteur qui vaut **0 sur la première ligne** d'un bloc `const`, puis
**+1 à chaque ligne suivante**. Idéal pour des énumérations :

```go
type Direction int

const (
	North Direction = iota // 0
	East                   // 1
	South                  // 2
	West                   // 3
)
```

> 💡 Idiome courant : décaler avec `iota + 1` pour que la **zero value** du type reste un
> état « non défini » plutôt qu'une valeur métier valide — utile pour détecter une variable
> oubliée à sa zero value :
>
> ```go
> type Weekday int
>
> const (
> 	Sunday Weekday = iota + 1 // 1 : la zero value (0) ne désigne aucun jour
> 	Monday                    // 2
> 	// …
> )
> ```

Quand une ligne **omet** son expression, elle **répète celle de la ligne précédente**
(en réévaluant `iota`). On exploite ça pour des multiples binaires (tiré de
`code/ch03-basics/sizes.go`) :

```go
type ByteSize int64

const (
	_  ByteSize = iota             // 0 (absorbé par le blank _)
	KB ByteSize = 1 << (10 * iota) // iota=1 -> 1<<10 = 1024
	MB                             // iota=2 -> 1<<20
	GB                             // iota=3 -> 1<<30
	TB                             // iota=4 -> 1<<40
)
```

Le `_` (identifiant **blank**) sert ici à **ignorer** la valeur 0. Pousser jusqu'à
`1<<70` provoquerait une **erreur de compilation** (débordement d'`int64`).

## Portée & _shadowing_

La **portée** d'une variable est le bloc `{ … }` où elle est déclarée. Une variable
interne peut **masquer** (_shadow_) une variable de même nom d'un bloc englobant :

```go
x := 1
if true {
	x := 2     // NOUVELLE variable, masque l'externe (note le :=)
	fmt.Println(x) // 2
}
fmt.Println(x)     // 1 : l'externe n'a pas changé
```

> ⚠️ Le _shadowing_ est un **piège classique**, surtout avec `err` :
>
> ```go
> v, err := f()
> if cond {
> 	v, err := g() // :=  -> NOUVEAU err, l'externe n'est pas mis à jour !
> 	_ = v
> }
> // ici, err est toujours celui de f()
> ```
>
> Utilisez `=` (pas `:=`) quand vous voulez **réaffecter** la variable existante.
> L'analyseur `shadow` (de `golang.org/x/tools`, hors `go vet` par défaut) le détecte.

## Les fonctions _built-in_

Quelques fonctions sont **intégrées au langage** (pas besoin d'import) :

| Built-in               | Rôle                                              | Détail     |
| ---------------------- | ------------------------------------------------- | ---------- |
| `len(x)`               | longueur (string, slice, map, chan, array)        | Ch. 6/7    |
| `cap(x)`               | capacité (slice, chan, array)                     | Ch. 6      |
| `make(T, …)`           | crée slice/map/chan **initialisés**               | Ch. 6/7    |
| `new(T)` / `new(expr)` | pointeur vers une valeur (zero value ou expr)     | ci-dessous |
| `append(s, …)`         | ajoute à un slice                                 | Ch. 6      |
| `copy(dst, src)`       | copie entre slices                                | Ch. 6      |
| `delete(m, k)`         | supprime une clé de map                           | Ch. 7      |
| `min(…)` / `max(…)`    | minimum / maximum (**🆕 1.21**)                   | ici        |
| `clear(x)`             | vide une map ou met un slice à zéro (**🆕 1.21**) | ici        |

## 🆕 Nouveautés

- **🆕 Go 1.21** — `min`, `max` et `clear` deviennent des **built-ins** :

  ```go
  min(42, 100) // 42
  max(3.14, 2) // 3.14
  m := map[string]bool{"a": true}
  clear(m) // m est désormais vide (len == 0)
  ```

- **🆕 Go 1.26** — `new` accepte une **expression d'initialisation**. `new(expr)` alloue
  une variable, l'initialise à la valeur de `expr`, et renvoie son pointeur — le **type
  est inféré** depuis l'expression :

  ```go
  p := new(7)     // *int   pointant vers 7
  q := new("hi")  // *string pointant vers "hi"
  ```

  Avant 1.26, il fallait écrire `x := 7; p := &x`. Le `new(expr)` condense ce motif très
  courant (obtenir un `*T` à partir d'une valeur). Une **valeur littérale** n'est pas
  adressable directement (`&7` est une erreur de compilation : il faut d'abord la stocker
  dans une variable) — c'est précisément ce détour que de nombreux projets contournaient
  avec une petite fonction maison du genre `func Ptr[T any](v T) *T { return &v }`
  (souvent nommée `Ptr`, `ToPtr` ou via des helpers comme `aws.String`). `new(expr)`
  rend ce générique-maison inutile pour le cas simple.

## ⚠️ Pièges récapitulés

- **Débordement silencieux à l'exécution** — l'arithmétique sur entiers de taille fixe
  « boucle » sans erreur : `var x int8 = 127; x++` donne **−128**, `var u uint8 = 0; u--`
  donne **255**. Contrôlez les bornes (voir `toInt8` dans l'exemple).
- **`int` n'est pas `int64`** — taille dépendante de la plateforme ; fixez la taille pour
  tout ce qui sort du programme (fichiers, réseau).
- **Conversion `float` → `int`** — **tronque** vers zéro (`int(3.99) == 3`), elle n'arrondit
  pas.
- **Comparaison de flottants** — jamais avec `==`.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch03-basics
go test ./ch03-basics/...
```

À essayer :

1. Remplacez `int8(n)` par une conversion sans garde dans `toInt8` et observez le résultat
   de `int8(200)`.
2. Ajoutez `PB` puis `EB` à l'`iota` de `ByteSize` ; ajoutez ensuite `1<<70` et lisez
   l'erreur de compilation.
3. Déclarez `var x int8 = 200` : constatez que l'erreur est attrapée **à la compilation**.

---

## 📌 À retenir

- Trois façons de déclarer : `var x T = v`, `var x = v`, `x := v` (cette dernière en
  fonction uniquement).
- Toute variable a une **zero value** ; concevez vos types pour qu'elle soit utile.
- **Aucune conversion implicite** : on écrit `T(v)`, toujours.
- Constantes **non typées** = précision arbitraire + type par défaut contextuel ; `iota`
  pour les énumérations.
- Débordement d'entier : **silencieux à l'exécution**, mais **rejeté à la compilation**
  pour une constante.
- `min`/`max`/`clear` (🆕 1.21) et `new(expr)` (🆕 1.26) sont des built-ins récents.

## 🔁 Pour aller plus loin

- [Ch. 4 — Flux de contrôle](04-flux-controle.md).
- [Ch. 6 — Arrays & slices](06-arrays-slices.md) et [Ch. 7 — Maps & strings](07-maps-strings.md) pour `make`, `append`, `len`/`cap`.
- [Ch. 26 — Allocation mémoire & escape analysis](26-allocation-escape.md) : où vivent `new` et `make`.
