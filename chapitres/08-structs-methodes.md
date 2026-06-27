# Ch. 8 — Structs, méthodes & composition

> **Objectif** — Modéliser des **données** (`struct`) et leur **comportement** (méthodes),
> choisir le bon **récepteur** (valeur vs pointeur) et composer par **embedding**.
>
> **Prérequis** — [Ch. 5 — Fonctions](05-fonctions.md), [Ch. 6 — Arrays & slices](06-arrays-slices.md)

---

## Introduction

Go n'a ni classes ni héritage. Pour structurer des données, on utilise des **structs** ; pour
leur donner un comportement, on leur attache des **méthodes** ; et pour réutiliser/assembler,
on **compose** par _embedding_ (pas d'héritage de classe). Ce chapitre pose ces trois piliers
et la décision la plus fréquente du quotidien Go : **récepteur valeur ou pointeur ?**

L'exemple complet est dans [`code/ch08-structs/`](../code/ch08-structs/).

---

## Déclarer un struct

Un **struct** agrège des **champs** nommés et typés :

```go
type Point struct {
	X, Y float64
}

type Rectangle struct {
	Min, Max Point // un struct peut contenir d'autres structs (composition)
}
```

### Littéraux

```go
p1 := Point{1, 2}                              // positionnel : DANS L'ORDRE des champs
p2 := Point{X: 3}                              // nommé : Y prend sa zero value (0)
rect := Rectangle{Min: Point{0, 0}, Max: Point{3, 4}} // imbriqué
pp := &Point{X: 5, Y: 6}                       // pointeur vers un struct (type *Point)
np := new(Point)                               // *Point pointant sur un Point à zéro
```

> 💡 Préférez la forme **nommée** (`Point{X: 1, Y: 2}`) : elle résiste à l'ajout/réordonnancement
> de champs et documente le code. La forme positionnelle est réservée aux très petits structs
> stables (ex. `Point`).

### Zero value et accès

Un struct non initialisé a **tous ses champs à leur zero value** — il est immédiatement
utilisable, sans constructeur :

```go
var z Point        // {0 0}
z.X = 10           // accès par point
fmt.Println(z.Y)   // 0
```

L'accès aux champs via un **pointeur** est **déréférencé automatiquement** : `pp.X` équivaut à
`(*pp).X` (pas besoin d'écrire l'étoile).

### Comparaison

Deux structs sont comparables avec `==` **si et seulement si tous leurs champs le sont**
(comparaison champ par champ) :

```go
Point{1, 2} == Point{1, 2}   // true
```

> ⚠️ Un struct contenant un champ **non comparable** (`slice`, `map`, `func`) ne peut pas être
> comparé : `struct{ data []int }` avec `==` est une **erreur de compilation**
> (`struct containing []int cannot be compared`). Pour ces cas, comparez à la main ou via
> `reflect.DeepEqual` / `slices.Equal`.

### Champs exportés & tags

La visibilité d'un champ suit la règle de la **majuscule** (rappel [Ch. 2](02-structure-programme.md)) :
`X` est exporté, `balance` ne l'est pas. Un champ peut porter un **tag** — une string de
métadonnées lue par réflexion (encodeurs JSON, ORM…) :

```go
type User struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}
```

Les tags sont inertes pour le compilateur ; seul du code réflexif les interprète (détail au
[Ch. 34](34-reflexion.md), usage concret au Projet 2 — API REST).

---

## Méthodes

Une **méthode** est une fonction avec un **récepteur** : un paramètre spécial, placé **avant**
le nom, qui lie la fonction à un type.

```go
func (p Point) Distance(q Point) float64 {
	return math.Hypot(q.X-p.X, q.Y-p.Y)
}

d := Point{0, 0}.Distance(Point{3, 4}) // 5
```

On peut déclarer une méthode sur **n'importe quel type nommé défini dans le même package**
(pas seulement un struct) — par exemple `type Celsius float64` avec une méthode
`String() string`.

### Récepteur **valeur** vs **pointeur** — LA décision

```
   func (p Point)   M()   <- récepteur VALEUR  : la méthode reçoit une COPIE de p
   func (a *Account) M()  <- récepteur POINTEUR : la méthode reçoit l'ADRESSE de a
```

- **Récepteur valeur** : travaille sur une **copie**. Toute mutation reste **locale** et est
  perdue au retour. Idéal pour les types **petits et immuables** (`Point`, `time.Time`).
- **Récepteur pointeur** : travaille sur l'**original**. Permet de **muter** l'état, et **évite
  de copier** un gros struct à chaque appel.

```go
type Account struct{ balance int }

func (a *Account) Deposit(n int) { a.balance += n } // POINTEUR : la mutation persiste
func (a Account) Lost(n int)     { a.balance += n } // VALEUR  : modifie une copie -> sans effet
```

> 💡 **Auto-adressage** : sur une variable **adressable**, `acc.Deposit(100)` est réécrit en
> `(&acc).Deposit(100)` automatiquement. Mais une valeur **non adressable** (élément de map,
> retour de fonction, littéral) ne peut **pas** appeler une méthode à récepteur pointeur :
> `Account{}.Deposit(1)` ne compile pas.

**Règles de choix** (dans l'ordre) :

1. La méthode **modifie** le récepteur → **pointeur**.
2. Le struct est **gros** (coût de copie) → **pointeur**.
3. Le struct contient un champ **à ne pas copier** (`sync.Mutex`, etc.) → **pointeur**.
4. **Cohérence** : si **une** méthode est sur pointeur, mettez **toutes** les méthodes du type
   sur pointeur (pas de mélange valeur/pointeur).
5. Sinon (petit, immuable) → **valeur**, c'est plus simple et sûr.

### Method set (aperçu)

Le **method set** d'un type détermine quelles interfaces il satisfait (détail
[Ch. 9](09-interfaces.md)) :

- méthodes à récepteur **valeur** → dans le method set de `T` **et** de `*T` ;
- méthodes à récepteur **pointeur** → dans le method set de **`*T` uniquement**.

> ⚠️ Conséquence : si `Deposit` est sur `*Account`, alors **`*Account`** satisfait une interface
> qui exige `Deposit`, mais **`Account`** (valeur) ne la satisfait pas. On y revient au
> [Ch. 9](09-interfaces.md) et au [Ch. 33](33-interfaces-profondeur.md).

---

## Composition par **embedding**

En déclarant un champ **anonyme** (un type sans nom de champ), on **embarque** ce type. Ses
champs et méthodes exportés sont **promus** : on y accède directement sur le type englobant.

```go
type AuditedAccount struct {
	Account          // embedding : champ anonyme
	log     []string
}

aa := &AuditedAccount{Account: Account{Owner: "bob"}}
aa.Owner       // champ promu depuis Account
aa.Deposit(10) // méthode promue depuis Account
aa.Account.Owner // accès EXPLICITE au champ embarqué
```

```
   AuditedAccount
   +-------------------------------+
   |  Account  (embarqué)          |   --promotion-->  aa.Owner, aa.Deposit(), aa.Balance()
   |    Owner   string             |
   |    balance int                |
   +-------------------------------+
   |  log  []string                |
   +-------------------------------+
```

### Redéfinition (shadowing) et délégation

Définir sur le type englobant un champ/méthode du **même nom** **masque** celui de l'embarqué.
On délègue alors explicitement :

```go
func (a *AuditedAccount) Deposit(cents int) {
	a.Account.Deposit(cents) // délégation à la méthode embarquée
	a.log = append(a.log, fmt.Sprintf("deposit %d", cents))
}
```

### ⚠️ Embedding ≠ héritage : pas de dispatch dynamique

C'est **le** piège pour qui vient de la POO. Une méthode de l'embarqué qui en appelle une autre
appelle **toujours** la version de l'embarqué — **jamais** une redéfinition du type englobant.
L'embarqué « ne sait pas » qu'il est embarqué :

```go
// Si Account.Deposit appelle a.audit(), et AuditedAccount redéfinit audit(),
// alors aa.Deposit() exécute quand même Account.audit() (PAS celle d'AuditedAccount).
```

Il n'y a pas de méthodes virtuelles : la composition assemble, elle ne **surcharge** pas le
comportement interne du type embarqué. Pour un vrai polymorphisme, utilisez les **interfaces**
([Ch. 9](09-interfaces.md)).

> 💡 On peut aussi embarquer un **pointeur** (`*Account`) ou une **interface** — c'est la base de
> nombreux patterns (décorateur, wrappers de `io.Reader`…).

---

## Structs vides & alignement mémoire

### Le struct vide `struct{}`

`struct{}` n'a aucun champ et occupe **0 octet**. Il sert de valeur « présence sans donnée » :
ensembles `map[T]struct{}` ([Ch. 7](07-maps-strings.md)), signaux `chan struct{}`
([Ch. 20](20-channels-select.md)).

### Padding : l'ordre des champs compte

Le compilateur **aligne** chaque champ sur un multiple de sa taille et insère du **padding**
(octets de remplissage). À champs identiques, **l'ordre** change donc la taille totale (mesures
sur plateforme **64 bits**, via `unsafe.Sizeof`) :

```
   Padded { a bool; b int64; c bool }   ->  24 octets

   offset:  0    1                   8                        16   17                  23
           +----+-------------------+-------------------------+----+--------------------+
           | a  | padding (7 oct.)  |       b  (int64)        | c  | padding (7 oct.)   |
           +----+-------------------+-------------------------+----+--------------------+
            1o   (b doit être aligné sur 8)                     1o  (taille = multiple de 8)

   Packed { b int64; a bool; c bool }   ->  16 octets   (champ large en tête : -33 %)

   offset:  0                        8    9    10                 15
           +-------------------------+----+----+-------------------+
           |       b  (int64)        | a  | c  | padding (6 oct.)  |
           +-------------------------+----+----+-------------------+
            8o                         1o   1o
```

Ranger les champs **du plus large au plus étroit** minimise le padding. Effet marginal sur un
struct isolé, mais réel sur un **tableau de millions d'éléments**. Détail (et `Alignof`,
`Offsetof`) au [Ch. 35](35-unsafe-cgo.md).

---

## 🆕 Go 1.2x

- **1.18** — **structs génériques** : `type Stack[T any] struct{ items []T }`
  ([Ch. 11](11-genericite.md)).
- **1.23** — package **`structs`** : le marqueur `structs.HostLayout` (champ de type
  `structs.HostLayout`) indique au compilateur de respecter la disposition mémoire attendue par
  la plateforme hôte — utile pour l'interop **cgo/syscall** ([Ch. 35](35-unsafe-cgo.md)).

## ⚠️ Pièges

- **Récepteurs mixtes** valeur/pointeur sur un même type → method set incohérent et confusion.
  Choisissez **un** style par type.
- **Muter via un récepteur valeur** → modifie une copie ; l'original est intact (bug silencieux).
- **Copier un gros struct** (passage par valeur, affectation) → coût caché ; passez un `*T`.
- **Embedding pris pour de l'héritage** → pas de dispatch dynamique (voir ci-dessus).
- **Comparer un struct à champ non comparable** (`slice`/`map`/`func`) avec `==` → erreur de
  compilation.
- **Copier un struct contenant un `sync.Mutex`** → copie le verrou (data race). `go vet` le
  signale (analyzer `copylocks`).

## ⚡ Performance

- **Passer/retourner un `*T`** pour les gros structs évite des copies ; pour les **petits**
  (≤ ~2-3 mots), la valeur est souvent **plus rapide** (pas d'indirection, reste sur la pile).
- **Ordonner les champs** (large → étroit) réduit le padding et la pression mémoire/cache.
- `struct{}` est **gratuit** (0 octet) : idéal pour les ensembles et les canaux de signal.
- L'**escape analysis** ([Ch. 26](26-allocation-escape.md)) décide pile vs tas : un `&T{}` ne
  va pas forcément sur le tas.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch08-structs
go test ./ch08-structs/...
```

À essayer :

1. Transformez `Deposit` en récepteur **valeur** (`func (a Account) Deposit`) et observez le
   test `TestAccountPointerReceiver` échouer (le solde ne bouge plus).
2. Ajoutez un champ `tags []string` à `Point` et tentez `Point{} == Point{}` : erreur de
   compilation.
3. Mesurez `unsafe.Sizeof` de vos propres structs et réordonnez les champs pour réduire la
   taille.

---

## 📌 À retenir

- Un **struct** agrège des champs ; **zero value** utilisable sans constructeur ; littéraux
  **nommés** de préférence.
- Une **méthode** a un **récepteur** : **valeur** (copie, petit/immuable) ou **pointeur**
  (mutation, gros struct) — ne **mélangez pas** les deux sur un type.
- Comparaison `==` champ par champ, **si tous les champs sont comparables**.
- L'**embedding** compose et **promeut** champs/méthodes, mais **n'est pas de l'héritage** : pas
  de dispatch dynamique.
- L'**ordre des champs** influe sur la taille (padding) ; `struct{}` pèse **0 octet**.

## 🔁 Pour aller plus loin

- [Ch. 9 — Interfaces](09-interfaces.md) : le polymorphisme par le comportement (method sets).
- [Ch. 33 — Interfaces & types en profondeur](33-interfaces-profondeur.md) : itab, dispatch.
- [Ch. 35 — `unsafe`](35-unsafe-cgo.md) : `Sizeof`/`Alignof`/`Offsetof`, alignement, `HostLayout`.
- [Ch. 34 — Réflexion](34-reflexion.md) : lecture des **tags** de struct.
- [Ch. 11 — Généricité](11-genericite.md) : structs et méthodes paramétrés.
