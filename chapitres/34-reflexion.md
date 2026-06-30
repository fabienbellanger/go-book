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
**confiner aux frontières** — la réflexion contourne le typage statique et coûte cher.

Le [Ch. 33](33-interfaces-profondeur.md) a montré qu'une interface n'est rien d'autre qu'un **couple**
`(*_type, data)` : le type concret d'un côté, un pointeur vers la valeur de l'autre. `reflect` ne fait
rien de magique : il **lit ce couple** et le range dans deux types Go distincts — `reflect.Type` pour le
mot `*_type`, `reflect.Value` pour le mot `data` (accompagné d'un peu d'état interne : adressable ?
exporté ?). C'est précisément **pourquoi** `Type` et `Value` vont toujours par paire et ne se
substituent jamais l'un à l'autre : l'un répond « de quel type s'agit-il ? », l'autre « quelle est la
valeur, et puis-je la changer ? ». Code dans [`code/ch34-reflexion/`](../code/ch34-reflexion/).

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

Ce triangle se résume en **trois lois**, formulées par Rob Pike dans l'article fondateur « The Laws of
Reflection » (référence en fin de chapitre) :

1. **La réflexion va d'une interface vers un objet de réflexion** — `reflect.TypeOf`/`ValueOf` à partir
   d'une `any`.
2. **La réflexion va d'un objet de réflexion vers une interface** — `Value.Interface()` referme le
   triangle.
3. **Pour modifier un objet de réflexion, sa valeur doit être adressable** — détaillé dans la section
   suivante ; c'est la loi la plus souvent oubliée, et celle qui produit le plus de panics en
   production.

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
vérifie.

```
   reflect.ValueOf(x)         -> Value NON adressable, CanSet()=false
        x (any)                  (x voyage par COPIE dans l'interface, sans lien avec la variable d'origine)

   reflect.ValueOf(&x)        -> Value de Kind Pointer
        |
        .Elem()  -------------> Value ADRESSABLE, CanSet()=true
                                 (déréférence le pointeur : on retombe sur x lui-même, pas une copie)
```

`Elem()` est donc la charnière : sans passer par un pointeur puis le déréférencer, il n'existe **aucun**
chemin vers une `Value` modifiable. Ici, on remplit les champs à zéro depuis les tags `default` :

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
l'écriture **panique**. C'est la **loi 3** ci-dessus : pour modifier, il faut l'**adressabilité**.

> 💡 **Pas de variable Go sous la main ?** `reflect.New(t)` alloue un zéro de `t` sur le tas et renvoie
> une `Value` **pointeur** dessus ; `.Elem()` donne alors une `Value` adressable, exactement comme `&x`
> le ferait pour une variable existante. C'est ainsi qu'un décodeur (`encoding/json`, un ORM) construit
> une instance adressable à partir du seul `reflect.Type` cible, sans variable Go préexistante.

## Les tags de struct : le pont vers le déclaratif

Un **tag** est une chaîne attachée à un champ, lue à l'exécution via `Field.Tag.Get("clé")` /
`.Lookup`. C'est la convention de tout l'écosystème : `json:"name,omitempty"`, `gorm:"primaryKey"`,
`validate:"required"`. La réflexion les lit pour **piloter** sérialisation, validation ou mapping SQL —
sans que le type concerné connaisse l'encodeur.

`Get` et `Lookup` ne sont pas interchangeables : `Get` renvoie simplement une chaîne vide si la clé est
absente — suffisant quand absence et valeur vide se valent (`InspectFields` ci-dessus, où un tag `field`
manquant donne juste un libellé vide). `Lookup` renvoie en plus un `bool`, indispensable dès qu'il faut
**distinguer** « pas de tag » de « tag présent mais vide » — c'est le cas de `FillDefaults` plus loin, où
l'absence du tag `default` signifie « ne touche pas à ce champ ».

## Appeler dynamiquement

`Value.MethodByName` + `Call` invoque une méthode dont le nom n'est connu qu'à l'exécution :

```go
// code/ch34-reflexion/reflectx.go
m := reflect.ValueOf(recv).MethodByName(name)
out := m.Call(in) // in et out sont des []reflect.Value
```

`reflect.ValueOf(recv).MethodByName(name)` renvoie une méthode **liée** (_bound_) : le récepteur est
déjà capturé dans `m`, et `in` ne contient **que** les paramètres déclarés — ne lui passez jamais `recv`
en premier argument, `Call` panique sinon (arité incorrecte).

Avec les itérateurs 1.26, on inspecte aussi la **signature** : `Method.Type.Ins()` / `Outs()` listent
paramètres et retours. Attention à la nuance : la signature ci-dessous vient de
`reflect.TypeOf(Server{}).Method(0)`, une méthode **non liée** obtenue depuis un `Type` — son récepteur
compte alors comme le **1ᵉʳ** `in`. Une méthode liée (`Value.MethodByName`, juste au-dessus) n'a pas ce
décalage : son `Type` exclut déjà le récepteur.

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

**~110× plus lent**, et il alloue **5** fois là où le code direct n'alloue pas. Ce n'est pas un détail
d'implémentation isolé mais la somme de plusieurs surcoûts structurels :

- **Aucun inlining possible** — le [Ch. 33](33-interfaces-profondeur.md) a montré que l'inlining perdu
  est le vrai coût d'un dispatch d'interface ; `reflect` cumule ce problème, car chaque méthode
  (`Field`, `Kind`, `SetString`...) revérifie ses **drapeaux internes** (adressable ? exporté ?
  `CanSet` ?) à chaque appel — des contrôles qu'un compilateur ordinaire prouverait **une fois**,
  statiquement, puis éliminerait.
- **Un `switch` sur `Kind()` remplace un `switch` sur des types Go** — `FillDefaults` teste
  `reflect.String`, `reflect.Int`... à **chaque appel** ; rien ne permet au compilateur de spécialiser
  ce code pour `Server` en particulier, contrairement à `fillDirect`, écrit pour ce type précis.
- **Boxing répété** — `Value.Interface()` et `Value.SetX` repassent par les mêmes mécanismes
  d'allocation que le boxing d'interface (détail dans les Pièges, plus bas).

La leçon : **réflexion aux frontières** (décodage d'une requête, chargement d'une config) — **jamais**
sur le chemin chaud. Pour de la généricité performante, préférez les **génériques**
([Ch. 11](11-genericite.md)) ou la **génération de code** ([Projet 6](../projets/6-codegen/)), qui
résolvent les types **à la compilation**.

### Réflexion, interfaces ou génériques ?

Trois façons de traiter un type qui n'est pas figé d'avance, à des coûts très différents :

| Mécanisme                                                                         | Type connu...                                             | Coût                                                | Cas d'usage typique                                            |
| --------------------------------------------------------------------------------- | --------------------------------------------------------- | --------------------------------------------------- | -------------------------------------------------------------- |
| **Génériques** ([Ch. 11](11-genericite.md))                                       | à la **compilation**, via les contraintes                 | quasi nul, inlining possible                        | conteneurs, algos paramétrés par un type connu au site d'appel |
| **Interfaces** ([Ch. 9](09-interfaces.md), [Ch. 33](33-interfaces-profondeur.md)) | à l'**exécution**, mais figé dans un contrat de méthodes  | dispatch quasi gratuit ; coût réel = inlining perdu | polymorphisme par comportement (`io.Writer`, `sort.Interface`) |
| **`reflect`**                                                                     | **totalement inconnu**, y compris la forme (champs, tags) | ~100× plus lent que le code direct, alloue          | décodeurs génériques, ORM, validation par tags, sérialisation  |

La règle de décision : si le type est connu au site d'appel, génériques ; s'il varie mais respecte un
contrat de méthodes fixe, interface ; seulement si la **forme** elle-même doit être découverte à
l'exécution (quels champs, quels tags, quelles méthodes), `reflect` — et encore, idéalement caché
derrière une fonction appelée une fois, jamais en boucle chaude.

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
- **`switch` sur `Kind()` non exhaustif** — un `case` oublié n'est **pas** une erreur de compilation, il
  est **silencieusement ignoré** à l'exécution : dans
  [`FillDefaults`](../code/ch34-reflexion/reflectx.go), un champ `bool` taggé `default:"true"` ne
  déclenche aucun des deux `case` et reste à sa valeur zéro, sans le moindre avertissement. Un
  `default:` qui retourne une erreur explicite (ou un test couvrant chaque `Kind` attendu) évite cette
  dérive silencieuse.
- **`Call` avec une arité ou des types erronés panique** — contrairement à un appel Go classique, rien
  ne vérifie `in` à la compilation : un nombre d'arguments incorrect ou un type incompatible avec la
  signature déclenche une panique à l'exécution. À valider en amont si l'appelant n'est pas fiable
  (`m.Type().NumIn()`, `m.Type().In(i)`).
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

- `reflect.TypeOf`/`ValueOf` ouvrent une `any` (loi 1) ; **`Kind`** donne la catégorie ;
  `Value.Interface()` referme (loi 2, boxing).
- **Écrire** exige l'**adressabilité** (loi 3 : passer un **pointeur**, `Elem()` — ou `reflect.New`) et
  un champ **exporté** — vérifiez `CanSet`.
- Les **tags de struct** pilotent sérialisation/validation/ORM ; `Get` ignore l'absence, `Lookup` la
  signale via un `bool`.
- **1.26** : itérateurs `Type.Fields/Methods`, `Value.Fields/Methods`, `Type.Ins/Outs`. **1.25** :
  `reflect.TypeAssert[T]` sans allocation.
- La réflexion est **~100× plus lente** que le code direct et alloue, faute d'inlining et à cause des
  vérifications dynamiques répétées : **confinez-la aux frontières** ; type connu → génériques, contrat
  de méthodes fixe → interface, forme inconnue à l'exécution → `reflect`.

## 🔁 Pour aller plus loin

- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : `_type`/`itab`, ce que `reflect` lit.
- [Ch. 11 — Généricité](11-genericite.md) : l'alternative typée à la compilation.
- [Ch. 13 — Tests & outillage](13-tests-outillage.md) : tags de struct et `Example` testables.
- [Projet 6 — Générateur de code](../projets/6-codegen/) : produire du code direct via `go generate`.
- Doc : `go doc reflect` ; « The Laws of Reflection » (go.dev/blog/laws-of-reflection).
