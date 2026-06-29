# 34 — Réflexion (`reflect`)

> **Objectif** — Introspecter et manipuler des valeurs **dynamiquement** : `Type`, `Value`, `Kind` ;
> **lire** et **écrire** (`CanSet`, adressabilité) ; exploiter les **tags de struct** (décodeurs, ORM) ;
> **appeler** une méthode par son nom ; connaître le **coût** de la réflexion et ses **alternatives**
> (génériques, code-gen). Avec les **itérateurs `reflect` de Go 1.26**.
>
> **Prérequis** — [Ch. 9](09-interfaces.md), [Ch. 33](33-interfaces-profondeur.md)

---

## Introduction

La réflexion, c'est la capacité d'un programme à **inspecter et modifier ses propres valeurs** à
l'exécution, sans connaître leur type à la compilation. C'est le moteur de `encoding/json`, des ORM, des
frameworks de validation : tout ce qui doit traiter **n'importe quelle** struct. Puissant, mais à
**confiner aux frontières** — la réflexion contourne le typage statique et coûte cher. Le
[Ch. 33](33-interfaces-profondeur.md) a montré `_type`/`itab` ; `reflect` les **expose**. Code dans
[`code/ch34-reflexion/`](../code/ch34-reflexion/).

---

## Le triangle `Type` / `Value` / `interface`

Tout part d'une `any`. Deux fonctions l'ouvrent ; une méthode referme :

```
         reflect.TypeOf(x)       +-----------+
   x  -------------------------> |   Type    |   Kind, Name, champs, méthodes
  (any)                          +-----------+
         reflect.ValueOf(x)      +-----------+
   x  -------------------------> |   Value   |   lecture ; écriture SI adressable
                                 +-----------+
         v.Interface()           renvoie une any
   Value ----------------------> x
```

- **`reflect.Type`** décrit le type : son **`Kind`** (catégorie : `Struct`, `Int`, `Slice`, `Pointer`…),
  son nom, ses champs, ses méthodes.
- **`reflect.Value`** enveloppe la **valeur** : on la lit, et on l'écrit **si elle est adressable**.
- **`Value.Interface()`** revient à une `any` (au prix d'un boxing, [Ch. 33](33-interfaces-profondeur.md)).

## Introspecter une struct : `Type.Fields()` (1.26)

Go 1.26 ajoute des **itérateurs** ([Ch. 18](18-iterateurs.md)) à `reflect`. `Type.Fields()` parcourt les
champs sans index manuel :

```go
// code/ch34-reflexion/reflectx.go
func InspectFields(v any) []FieldInfo {
	t := reflect.TypeOf(v)
	var out []FieldInfo
	for f := range t.Fields() { // itérateur 1.26 : iter.Seq[StructField]
		if !f.IsExported() {
			continue // les champs non exportés sont visibles mais non modifiables
		}
		out = append(out, FieldInfo{f.Name, f.Type.String(), f.Tag.Get("field")})
	}
	return out
}
```

```
$ go run ./ch34-reflexion
champs de Server :
  Host  string  tag="host"
  Port  int     tag="port"
```

## Écrire : adressabilité et `CanSet`

Lire est facile ; **écrire** impose deux conditions : la `Value` doit être **adressable** (donc obtenue
via un **pointeur** : `reflect.ValueOf(&x).Elem()`) et le champ doit être **exporté**. `CanSet` le
vérifie. Ici, on remplit les champs à zéro depuis les tags `default` :

```go
// code/ch34-reflexion/reflectx.go
func FillDefaults(ptr any) error {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("FillDefaults attend un *struct non nil, reçu %T", ptr)
	}
	for sf, fv := range v.Elem().Fields() { // Value.Fields() 1.26 : (StructField, Value)
		def, ok := sf.Tag.Lookup("default")
		if !ok || !fv.CanSet() || !fv.IsZero() {
			continue
		}
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(def)
		case reflect.Int, reflect.Int64:
			n, _ := strconv.ParseInt(def, 10, 64)
			fv.SetInt(n)
		}
	}
	return nil
}
```

Passer une **valeur** (non pointeur) donnerait une `Value` **non adressable** : `CanSet` vaut `false`,
l'écriture **panique**. C'est la « troisième loi de la réflexion » : pour modifier, il faut
l'**adressabilité**.

## Les tags de struct : le pont vers le déclaratif

Un **tag** est une chaîne attachée à un champ, lue à l'exécution via `Field.Tag.Get("clé")` /
`.Lookup`. C'est la convention de tout l'écosystème : `json:"name,omitempty"`, `gorm:"primaryKey"`,
`validate:"required"`. La réflexion les lit pour **piloter** sérialisation, validation ou mapping SQL —
sans que le type concerné connaisse l'encodeur.

## Appeler dynamiquement

`Value.MethodByName` + `Call` invoque une méthode dont le nom n'est connu qu'à l'exécution :

```go
// code/ch34-reflexion/reflectx.go
m := reflect.ValueOf(recv).MethodByName(name)
out := m.Call(in) // in et out sont des []reflect.Value
```

Avec les itérateurs 1.26, on inspecte aussi la **signature** : `Method.Type.Ins()` / `Outs()` listent
paramètres et retours (le récepteur est le 1ᵉʳ `in`).

```
$ go run ./ch34-reflexion
signature Addr : in=[main.Server ] out=[string ]
```

## Le coût : confiner la réflexion

La réflexion **paie l'introspection à chaque appel**. Comparé au code écrit à la main :

| Variante                 | ns/op     | B/op  | allocs/op |
| ------------------------ | --------- | ----- | --------- |
| `FillDefaults` (reflect) | **355,5** | 160   | **5**     |
| `fillDirect` (à la main) | **3,2**   | **0** | **0**     |

**~110× plus lent**, et il alloue. La leçon : **réflexion aux frontières** (décodage d'une requête,
chargement d'une config) — **jamais** sur le chemin chaud. Pour de la généricité performante, préférez
les **génériques** ([Ch. 11](11-genericite.md)) ou la **génération de code**
([Projet 6](../projets/6-codegen/)), qui résolvent les types **à la compilation**.

---

## 🆕 Go 1.2x

- **1.25** — **`reflect.TypeAssert[T](v)`** : extraire un type concret d'une `Value` **sans** l'allocation
  de `v.Interface().(T)` ([Ch. 33](33-interfaces-profondeur.md)).
- **1.26** — **itérateurs** : `Type.Fields()`, `Type.Methods()`, `Value.Fields()`, `Value.Methods()`,
  et `Type.Ins()`/`Outs()` pour les signatures. Fini les boucles `for i := 0; i < t.NumField(); i++` —
  vérifié sur 1.26.4. 🔁 [Ch. 18](18-iterateurs.md).

## ⚠️ Pièges

- **Écrire une `Value` non adressable** — `SetX` **panique** si la `Value` ne vient pas d'un pointeur
  (`reflect.ValueOf(&x).Elem()`). Testez `CanSet`.
- **Champs non exportés** — visibles en lecture, **non modifiables** ; `CanSet` est `false`.
- **`Interface()` sur le chemin chaud** — re-boxe (alloue). Utilisez `TypeAssert` (1.25) ou les accès
  typés (`Value.Int()`, `Value.String()`).
- **Réflexion partout** — c'est un **trou** dans le typage statique : les erreurs surgissent à
  l'exécution. Confinez-la, testez-la, documentez-la.
- **`DeepEqual` en production** — pratique en test, mais **lent** et parfois surprenant (ne l'utilisez pas
  pour comparer des valeurs sur un chemin critique).

## ⚡ Performance

- **Mettez en cache** ce qui est stable : `reflect.Type`, la liste des champs, les tags **précompilés**
  par type (c'est ce que fait `encoding/json` en interne).
- Privilégiez les **génériques** ([Ch. 11](11-genericite.md)) quand le type varie peu : zéro réflexion,
  inlining possible.
- La **génération de code** (`go generate`) produit du code **direct** à partir d'un schéma : vitesse du
  code à la main, ergonomie du déclaratif.
- 🔁 [Ch. 33](33-interfaces-profondeur.md) (boxing) et [Ch. 36](36-tests-benchmarks-fuzzing.md) (mesurer).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch34-reflexion
go test ./ch34-reflexion/...
go test -bench=. -benchmem -run=^$ ./ch34-reflexion/...
```

À essayer :

1. Ajoutez un champ `bool` avec un tag `default:"true"` et gérez `reflect.Bool` dans `FillDefaults`.
2. Écrivez un mini-encodeur `toQuery(v any) string` (`host=...&port=...`) avec `Type.Fields()` + tags.
3. Mesurez l'écart `FillReflect`/`FillDirect` : la réflexion mérite-t-elle d'être mise en cache ?

---

## 📌 À retenir

- `reflect.TypeOf`/`ValueOf` ouvrent une `any` ; **`Kind`** donne la catégorie ; `Value.Interface()`
  referme (boxing).
- **Écrire** exige l'**adressabilité** (passer un **pointeur**, `Elem()`) et un champ **exporté** —
  vérifiez `CanSet`.
- Les **tags de struct** pilotent sérialisation/validation/ORM ; la réflexion les lit à l'exécution.
- **1.26** : itérateurs `Type.Fields/Methods`, `Value.Fields/Methods`, `Type.Ins/Outs`. **1.25** :
  `reflect.TypeAssert[T]` sans allocation.
- La réflexion est **~100× plus lente** que le code direct et alloue : **confinez-la aux frontières** ;
  ailleurs, génériques ou code-gen.

## 🔁 Pour aller plus loin

- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : `_type`/`itab`, ce que `reflect` lit.
- [Ch. 11 — Généricité](11-genericite.md) : l'alternative typée à la compilation.
- [Ch. 13 — Tests & outillage](13-tests-outillage.md) : tags de struct et `Example` testables.
- [Projet 6 — Générateur de code](../projets/6-codegen/) : produire du code direct via `go generate`.
- Doc : `go doc reflect` ; « The Laws of Reflection » (go.dev/blog/laws-of-reflection).
