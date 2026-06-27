# Ch. 9 — Interfaces (fondamentaux)

> **Objectif** — Abstraire par le **comportement** : déclarer des interfaces, comprendre la
> **satisfaction implicite**, et manipuler les valeurs dynamiques (assertions, type switch).
>
> **Prérequis** — [Ch. 8 — Structs, méthodes & composition](08-structs-methodes.md)

---

## Introduction

Une **interface** décrit ce qu'un type **sait faire** (un ensemble de méthodes), pas ce qu'il
**est**. C'est l'outil de découplage central de Go : une fonction qui accepte une interface
fonctionne avec **tout** type présent ou futur qui en possède les méthodes — sans héritage, sans
déclaration explicite. Les internals (itab, dispatch dynamique) sont au
[Ch. 33](33-interfaces-profondeur.md).

L'exemple complet est dans [`code/ch09-interfaces/`](../code/ch09-interfaces/).

---

## Déclarer une interface

Une interface est une liste de **signatures de méthodes** :

```go
type Shape interface {
	Area() float64
	Perimeter() float64
}
```

## Satisfaction **implicite**

Un type satisfait une interface **dès qu'il en possède toutes les méthodes** — il n'y a **aucun
mot-clé** (`implements`) à écrire, aucune déclaration de lien. C'est le **duck typing** vérifié à
la **compilation**.

```go
type Circle struct{ Radius float64 }

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }

var s Shape = Circle{Radius: 2} // OK : Circle a Area + Perimeter
```

> 💡 **Assertion à la compilation** : pour garantir qu'un type satisfait une interface (et
> obtenir une erreur claire sinon), ajoutez `var _ Shape = Circle{}` ou
> `var _ Shape = (*Circle)(nil)`. Le `_` ne crée aucune variable, c'est juste un contrôle.

## Une valeur d'interface = un couple (type, valeur)

Sous le capot, une variable d'interface stocke **deux** mots : le **type dynamique** (le type
concret rangé dedans) et la **donnée** (la valeur, ou un pointeur vers elle).

```
   var s Shape = Circle{Radius: 2}

   s  +------------------+------------------+
      |  type: Circle    |  data: ---------> { Radius: 2 }
      +------------------+------------------+
         (descripteur)       (la valeur concrète)

   s.Area()  ->  via le type, on trouve et on appelle la bonne méthode (dispatch dynamique)
```

On observe le type dynamique avec `%T` :

```go
fmt.Printf("%T\n", s) // main.Circle
```

## L'interface vide : `any`

Une interface **sans méthode** est satisfaite par **tous** les types. Elle s'écrit `any`
(🆕 1.18), alias officiel de `interface{}` :

```go
var x any = 42
x = "go"
x = Circle{Radius: 1} // tout est assignable à any
```

> ⚠️ `any` fait **perdre** l'information de type : pour réutiliser la valeur, il faut une
> **assertion** ou un **type switch** (ci-dessous). N'utilisez `any` que lorsque le type est
> réellement inconnu (décodage générique, conteneurs hétérogènes) ; sinon préférez un type
> précis ou les **génériques** ([Ch. 11](11-genericite.md)).

## Idiome : « accepter une interface, renvoyer un type concret »

Une fonction gagne en réutilisabilité quand elle **accepte** une interface (le plus petit
contrat suffisant) ; elle reste claire quand elle **renvoie** un type concret.

```go
// Accepte n'importe quel Shape : marche pour Circle, Rect, et tout type futur.
func totalArea(shapes []Shape) float64 {
	var sum float64
	for _, s := range shapes {
		sum += s.Area()
	}
	return sum
}
```

## Type assertion : retrouver le type concret

`x.(T)` extrait la valeur concrète `T` d'une interface. **Deux formes** :

```go
c := s.(Circle)      // forme à 1 résultat : PANIQUE si s ne contient pas un Circle
c, ok := s.(Circle)  // forme comma-ok : ok=false (et c = zero value) si échec, JAMAIS de panique
```

Préférez **toujours** la forme `comma-ok`, sauf si une erreur de type est un bug que vous voulez
voir paniquer.

On peut aussi asserter vers **une autre interface** (« ce type sait-il aussi faire ceci ? ») :

```go
if str, ok := x.(fmt.Stringer); ok {
	fmt.Println(str.String())
}
```

## Type switch : brancher selon le type dynamique

Pour traiter plusieurs types, le **type switch** est plus lisible qu'une cascade d'assertions :

```go
func classify(x any) string {
	switch v := x.(type) {
	case nil:
		return "nil"
	case int:
		return fmt.Sprintf("int:%d", v) // v est un int dans ce cas
	case string:
		return fmt.Sprintf("string:%q", v)
	case Shape: // un cas peut être une INTERFACE
		return fmt.Sprintf("shape:%.2f", v.Area())
	default:
		return fmt.Sprintf("autre:%T", v)
	}
}
```

> 💡 Les cas **concrets** (`int`, `string`) doivent précéder les cas **interface** (`Shape`,
> `error`) : le **premier** cas qui correspond gagne, et un type concret peut satisfaire
> plusieurs interfaces.

## Interfaces idiomatiques de la stdlib

Quelques interfaces que l'on croise et implémente sans cesse :

| Interface      | Méthode(s)                   | Rôle                                          |
| -------------- | ---------------------------- | --------------------------------------------- |
| `fmt.Stringer` | `String() string`            | représentation textuelle (utilisée par `fmt`) |
| `error`        | `Error() string`             | valeur d'erreur ([Ch. 10](10-erreurs.md))     |
| `io.Reader`    | `Read([]byte) (int, error)`  | source d'octets                               |
| `io.Writer`    | `Write([]byte) (int, error)` | destination d'octets                          |

Implémenter `String()` suffit à ce que `fmt` l'utilise **automatiquement** :

```go
func (c Circle) String() string { return fmt.Sprintf("Circle(r=%g)", c.Radius) }
fmt.Println(Circle{Radius: 2}) // -> Circle(r=2)
```

> 💡 **Petites interfaces** : les meilleures interfaces Go ont **une ou deux** méthodes
> (`io.Reader`, `Stringer`). Plus une interface est petite, plus elle est facile à satisfaire et
> à composer. On peut d'ailleurs **composer** des interfaces par embedding :
> `type ReadWriter interface { io.Reader; io.Writer }`.

## ⚠️ Le piège : interface `nil` vs pointeur `nil`

Une interface vaut `nil` **seulement si son type ET sa valeur sont nil**. Y ranger un **pointeur
nil typé** donne une interface… **non nil** (le type, lui, n'est pas nil) :

```
   var x error                       ( type=nil , data=nil )   ->  x == nil   ✅

   var p *ValidationError = nil
   var x error = p                   ( type=*ValidationError , data=nil )  ->  x != nil  ⚠️
```

Le cas le plus courant : renvoyer une variable de type pointeur concret comme `error`.

```go
func bad() error {
	var p *ValidationError // nil
	return p               // renvoie une interface NON nil contenant un pointeur nil !
}
// bad() == nil  vaut  FALSE  -> le code appelant croit qu'il y a une erreur
```

**Parade** : déclarez le type de retour `error` et renvoyez `nil` **littéral** en cas de succès
(ne renvoyez jamais un pointeur concret « nil » par l'interface).

## Method set : valeur ou pointeur (rappel du Ch. 8)

Si une méthode a un **récepteur pointeur**, seul **`*T`** la possède dans son method set —
**`T` (valeur) ne satisfait alors pas** l'interface :

```go
func (e *ValidationError) Error() string { ... } // récepteur POINTEUR

var _ error = &ValidationError{} // OK
var _ error = ValidationError{}  // ERREUR de compilation :
// ValidationError does not implement error (method Error has pointer receiver)
```

---

## 🆕 Go 1.2x

- **1.18** — `any` devient l'alias idiomatique de `interface{}` ; les interfaces gagnent les
  **éléments de type** (`~T`, unions) pour servir de **contraintes** aux génériques
  ([Ch. 11](11-genericite.md)).
- **1.21** — `min`/`max`/`clear` et le package `cmp` réduisent le besoin d'interfaces pour des
  opérations simples de comparaison.

## ⚠️ Pièges

- **Interface nil vs pointeur nil typé** (voir ci-dessus) : LA source de bugs d'erreurs
  fantômes.
- **Assertion à 1 résultat** sur un type incertain → panique. Utilisez `comma-ok`.
- **`any` à tout-va** → on perd le typage statique et la sécurité ; souvent un signe qu'un type
  concret ou un générique conviendrait mieux.
- **Gros contrats** : une interface à 8 méthodes est difficile à implémenter et à mocker.
  Découpez.
- **Récepteur pointeur oublié** : `T` (valeur) ne satisfait pas une interface dont une méthode
  est sur `*T`.

## ⚡ Performance

- Un appel **via interface** est un **dispatch indirect** (par la table de méthodes) : il
  empêche certaines optimisations d'inlining (détail [Ch. 33](33-interfaces-profondeur.md)).
- Convertir une valeur concrète en interface peut **allouer** (boxing) si elle s'échappe sur le
  tas ([Ch. 26](26-allocation-escape.md)).
- Dans le code **chaud**, préférez un type concret ou les **génériques** (résolus à la
  compilation, sans dispatch) quand c'est possible.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch09-interfaces
go test ./ch09-interfaces/...
```

À essayer :

1. Faites passer `Error()` de `*ValidationError` à `ValidationError` (récepteur valeur) et
   observez que `ValidationError{}` satisfait alors `error` (le method set change).
2. Ajoutez un type `Triangle` satisfaisant `Shape` : sans rien changer d'autre, il fonctionne
   dans `totalArea` et `biggest` (puissance de la satisfaction implicite).
3. Écrivez `func describe(x any) string` qui distingue `fmt.Stringer` des autres types via une
   assertion.

---

## 📌 À retenir

- Une **interface** = un ensemble de méthodes ; la satisfaction est **implicite** (aucun
  `implements`), vérifiée à la **compilation**.
- Une valeur d'interface est un couple **(type dynamique, valeur)** ; `%T` révèle le type.
- **Assertion** (`x.(T)`, préférez `comma-ok`) et **type switch** récupèrent le type concret.
- `any` (= `interface{}`) accepte tout mais fait perdre le typage : à réserver aux vrais cas.
- ⚠️ Une interface contenant un **pointeur nil typé** n'est **pas** `nil` — renvoyez `nil`
  littéral.

## 🔁 Pour aller plus loin

- [Ch. 10 — Gestion des erreurs](10-erreurs.md) : `error`, wrapping, `errors.Is`/`As`.
- [Ch. 11 — Généricité](11-genericite.md) : interfaces comme **contraintes** de type.
- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : `eface`/`iface`, itab,
  coût du dispatch.
- [Ch. 34 — Réflexion](34-reflexion.md) : inspecter dynamiquement type et valeur.
