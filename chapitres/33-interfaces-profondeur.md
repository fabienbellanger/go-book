# 33 — Interfaces & système de types en profondeur

> **Objectif** — Comprendre la représentation d'une interface — **`eface`** (interface vide) et
> **`iface`** (`itab` + données) —, le rôle de l'**`itab`** et de son cache, le **dispatch dynamique**
> et son coût réel (inlining perdu, pas l'appel indirect), le **boxing** et ses allocations, le piège de
> l'**interface nil non-nil**, et `reflect.TypeAssert`.
>
> **Prérequis** — [Ch. 9](09-interfaces.md), [Ch. 8](08-structs-methodes.md)

---

## Introduction

Le [Ch. 9](09-interfaces.md) a présenté les interfaces comme des **contrats de comportement**. Sous le
capot, une valeur d'interface est un **couple de deux mots** : _quel type concret_ et _où sont les
données_. Ces deux mots expliquent le dispatch dynamique, le coût d'une conversion, l'allocation
surprise d'un `int` mis dans un `any`, et le bug le plus déroutant de Go — une interface « nil » qui ne
l'est pas. Code dans [`code/ch33-interfaces-profondeur/`](../code/ch33-interfaces-profondeur/).

---

## Deux représentations : `eface` et `iface`

Une interface fait **16 octets** (2 mots) sur une machine 64 bits, quelle que soit la valeur qu'elle
porte. Mais le **premier** mot diffère selon que l'interface est vide ou non :

```
  eface (interface vide : any)         iface (interface à méthodes : Shape)
  +----------+----------+              +----------+----------+
  | *_type   |  data    |              |  *itab   |  data    |
  +----------+----------+              +----+-----+----------+
   type concret  -> valeur                  |          -> valeur concrète
                                            v
                                       itab (mis en cache) :
                                       +--------------------+
                                       | type interface     |
                                       | type concret       |
                                       | [ &méthodes ]      |  <- pointeurs de fonctions
                                       +--------------------+
```

- **`eface`** (`any`) : `(*_type, data)` — juste le **type concret** et un pointeur vers la valeur.
- **`iface`** (interface à méthodes) : `(*itab, data)` — l'**`itab`** relie le type concret au type
  interface et porte la **table des méthodes**.

```go
var empty any = 42
var shape Shape = Circle{R: 2}
// Sizeof(any)=16, Sizeof(Shape)=16  (2 mots chacun)
```

## L'`itab` et le dispatch dynamique

L'**`itab`** (interface table) est construit **une fois** par couple (type concret, type interface),
puis **mis en cache** par le runtime — les conversions suivantes le réutilisent. Il contient les
**pointeurs vers les méthodes** du type concret qui satisfont l'interface.

Appeler `s.Area()` sur une `Shape` fait un **dispatch dynamique** : aller chercher le pointeur de
`Area` dans l'`itab`, puis sauter dessus.

```go
// code/ch33-interfaces-profondeur/iface.go
func TotalArea(shapes []Shape) float64 {
	var sum float64
	for _, s := range shapes {
		sum += s.Area() // méthode résolue via l'itab du type concret
	}
	return sum
}
```

## Le coût réel : l'inlining perdu

On croit souvent que « l'appel indirect coûte cher ». En pratique, sur un CPU moderne, un dispatch
**monomorphe** est presque gratuit (le prédicteur de branche l'anticipe). Le **vrai** coût est que
l'appel via interface **empêche l'inlining** de la méthode — et toutes les optimisations qui en
découlent. Mesuré (1000 formes, `Area()` trivial) :

| Variante               | ns/op    | allocs/op |
| ---------------------- | -------- | --------- |
| dispatch via interface | **4562** | 0         |
| appel concret (inliné) | **1716** | 0         |

**2,7× plus rapide** en concret — non pas parce que l'appel indirect est lent, mais parce que `Area()`
est **inliné** et la boucle optimisée. 💡 Sur un chemin chaud avec une méthode minuscule, préférez un
type concret ou les **génériques** ([Ch. 11](11-genericite.md)) ; ailleurs, le dispatch est négligeable.

## Le boxing : quand une conversion alloue

Ranger une valeur dans une interface, c'est la **boxer** : l'interface doit pointer vers les données.
Si la valeur ne tient pas déjà sous forme de pointeur, le runtime **l'alloue sur le tas**
([Ch. 26](26-allocation-escape.md)). Exception : les petits entiers **0 à 255** sont **mis en cache** par
le runtime (pointeurs pré-alloués) :

```go
// code/ch33-interfaces-profondeur/iface.go
var sink any
func BoxValue(v any) { sink = v }
```

| Valeur boxée dans `any`     | alloc/op |
| --------------------------- | -------- |
| `int` de 0 à 255 (en cache) | **0**    |
| `int` quelconque (> 255)    | **1**    |

C'est pourquoi `fmt.Println(x)` (qui prend des `...any`) peut **allouer** : chaque argument est boxé.
Sur le chemin chaud, `-gcflags=-m` révèle ces boxings (`x escapes to heap`).

## Le piège : interface nil non-nil

Le bug le plus déroutant de Go. Une interface vaut `nil` **seulement si ses deux mots sont nuls**
(type **et** données). Mettre un **pointeur nil typé** dans une interface lui donne un **type** : elle
n'est donc **pas** `nil`.

```go
// code/ch33-interfaces-profondeur/iface.go
func FailBuggy(ok bool) error {
	var e *myError // pointeur nil
	if !ok {
		e = &myError{"échec"}
	}
	return e // PIEGE : si e==nil, l'interface error renvoyée n'est PAS nil
}
```

```
$ go run ./ch33-interfaces-profondeur
FailBuggy(true)   == nil ? false   (PIEGE : on attendait true)
FailCorrect(true) == nil ? true    (correct)
```

L'interface renvoyée porte `(*myError, nil)` : type **non nul**, donc `err != nil` est vrai même sans
erreur. La parade : renvoyer **explicitement** `nil` (`FailCorrect`), jamais une variable de type
pointeur concret. ⚠️ Ne déclarez pas vos fonctions avec un type d'erreur **concret** en retour ;
renvoyez `error`.

## Assertions, type switch & `reflect.TypeAssert`

`v, ok := s.(Circle)` (assertion) et le `type switch` ([Ch. 9](09-interfaces.md)) lisent l'`itab`/`_type`
**sans allocation**. Depuis **Go 1.25**, **`reflect.TypeAssert[T]`** extrait un type concret d'une
`reflect.Value` **sans** repasser par `Value.Interface()` (qui re-boxe) :

```go
// code/ch33-interfaces-profondeur/iface.go
func AsCircle(s Shape) (Circle, bool) {
	return reflect.TypeAssert[Circle](reflect.ValueOf(s)) // pas de re-boxing
}
```

---

## 🆕 Go 1.2x

- **1.25** — **`reflect.TypeAssert[T](v)`** : assertion typée depuis une `reflect.Value`, **sans
  l'allocation** de `Value.Interface().(T)`. Utile dans les décodeurs réflexifs ([Ch. 34](34-reflexion.md)).
- **continuité** — l'`itab` est **mis en cache** par le runtime : la première conversion (type, interface)
  le construit, les suivantes sont quasi gratuites.

## ⚠️ Pièges

- **Interface nil non-nil** — renvoyer un pointeur concret (même nil) typé en `error`. Renvoyez `error`
  et `nil` **explicitement**.
- **Boxing sur le chemin chaud** — passer des valeurs en `any` (logs, `fmt`, `[]any`) alloue. Mesurez
  avec `-gcflags=-m` / `-benchmem`.
- **Croire que « interface = lent »** — l'appel indirect monomorphe est bon marché ; c'est l'**inlining
  perdu** qui compte, et seulement pour des méthodes minuscules très chaudes.
- **Grosse interface en valeur** — `data` pointe vers la valeur ; une struct volumineuse mise en
  interface est **copiée** (et souvent boxée).

## ⚡ Performance

- Pour un point chaud polymorphe sur un type unique, un **type concret** ou un **générique**
  ([Ch. 11](11-genericite.md)) évite le dispatch et autorise l'inlining.
- **Réutilisez** les valeurs boxées plutôt que de reboxer en boucle ; évitez les `[]any` intermédiaires.
- `reflect.TypeAssert` (1.25) supprime une allocation dans le code réflexif.
- 🔁 [Ch. 26](26-allocation-escape.md) (boxing = allocation) et [Ch. 39](39-compilation-inlining-pgo.md)
  (inlining, dévirtualisation).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch33-interfaces-profondeur
go test ./ch33-interfaces-profondeur/...
go test -bench=. -benchmem -run=^$ ./ch33-interfaces-profondeur/...
```

À essayer :

1. `go build -gcflags=-m ./ch33-interfaces-profondeur` : repérez les `escapes to heap` du boxing.
2. Ajoutez un 3ᵉ type à `Shape` et observez que `TotalArea` n'a **pas** besoin de changer (dispatch).
3. Remplacez `FailBuggy` par `FailCorrect` dans un appelant et constatez la disparition du bug.

---

## 📌 À retenir

- Une interface = **2 mots** (16 o). **`eface`** (`any`) = `(*_type, data)` ; **`iface`** (à méthodes) =
  `(*itab, data)`.
- L'**`itab`** relie type concret et interface, porte la **table des méthodes**, et est **mis en cache**
  par le runtime.
- Le **dispatch dynamique** coûte surtout par l'**inlining perdu** (ici 2,7×), pas par l'appel indirect.
- Le **boxing** alloue (1/op) sauf petits entiers 0..255 (cache) ; passer des valeurs en `any` (logs,
  `fmt`) peut allouer.
- **Interface nil non-nil** : un pointeur nil typé dans une interface **n'est pas** `nil`. Renvoyez
  `error`/`nil` explicitement.

## 🔁 Pour aller plus loin

- [Ch. 9 — Interfaces (fondamentaux)](09-interfaces.md) : satisfaction implicite, type switch, idiomes.
- [Ch. 11 — Généricité](11-genericite.md) : l'alternative au dispatch quand le type varie peu.
- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : le boxing comme source d'allocations.
- [Ch. 34 — Réflexion](34-reflexion.md) : `reflect` exploite `_type`/`itab` à l'exécution.
- Doc : `go doc reflect.TypeAssert` ; commentaires de `runtime/iface.go` (sources Go).
