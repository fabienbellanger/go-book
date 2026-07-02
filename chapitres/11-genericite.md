# 11 — Généricité

> **Objectif** — Écrire du code **réutilisable et typé** : fonctions et types **paramétrés par
> un type**, **contraintes**, et les packages génériques `slices`/`maps`/`cmp` — sans le surcoût
> d'une interface.
>
> **Prérequis** — [Ch. 9 — Interfaces](09-interfaces.md) (les contraintes **sont** des
> interfaces), [Ch. 8 — Méthodes](08-structs-methodes.md)

---

## Introduction

Avant les génériques (🆕 **1.18**), réutiliser une fonction pour plusieurs types imposait deux
mauvais choix : la **duplication** (`MaxInt`, `MaxFloat`, `MaxString`…) ou l'**interface vide**
`any`, qui fait **perdre le type** et impose assertions et boxing ([Ch. 9](09-interfaces.md)).

La **généricité** offre une troisième voie : on écrit la logique **une seule fois**, paramétrée
par un **type** que le compilateur **résout à la compilation**. Le résultat est **type-safe** (pas
d'assertion) et **sans dispatch dynamique** (contrairement aux interfaces).

L'exemple complet est dans [`code/ch11-generics/`](../code/ch11-generics/).

---

## Un paramètre de type

On déclare des **paramètres de type** entre crochets `[...]`, **avant** les paramètres normaux.
Chaque paramètre a une **contrainte** (ici `any`, « n'importe quel type ») :

```go
// Map applique f à chaque élément et renvoie un nouveau slice.
// E = type d'entrée, R = type de résultat ; les deux sont libres (any).
func Map[E, R any](s []E, f func(E) R) []R {
	out := make([]R, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}
```

À l'appel, le compilateur **infère** les types — inutile de les écrire :

```go
lengths := Map([]string{"go", "rust"}, func(s string) int { return len(s) })
// E=string, R=int inférés -> lengths == []int{2, 4}
```

L'inférence regarde les **types des arguments fournis** et les **unifie** avec la signature :
`s []E` reçoit un `[]string` → `E=string` ; `f func(E) R` reçoit une fonction qui renvoie `int`
→ `R=int`. C'est une inférence **locale**, fondée uniquement sur les arguments — contrairement à
Rust ou Haskell, elle ne tient compte **ni** du contexte d'appel (le type attendu par
l'appelant) **ni** du corps de la fonction.

> 💡 On **peut** instancier explicitement (`Map[string, int](...)`) mais l'inférence suffit
> presque toujours. On l'écrit surtout quand l'inférence **échoue** — typiquement quand un
> paramètre de type n'apparaît **que dans le type de retour**, donc dans aucun argument :
>
> ```go
> func Zero[T any]() T {
> 	var z T
> 	return z
> }
>
> Zero()       // ❌ "cannot infer T" : aucun argument ne porte T
> Zero[int]()  // ✅ instanciation explicite obligatoire ici
> ```

## Les contraintes **sont** des interfaces

Une **contrainte** décrit ce qu'un type paramètre a le droit de faire. C'est une **interface**,
mais enrichie de deux éléments :

- des **méthodes** (comme une interface classique : « ce type a `String()` ») ;
- des **éléments de type** (🆕 1.18) : une **union** `A | B`, et l'approximation `~T`
  (« tout type dont le **sous-jacent** est `T` »).

```go
// Number : tout type dont le sous-jacent est l'un de ceux-ci.
type Number interface {
	~int | ~int64 | ~float64
}

func Sum[T Number](xs []T) T {
	var acc T // zero value du type T
	for _, x := range xs {
		acc += x // autorisé : la contrainte garantit l'opérateur +
	}
	return acc
}
```

Le `~` est crucial : `~int` accepte aussi un type **défini** sur `int`, comme
`type Celsius int`. Sans le `~`, seul `int` **exact** passerait.

```
   contrainte  ~int | ~float64
   -----------------------------------------------------------------
   int            ✅  exact
   float64        ✅  exact
   Celsius        ✅  type Celsius int     -> sous-jacent int, capté par ~int
   time.Duration  ❌  sous-jacent int64 != int  (int et int64 sont distincts)
   string         ❌  hors de l'union
```

> 📌 Vocabulaire de la spec : l'ensemble des types qui satisfont une contrainte s'appelle son
> **type set**. Une interface classique (méthodes seules) a un type set implicite — « tous les
> types qui implémentent ces méthodes », jamais énuméré. Des **éléments de type** restreignent
> ce type set à une liste fermée et explicite :
>
> ```
>    CONTRAINTE (interface)                 TYPE SET (ce que ça autorise)
>    interface {
>        ~int | ~int64 | ~float64     -->   { int, int64, float64, Celsius, ... }
>    }                                       (tout type DÉRIVÉ via ~, en plus des types exacts)
> ```
>
> C'est cette fermeture qui permet au compilateur de garantir que `+` est applicable à `T` :
> il vérifie que **chaque** type du type set supporte l'opération, une fois pour toutes.

> 💡 La contrainte **`any`** (= `interface{}`) n'autorise **aucune** opération particulière : on
> peut seulement passer la valeur, la stocker, la comparer à `nil`. Pour `+`, `<`, `==`, il faut
> une contrainte plus précise.

## `comparable` : les types qui supportent `==`

`comparable` est une contrainte **prédéclarée** : elle regroupe les types utilisables avec `==` /
`!=` (donc valides comme **clés de map**, [Ch. 7](07-maps-strings.md)). Elle ne s'exprime pas
avec `|`/`~` comme une union ordinaire, car son type set n'est **pas une liste fermée** : il
couvre tous les types de base comparables (numériques, `string`, `bool`, pointeurs, channels…)
**et** tout struct ou array **dont tous les champs/éléments sont eux-mêmes comparables** — une
règle récursive, pas une énumération qu'on pourrait écrire à la main.

```go
func Index[T comparable](s []T, target T) int {
	for i, v := range s {
		if v == target { // == autorisé grâce à comparable
			return i
		}
	}
	return -1
}
```

> ⚠️ Les **slices**, **maps** et **fonctions** ne sont **pas** `comparable` ([Ch. 8](08-structs-methodes.md)) :
> un type qui en contient ne satisfait pas `comparable`. Depuis 1.20, une **interface** satisfait
> `comparable`, mais comparer deux interfaces de type dynamique non comparable **panique** à
> l'exécution.

## Types génériques

Un **type** peut aussi être paramétré. Les paramètres se reportent sur ses **méthodes** :

```go
type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false // pile vide : on renvoie le zero value de T
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}
```

```go
var s Stack[int]   // instanciation : T = int
s.Push(1)
s.Push(2)
top, ok := s.Pop() // top == 2, ok == true
```

> ⚠️ Une **méthode** ne peut **pas** introduire ses **propres** paramètres de type : seuls ceux du
> **type** (ou de la fonction) sont disponibles. `func (s *Stack[T]) MapTo[R any](...)` est
> **interdit**. Quand il faut un nouveau type, écrivez une **fonction** générique libre.

## La boîte à outils : `slices`, `maps`, `cmp` (🆕 1.21)

La stdlib fournit l'essentiel **déjà écrit** et générique. Inutile de réimplémenter `Max`,
`Contains`, `Sort` :

| Package  | Exemples                                                                                |
| -------- | --------------------------------------------------------------------------------------- |
| `slices` | `Sort`, `SortFunc`, `Contains`, `Index`, `Max`, `Min`, `Equal`, `BinarySearch`, `Clone` |
| `maps`   | `Keys`, `Values` (itérateurs, [Ch. 18](18-iterateurs.md)), `Clone`, `Equal`             |
| `cmp`    | `Compare`, `Less`, `Or`, et la contrainte **`cmp.Ordered`**                             |

```go
nums := []int{3, 1, 2}
slices.Sort(nums)              // [1 2 3]
fmt.Println(slices.Max(nums))  // 3

// cmp.Ordered = tous les types ordonnables (entiers, flottants, strings).
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// cmp.Or renvoie le premier argument non-zero : idéal pour trier par clés en cascade.
slices.SortFunc(people, func(a, b Person) int {
	return cmp.Or(cmp.Compare(a.Age, b.Age), cmp.Compare(a.Name, b.Name))
})
```

> 💡 **Avant d'écrire une fonction générique, cherchez-la dans `slices`/`maps`/`cmp`.** Elle y est
> souvent déjà, testée et optimisée.

## Comment le compilateur instancie : _GC-shape stenciling_

Go n'est ni du **monomorphisme** total à la C++ (une copie par type, code volumineux), ni du
**boxing** systématique à la Java (tout passe par des pointeurs). Il **sténcile** une copie du
code **par _forme mémoire_ (GC shape)** : tous les types **pointeurs** partagent **une seule**
instanciation, les valeurs de même layout en partagent d'autres. Les copies partagées reçoivent
un **dictionnaire** (table décrivant le type réel) en argument caché.

```
   Sum[int]        \
   Sum[int64]       >  formes "valeur" distinctes -> sténciles dédiés
   Sum[float64]    /

   Stack[*User]    \
   Stack[*Order]    >  même forme "pointeur"      -> UN sténcile partagé + dictionnaire
   Stack[*Conn]    /
```

| Approche                     | Code généré                               | Coût binaire                                                            | Coût runtime                                                   |
| ---------------------------- | ----------------------------------------- | ----------------------------------------------------------------------- | -------------------------------------------------------------- |
| **C++** (templates)          | à la compilation, par unité de traduction | une copie **par type** instancié (le linker dédoublonne les identiques) | quasi nul — code entièrement spécialisé, inlinable             |
| **Java** (type erasure)      | une seule fois, types primitifs effacés   | minimal — un seul bytecode pour tous les types                          | **boxing** des primitifs + cast caché à chaque accès           |
| **Go** (GC-shape stenciling) | à la compilation, par **forme mémoire**   | une copie par forme valeur ; pointeurs mutualisés en un seul sténcile   | dictionnaire passé en argument caché pour les formes partagées |

Conséquence pratique : **type-safe et sans dispatch dynamique**, mais le partage par forme a un
coût (le dictionnaire). Détails et stratégie d'optimisation au
[Ch. 39](39-compilation-inlining-pgo.md).

## Quand **ne pas** utiliser les génériques

Les génériques ne remplacent **ni** les interfaces **ni** les types concrets :

- **Une seule implémentation** ? Un **type concret** suffit. Ne généralisez pas « au cas où ».
- **Comportement** différent par type (polymorphisme à l'exécution, plugins) ? Une **interface**
  ([Ch. 9](09-interfaces.md)) est le bon outil — un `io.Writer` n'a pas besoin d'être générique.
- **Vous appelez des méthodes** sur le paramètre sans contrainte naturelle ? C'est probablement
  une **interface** déguisée.
- Règle de Go : _« Write code, not types. »_ Écrivez le code concret d'abord ; n'extrayez un
  paramètre de type que lorsque la **duplication réelle** apparaît.

---

## 🆕 Go 1.2x

- **1.18** — naissance des génériques : paramètres de type, contraintes, éléments de type
  (`~T`, unions `|`), `any` alias de `interface{}`.
- **1.21** — packages **`slices`**, **`maps`**, **`cmp`** ; builtins génériques `min`/`max`/`clear`
  ([Ch. 3](03-variables-constantes-types.md)). Inférence de type sensiblement améliorée. Avant
  cette version, seul le module externe `golang.org/x/exp/constraints` fournissait des
  contraintes comme `constraints.Ordered` — si vous le croisez encore dans du code existant ou
  un tutoriel, préférez désormais `cmp.Ordered` (stdlib, strictement équivalent).
- **1.24** — **alias de type génériques** : `type Set[T comparable] = map[T]struct{}` est désormais
  légal (un alias peut porter des paramètres de type).
- **1.26** — **contraintes auto-référentielles** : une contrainte peut se nommer elle-même, ce qui
  permet le motif « F-bounded » :

```go
// A doit avoir une méthode Add prenant et renvoyant son PROPRE type.
type Adder[A Adder[A]] interface {
	Add(A) A
}

func SumAll[A Adder[A]](xs []A) A {
	var acc A
	for _, x := range xs {
		acc = acc.Add(x)
	}
	return acc
}
```

## ⚠️ Pièges

- **Méthode avec ses propres paramètres de type** → interdit. Seuls les paramètres du type/de la
  fonction sont disponibles ; sinon, une **fonction** libre.
- **`any` quand on a besoin d'opérateurs** : `+`, `<`, `==` exigent une contrainte (`Number`,
  `cmp.Ordered`, `comparable`). `any` n'autorise rien de tout cela.
- **Oublier le `~`** : `int | float64` rejette `type Celsius int`. Utilisez `~int | ~float64` pour
  capter les types **dérivés**.
- **Sur-généraliser** : un paramètre de type utilisé une seule fois, ou une « usine à abstraction »
  illisible. Préférez le concret tant que la duplication n'existe pas.
- **`comparable` avec des interfaces** : autorisé depuis 1.20, mais `==` **panique** si les types
  dynamiques ne sont pas comparables.
- **Alias générique sans méthode propre** : `type Set[T comparable] = map[T]struct{}` (1.24)
  reste un **alias**, pas un type distinct. Lui attacher une méthode est refusé à la
  compilation (`cannot define new methods on generic alias type`). Pour des méthodes propres,
  il faut une **définition** (`type Set[T comparable] struct{ ... }`), pas un alias.

## ⚡ Performance

- Un appel **générique** est résolu à la **compilation** : pas de dispatch indirect, et l'addition
  d'un `Sum[int]` s'inline comme du code concret. Une approche **interface** équivalente paie le
  **dispatch dynamique** à chaque élément.
- Mesure indicative (somme de 1000 entiers, `go test -bench`) :

```
   BenchmarkGeneric     477 ns/op     0 allocs/op
   BenchmarkIface      1878 ns/op     0 allocs/op    (3 à 6x plus lent selon la machine :
                                                      un dispatch indirect par élément)
```

- Le partage par **forme** (pointeurs) ajoute un **dictionnaire** : marginal, mais réel. À
  l'inverse, chaque forme **valeur** distincte (`Stack[int]` et `Stack[string]` ne partagent
  rien) duplique le code généré — un coût de compilation et de taille de binaire qui n'existe
  pas avec une unique implémentation par interface. Pour le code ultra-chaud sur un seul type,
  le concret reste imbattable ([Ch. 39](39-compilation-inlining-pgo.md)).
- Génériques **vs** interface : choisissez les génériques quand le **type** varie mais la **logique**
  est identique ; l'interface quand le **comportement** varie ([Ch. 33](33-interfaces-profondeur.md)).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch11-generics
go test ./ch11-generics/...
go test -bench=. -benchmem ./ch11-generics/...
```

À essayer :

1. Retirez les `~` de la contrainte `Number` et observez l'erreur de compilation dès qu'on passe
   un `type Celsius float64`.
2. Ajoutez une méthode `func (s *Stack[T]) Map[R any](...)` et constatez le refus du compilateur
   (les méthodes ne sont pas paramétrables).
3. Réécrivez `Index` avec `slices.Index` et vérifiez que le test passe toujours.
4. Écrivez `func Zero[T any]() T { var z T; return z }` puis appelez `Zero()` sans préciser `T` :
   observez le message `cannot infer T`, puis corrigez avec `Zero[int]()`.

---

## 📌 À retenir

- Un **paramètre de type** `[T C]` rend une fonction/un type réutilisable **et** typé, sans `any`
  ni assertion ; le compilateur **infère** souvent `T`.
- Une **contrainte** est une interface enrichie d'**éléments de type** (`~T`, unions `|`) ;
  `comparable` = types utilisables avec `==`.
- La stdlib `slices`/`maps`/`cmp` (1.21) couvre déjà l'essentiel — cherchez-y avant d'écrire.
- Instanciation par **GC-shape stenciling** : type-safe, sans dispatch, plusieurs fois plus rapide
  qu'une interface sur ce genre de boucle.
- Génériques pour faire **varier le type**, interfaces pour faire **varier le comportement** ;
  ne généralisez pas avant que la duplication existe.

## 🔁 Pour aller plus loin

- [Ch. 18 — Itérateurs](18-iterateurs.md) : `maps.Keys`/`slices.Values` renvoient des `iter.Seq`.
- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : le coût du dispatch que les
  génériques évitent.
- [Ch. 39 — Compilation & inlining](39-compilation-inlining-pgo.md) : GC-shape stenciling,
  dictionnaires, inlining des instanciations.
- [Projet 4 — Bibliothèque générique](../projets/4-lib-generique/) : ensemble, file de priorité, cache LRU.
