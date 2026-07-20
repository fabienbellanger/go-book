# 8 — Structs, méthodes & composition

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

Le littéral **positionnel** doit fournir une valeur pour **chaque champ**, dans l'ordre exact de
la déclaration : en omettre un est une erreur de compilation (`too few values in struct
literal`). Le littéral **nommé** n'a pas cette contrainte — un champ absent prend simplement sa
zero value, et l'ordre des clés est libre. Les deux styles ne se mélangent pas dans un même
littéral.

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

### Affecter un struct, c'est le copier — superficiellement

`v2 := v1` (et de même, passer `v1` **par valeur** à une fonction) copie **tous les champs**
de `v1`, y compris ceux qui sont des **slices, maps ou pointeurs**. Mais ces champs-là ne sont
que des **en-têtes** (pointeur + métadonnées, [Ch. 6](06-arrays-slices.md)) : les copier ne
copie pas les données qu'ils référencent — les deux structs finissent par **partager** le même
tableau ou la même table sous-jacente.

```go
type Cart struct {
	Items []string
}

c1 := Cart{Items: []string{"a", "b"}}
c2 := c1                  // copie le struct... mais Items pointe vers le MÊME tableau
c2.Items[0] = "z"
fmt.Println(c1.Items[0])  // "z" : c1 est modifié aussi !
```

> ⚠️ C'est une **copie superficielle** (_shallow copy_). Seuls les champs scalaires
> (`int`, `string`, `bool`, `Point`…) sont réellement indépendants après l'affectation. Pour
> une copie totalement isolée d'un champ slice, dupliquez-le explicitement avec
> `slices.Clone` ([Ch. 6](06-arrays-slices.md)) ; pour une map, copiez clé par clé ou via
> `maps.Clone` ([Ch. 7](07-maps-strings.md)).

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

Syntaxiquement, un tag est une suite de paires `clé:"valeur"` **séparées par des espaces**, à
l'intérieur d'un littéral brut (backticks) — par exemple `json:"name" validate:"required"`.
Pour `encoding/json`, le premier élément de la valeur (avant une éventuelle virgule) est le nom
de clé JSON ; les suivants sont des options : `omitempty` omet le champ si sa valeur vaut sa
zero value, `-` l'exclut systématiquement de l'encodage.

Les tags sont inertes pour le compilateur — une faute de frappe (`josn:"name"`) ne provoque
**aucune erreur de compilation**, elle est simplement ignorée silencieusement à l'exécution.
Seul du code réflexif les interprète (détail au [Ch. 34](34-reflexion.md), usage concret au
Projet 2 — API REST).

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

> 💡 Convention de nommage : le récepteur est une **abréviation courte** du type (`p` pour
> `Point`, `a` pour `Account`) — jamais `this` ni `self` comme dans d'autres langages, Go n'a
> pas de mot-clé dédié au récepteur. La **même** lettre doit être réutilisée sur **toutes** les
> méthodes d'un même type ; un linter externe à `go vet` (`staticcheck`, règle `ST1006`) le
> signale sinon.

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
> `Account{}.Deposit(1)` ne compile pas. Symétriquement, un **pointeur** peut toujours appeler
> une méthode à récepteur **valeur** : `pp.Distance(q)` (avec `pp *Point`) est réécrit en
> `(*pp).Distance(q)` — un pointeur valide est toujours déréférençable, cette direction ne
> connaît donc pas l'exception d'adressabilité ci-dessus.

**Règles de choix** (dans l'ordre) :

1. La méthode **modifie** le récepteur → **pointeur**.
2. Le struct est **gros** (coût de copie) → **pointeur**.
3. Le struct contient un champ **à ne pas copier** (`sync.Mutex`, etc.) → **pointeur**.
4. **Cohérence** : si **une** méthode est sur pointeur, mettez **toutes** les méthodes du type
   sur pointeur (pas de mélange valeur/pointeur).
5. Sinon (petit, immuable) → **valeur**, c'est plus simple et sûr.

> 📌 La règle 4 se résume à une idée : **le choix valeur/pointeur se décide au niveau du
> type, pas méthode par méthode.** Dès que **l'une** des règles 1-3 impose un pointeur pour
> **une** méthode, alignez **tout le type** sur pointeur. On ne garde la valeur (règle 5) que
> pour les types **entièrement** petits et immuables où **aucune** méthode n'a besoin d'un
> pointeur (`Point`, `time.Time`). C'est aussi la recommandation officielle de Go
> (« don't mix receiver types »).

Mélanger les deux styles a trois conséquences fâcheuses.

**Mutation silencieusement perdue.** Une méthode à récepteur valeur travaille sur une
**copie**, une méthode à récepteur pointeur sur l'**original**. Selon l'adressabilité du
support, la même API mute l'état... ou pas :

```go
func (a Account)  Balance() int  { return a.balance } // VALEUR
func (a *Account) Deposit(n int) { a.balance += n }   // POINTEUR

s := []Account{{balance: 10}}
s[0].Deposit(5)   // OK : élément de slice adressable -> mute l'original

m := map[string]Account{"x": {balance: 10}}
// m["x"].Deposit(5) // NE COMPILE PAS : élément de map non adressable
```

Avec un type **tout pointeur** (`[]*Account`, `map[string]*Account`), ces cas se comportent
**uniformément**.

**Satisfaction d'interface incohérente.** Une méthode sur pointeur n'entre que dans le
method set de `*T` (voir juste en dessous). Mélanger force l'appelant à retenir « ici il me
faut un `*Account`, pas un `Account` » ; tout sur pointeur rend la règle triviale : « on
manipule toujours des `*Account` ».

**Copies accidentelles dangereuses.** Si le struct contient un champ non copiable
(`sync.Mutex`) ou dont la copie casse un invariant, une méthode à récepteur valeur en fait
une copie **à chaque appel** — une porte laissée ouverte sur les seules méthodes « oubliées ».

> 🔁 Le *pourquoi* technique de la première et de la deuxième conséquence tient au **method
> set**, l'objet de la section suivante.

### Method set (aperçu)

Le **method set** d'un type détermine quelles interfaces il satisfait (détail
[Ch. 9](09-interfaces.md)) :

| Récepteur déclaré                | Dans le method set de `T` ? | Dans le method set de `*T` ? |
| -------------------------------- | :-------------------------: | :--------------------------: |
| **valeur** (`func (p T) M()`)    |             ✅              |              ✅              |
| **pointeur** (`func (p *T) M()`) |             ❌              |              ✅              |

Concrètement, avec `Deposit` déclaré sur `*Account` (récepteur pointeur, voir plus haut) :

```go
type Depositor interface{ Deposit(int) }

var _ Depositor = &Account{} // OK : *Account a Deposit dans son method set
// var _ Depositor = Account{} // erreur de compilation :
//   Account ne possède pas Deposit (méthode déclarée à récepteur pointeur)
```

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

### Conflits de noms entre plusieurs embeddings

Avec **plusieurs** champs embarqués, deux champs/méthodes de **même nom** à la **même
profondeur** créent une **ambiguïté** : y accéder directement (`x.Foo`) est une erreur de
compilation (`ambiguous selector x.Foo`) — il faut alors désambiguïser explicitement
(`x.A.Foo` ou `x.B.Foo`). Si les noms apparaissent à des **profondeurs différentes**, Go
retient automatiquement le **moins profond**, sans erreur. C'est précisément ce mécanisme de
profondeur qui rend possible la redéfinition (shadowing) ci-dessous : un champ ou une méthode
du type englobant est à la profondeur **0**, donc toujours plus proche que son équivalent
embarqué (profondeur **1**) — il l'emporte systématiquement.

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
- **Copier un struct contenant un slice/map** → copie **superficielle** : les deux variables
  partagent les mêmes données sous-jacentes (voir « Affecter un struct » ci-dessus).
- **Embedding pris pour de l'héritage** → pas de dispatch dynamique (voir ci-dessus).
- **Comparer un struct à champ non comparable** (`slice`/`map`/`func`) avec `==` → erreur de
  compilation.
- **Copier un struct contenant un `sync.Mutex`** → copie le verrou (data race). `go vet` le
  signale (analyzer `copylocks`).

## ⚡ Performance

- **Passer/retourner un `*T`** pour les gros structs évite des copies ; pour les **petits**
  (≤ ~2-3 mots), la valeur est souvent **plus rapide** (pas d'indirection, reste sur la pile).
- **Ordonner les champs** (large → étroit) réduit le padding et la pression mémoire/cache.
  L'analyseur `fieldalignment` (`golang.org/x/tools/go/analysis/passes/fieldalignment`,
  `go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest`)
  détecte — et peut réécrire (`-fix`) — automatiquement un ordre de champs sous-optimal.
- `struct{}` est **gratuit** (0 octet) : idéal pour les ensembles et les canaux de signal.
- **Embarquer par valeur ne coûte rien de plus** qu'un champ nommé classique : mêmes octets,
  mêmes accès directs, pas d'indirection. Embarquer un **pointeur** (`*Account`) ajoute en
  revanche une indirection à chaque champ/méthode promu, exactement comme un champ `*T` normal.
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
- Affecter un struct le **copie**, mais **superficiellement** : les champs slice/map/pointeur
  restent partagés entre l'original et la copie.
- Une **méthode** a un **récepteur** : **valeur** (copie, petit/immuable) ou **pointeur**
  (mutation, gros struct) — ne **mélangez pas** les deux sur un type.
- Comparaison `==` champ par champ, **si tous les champs sont comparables**.
- L'**embedding** compose et **promeut** champs/méthodes, mais **n'est pas de l'héritage** : pas
  de dispatch dynamique. Un nom promu en conflit à la **même profondeur** entre deux embeddings
  est **ambigu** (erreur de compilation) ; à profondeurs différentes, le plus proche l'emporte.
- L'**ordre des champs** influe sur la taille (padding) ; `struct{}` pèse **0 octet**.

## 🔁 Pour aller plus loin

- [Ch. 9 — Interfaces](09-interfaces.md) : le polymorphisme par le comportement (method sets).
- [Ch. 33 — Interfaces & types en profondeur](33-interfaces-profondeur.md) : itab, dispatch.
- [Ch. 35 — `unsafe`](35-unsafe-cgo.md) : `Sizeof`/`Alignof`/`Offsetof`, alignement, `HostLayout`.
- [Ch. 34 — Réflexion](34-reflexion.md) : lecture des **tags** de struct.
- [Ch. 11 — Généricité](11-genericite.md) : structs et méthodes paramétrés.
