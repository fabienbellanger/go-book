# 9 — Interfaces (fondamentaux)

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

> 💡 **Convention de nommage** : une interface à **une seule méthode** se nomme souvent en
> suffixant le nom de la méthode par `-er` (`Reader` pour `Read`, `Writer` pour `Write`,
> `Stringer` pour `String`, `Closer` pour `Close`). Une interface à plusieurs méthodes décrit
> plutôt un **rôle** (`Shape`, `Handler`). Rien d'imposé par le compilateur, mais largement
> suivi dans la stdlib et le code idiomatique.

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

Cette absence de déclaration n'est pas un détail cosmétique : `Circle` n'a même pas besoin de
**connaître** ni d'**importer** le paquet où `Shape` est défini. C'est l'inverse d'un
`implements` Java/C# (lien explicite et bilatéral, déclaré côté type) : ici le lien est
**unilatéral et a posteriori**. Un paquet `geo` peut définir `Circle` sans la moindre interface
en tête, et un paquet `render` totalement indépendant peut définir `Shape` et l'utiliser avec
`Circle` sans que `geo` en sache rien. C'est ce qui permet de découpler les paquets et de définir
une interface **côté consommateur**, taillée a posteriori sur ses propres besoins plutôt
qu'anticipée côté producteur.

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

La stdlib applique systématiquement la seconde moitié de l'idiome : `bytes.NewReader` renvoie un
`*bytes.Reader` **concret**, pas un `io.Reader`, bien qu'il soit le plus souvent utilisé comme
tel ailleurs. Renvoyer le type concret laisse l'appelant profiter de **toutes** ses méthodes
(`Len()`, `Seek()`...), pas seulement du sous-ensemble imposé par l'interface, et évite une
indirection inutile (⚡ voir plus bas).

> 💡 **Exception assumée** : `error` est le contre-exemple de cette règle — les fonctions
> renvoient systématiquement l'interface `error`, jamais un type d'erreur concret, car
> l'appelant ne doit pas avoir à connaître **laquelle** des erreurs possibles a été produite
> (détail [Ch. 10](10-erreurs.md)). La règle par défaut reste « renvoyer un type concret » ; on
> s'en écarte quand cacher le type est précisément le but recherché.

## Type assertion : retrouver le type concret

`x.(T)` extrait la valeur concrète `T` d'une interface. **Deux formes** :

```go
c := s.(Circle)      // forme à 1 résultat : PANIQUE si s ne contient pas un Circle
c, ok := s.(Circle)  // forme comma-ok : ok=false (et c = zero value) si échec, JAMAIS de panique
```

Préférez **toujours** la forme `comma-ok`, sauf si une erreur de type est un bug que vous voulez
voir paniquer. Dans ce cas, le message de panique nomme les deux types en cause, ce qui en fait
un diagnostic directement exploitable :

```
panic: interface conversion: main.Shape is main.Circle, not main.Rect
```

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

> 💡 Un `case` peut lister **plusieurs types** séparés par des virgules (`case int, int64:`) —
> mais cela change le type de `v` dans ce cas :
>
> - **un seul type** dans le `case` → `v` a **exactement ce type** (dans `case int:`, `v` est un
>   `int`, on peut donc écrire `v + 1`) ;
> - **plusieurs types** dans le `case` → le compilateur ne peut pas deviner lequel choisir, donc
>   `v` conserve le **type statique de l'expression** du switch (ici `any`), et **non** l'un des
>   types listés. Pour manipuler la valeur comme un entier, il faut alors une **nouvelle
>   assertion** à l'intérieur du `case`.
>
> ```go
> switch v := x.(type) { // x est de type any
> case int: // un seul type
> 	_ = v + 1 // OK : v est un int
> case int8, int16: // plusieurs types
> 	_ = v // v est de type any ici ; `v + 1` ne compilerait pas
> }
> ```

## Interfaces idiomatiques de la stdlib

Quelques interfaces que l'on croise et implémente sans cesse :

| Interface      | Méthode(s)                   | Rôle                                          |
| -------------- | ---------------------------- | --------------------------------------------- |
| `fmt.Stringer` | `String() string`            | représentation textuelle (utilisée par `fmt`) |
| `error`        | `Error() string`             | valeur d'erreur ([Ch. 10](10-erreurs.md))     |
| `io.Reader`    | `Read([]byte) (int, error)`  | source d'octets                               |
| `io.Writer`    | `Write([]byte) (int, error)` | destination d'octets                          |
| `io.Closer`    | `Close() error`              | libération de ressource (fichier, connexion)  |

Implémenter `String()` suffit à ce que `fmt` l'utilise **automatiquement** :

```go
func (c Circle) String() string { return fmt.Sprintf("Circle(r=%g)", c.Radius) }
fmt.Println(Circle{Radius: 2}) // -> Circle(r=2)
```

> 💡 **Petites interfaces** : les meilleures interfaces Go ont **une ou deux** méthodes
> (`io.Reader`, `Stringer`). Plus une interface est petite, plus elle est facile à satisfaire et
> à composer. On peut d'ailleurs **composer** des interfaces par embedding — c'est ainsi que la
> stdlib construit `io.ReadWriteCloser` à partir de `io.Reader`, `io.Writer` et `io.Closer` :
> `type ReadWriter interface { io.Reader; io.Writer }`.

## ⚠️ Le piège : interface `nil` vs pointeur `nil`

**Modèle mental.** Une valeur d'interface n'est pas une simple case : c'est une **paire de deux
champs**, `(type dynamique, valeur)`. Le premier dit _quel type concret_ est rangé dedans, le
second _quelle donnée_ (🔁 Ch. 33 pour le layout `eface`/`iface`).

Une interface vaut `nil` **uniquement quand ses DEUX champs sont vides** — c'est-à-dire quand elle
ne contient **aucun type**. D'où le piège : un **pointeur nil typé** a beau pointer sur « rien », il
porte quand même un **type** (`*ValidationError`). Le ranger dans une interface remplit donc le
champ _type_ — l'interface n'est **plus vide**, donc **plus `nil`** :

```
                                type dynamique        valeur
                                --------------        ------
  var x error                   nil                   nil      ->  x == nil   ✅  (les deux vides)

  var p *ValidationError = nil
  var x error = p               *ValidationError      nil      ->  x != nil   ⚠️  (le type n'est PAS vide)
```

> 💡 **Analogie** : une interface est un **colis étiqueté**. Un `nil` interface = un colis **sans
> étiquette et vide**. Un pointeur nil typé = un colis **étiqueté `*ValidationError`** mais vide à
> l'intérieur. Comparer à `nil` teste l'**étiquette**, pas le contenu : dès qu'il y a une étiquette,
> `== nil` est **faux**.

```
  « x == nil ? » teste l'ÉTIQUETTE (le champ type), pas le contenu du colis :

  (1) interface nil        var x error
      +----------------------------------+
      |  type   : (aucun)                |
      |  valeur : (vide)                 |
      +----------------------------------+
      ->  x == nil  vaut  true    (colis SANS étiquette : vraiment vide)

  (2) pointeur nil typé    var p *ValidationError = nil ; var x error = p
      +----------------------------------+
      |  type   : *ValidationError       |
      |  valeur : nil                    |
      +----------------------------------+
      ->  x == nil  vaut  FALSE   (colis AVEC étiquette mais vide : LE piège)
```

**Où ça mord.** Le cas classique : une fonction déclare un retour `error` mais renvoie une variable
de **type pointeur concret**. Même quand ce pointeur est `nil`, l'interface renvoyée porte son type
et devient **non nil** :

```go
func bad() error {
	var p *ValidationError // p vaut nil
	return p               // range (*ValidationError, nil) dans l'interface : NON nil !
}

func good() error {
	return nil // renvoie une interface VRAIMENT nil (type ET valeur vides)
}
```

Résultat : `bad() == nil` vaut **`false`**, alors que `good() == nil` vaut `true`. Le test d'erreur
habituel côté appelant **se trompe** donc avec `bad()` :

```go
if err := bad(); err != nil {
	// On entre ICI alors qu'il n'y a PAS d'erreur (err != nil vaut true).
	// Pire : err.Error() déréférence un pointeur nil -> panique.
	log.Fatal(err)
}
```

> 🧪 Démo exécutable : comparez `typedNilError()` (le mauvais cas) et `validateAge()` (le bon) dans
> [`code/ch09-interfaces/errors.go`](../code/ch09-interfaces/errors.go). `go run ./ch09-interfaces`
> affiche `typedNilError()==nil ? false` mais `validateAge(30) == nil ? true`.

**Parade** — deux règles simples :

- Le type de retour est **`error`** (l'interface), **jamais** un `*MonErreur` concret.
- En cas de succès, renvoyez le **`nil` littéral** (`return nil`), pas une variable de type pointeur
  qui « se trouve » être nil.

Ainsi le champ _type_ de l'interface reste vide, et `err != nil` ne se déclenche que pour une
**vraie** erreur.

## Method set : valeur ou pointeur (rappel du Ch. 8)

Si une méthode a un **récepteur pointeur**, seul **`*T`** la possède dans son method set —
**`T` (valeur) ne satisfait alors pas** l'interface :

```go
func (e *ValidationError) Error() string { ... } // récepteur POINTEUR

var _ error = &ValidationError{} // OK
var _ error = ValidationError{}  // ERREUR de compilation :
// ValidationError does not implement error (method Error has pointer receiver)
```

**Pourquoi cette restriction ?** Sur une variable **adressable**, `v.M()` est réécrit
silencieusement en `(&v).M()` quand `M` a un récepteur pointeur (🔁 Ch. 8). Mais une fois `v`
rangé dans une interface, seule la **copie** stockée dans l'interface subsiste : il n'y a plus
de variable adressable à laquelle prendre l'adresse (la valeur d'origine a pu disparaître,
être un élément de map, un retour de fonction...). Le compilateur ne peut donc pas garantir que
prendre l'adresse de cette copie a un sens, et exclut ces méthodes du method set de `T`. Stocker
directement un `*T` dans l'interface contourne le problème : l'adresse existe déjà.

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
- **Comparer deux interfaces dont le type dynamique est incomparable** (slice, map, fonction)
  → panique à l'exécution, y compris via `==` ou comme clé de `map[any]...` :
  `panic: runtime error: comparing uncomparable type []int`. Le compilateur ne peut pas le
  détecter à l'avance puisque le type dynamique n'est connu qu'à l'exécution.

## ⚡ Performance

Une interface échange un **petit coût à l'exécution** contre du **découplage**. Dans la plupart du
code ce coût est négligeable et la souplesse l'emporte largement ; il ne devient mesurable que sur
les **chemins très chauds** (appelés des millions de fois). Voici les quatre mécanismes à connaître.

**1. Un appel via interface est un dispatch _indirect_.** Sur un type concret, `c.Area()` est un
appel **direct** : le compilateur connaît la fonction exacte et peut l'**inliner** (recopier son
corps sur place, sans appel). À travers une interface, `s.Area()` passe par l'**itab** — la petite
table de méthodes rangée dans la valeur d'interface (🔁 [Ch. 33](33-interfaces-profondeur.md)) : le
runtime y **lit** l'adresse de `Area`, puis l'appelle **indirectement**. Cette indirection coûte un
chargement + un saut, mais surtout elle est **opaque** à l'optimiseur : il ignore quel code concret
tournera, donc **pas d'inlining ni d'optimisation** au travers de l'appel. (Depuis 1.21, la **PGO**
sait _dévirtualiser_ les appels chauds dominés par un seul type — 🔁 [Ch. 39](39-compilation-inlining-pgo.md).)

**2. Passer une valeur en interface peut _allouer_ (boxing).** Une valeur d'interface range un
**pointeur** vers la donnée. Y mettre une valeur concrète peut donc la **copier sur le tas**
(_boxing_) si elle s'échappe — une **allocation**, souvent invisible dans le code : `var x any = 42`,
une `error` renvoyée, les `...any` passés à `fmt.Println`. Anodin ponctuellement, coûteux dans une
boucle serrée (pression sur le GC). 🔁 [Ch. 26](26-allocation-escape.md) pour l'escape analysis.
(Go met en cache les petits entiers 0–255 : eux n'allouent pas.)

**3. Une assertion de type n'est PAS de la « réflexion ».** C'est la confusion la plus fréquente.
`x.(T)` (et le `type switch`) se compile en une simple **comparaison du descripteur de type** rangé
dans l'interface (ou de son itab) avec le type attendu : une ou deux comparaisons de pointeurs et un
branchement, de l'ordre de la **nanoseconde**. Cela **n'a rien à voir** avec le paquet `reflect`,
qui est une API d'introspection **dynamique** bien plus lourde (🔁 [Ch. 34](34-reflexion.md)).
Conséquence pratique : **utilisez les assertions et les type switches sans hésiter** — ce n'est pas
un poste de coût, et les éviter « par prudence » complique le code pour rien.

**4. Dans le code chaud, préférez un type concret ou les génériques.** Quand un appel tourne des
millions de fois et que l'abstraction n'apporte rien de concret, deux façons d'éviter le dispatch
indirect :

- **Type concret** — pas d'interface du tout : les appels sont directs et inlinables. À privilégier
  quand vous n'avez, de fait, qu'**un seul** type.
- **Génériques** ([Ch. 11](11-genericite.md)) — une fonction comme `func Sum[T Number](xs []T) T`
  est **spécialisée à la compilation** : les appels sont **résolus statiquement** (pas d'itab),
  donc inlinables et sans boxing. On garde le « écrire une fois, marche pour plusieurs types » de
  l'interface, mais **fixé à la compilation** plutôt que résolu à l'exécution.

> ⚡ **À mesurer, pas à supposer.** Le gain des génériques n'est **pas automatique** : pour les
> types **pointeur**, le compilateur mutualise une seule instanciation par « forme GC » via un
> **dictionnaire** passé en paramètre caché — ce qui peut **annuler** l'avantage, voire rendre le
> générique **plus lent** qu'une interface (démonstration chiffrée, 🔁
> [Annexe E](../annexes/E-demonstrations-benchmarks.md)). Règle de conduite : **interface** pour
> découpler (I/O, stockage, handlers), **génériques** pour les algorithmes génériques chauds
> (conteneurs, calcul numérique), **type concret** sinon — et **profilez avant d'optimiser**
> (🔁 [Ch. 40](40-methodologie-performance.md)). Ne remplacez jamais une interface par un générique
> sur une simple intuition de performance.

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
4. Comparez deux valeurs `any` contenant chacune un `[]int` avec `==` et observez la panique
   `comparing uncomparable type`.

---

## 📌 À retenir

- Une **interface** = un ensemble de méthodes ; la satisfaction est **implicite** (aucun
  `implements`), vérifiée à la **compilation**.
- Une valeur d'interface est un couple **(type dynamique, valeur)** ; `%T` révèle le type.
- **Assertion** (`x.(T)`, préférez `comma-ok`) et **type switch** récupèrent le type concret.
- `any` (= `interface{}`) accepte tout mais fait perdre le typage : à réserver aux vrais cas.
- ⚠️ Une interface contenant un **pointeur nil typé** n'est **pas** `nil` — renvoyez `nil`
  littéral.
- Le **method set** détermine qui satisfait quoi : un récepteur pointeur exclut le type valeur
  de la satisfaction (utilisez `*T`).

## 🔁 Pour aller plus loin

- [Ch. 10 — Gestion des erreurs](10-erreurs.md) : `error`, wrapping, `errors.Is`/`As`.
- [Ch. 11 — Généricité](11-genericite.md) : interfaces comme **contraintes** de type.
- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : `eface`/`iface`, itab,
  coût du dispatch.
- [Ch. 34 — Réflexion](34-reflexion.md) : inspecter dynamiquement type et valeur.
