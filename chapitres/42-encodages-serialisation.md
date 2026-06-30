# 42 — Encodages & sérialisation

> **Objectif** — Sérialiser et désérialiser des données dans les formats du quotidien
> (JSON en tête), choisir le bon encodage selon le contexte, et manipuler du texte structuré
> avec `regexp` sans se faire piéger.
>
> **Prérequis** — [Ch. 7 — Maps & strings](07-maps-strings.md), [Ch. 8 — Structs](08-structs-methodes.md),
> [Ch. 9 — Interfaces](09-interfaces.md)

---

## Introduction

Dès qu'un programme parle au monde extérieur — une API, un fichier de config, un message réseau —
il doit **transformer ses valeurs Go en octets** et inversement. C'est la **sérialisation**. Go
fournit dans `encoding/…` une famille de packages cohérents, tous bâtis sur le même contrat :
des fonctions `Marshal`/`Unmarshal` et des types `Encoder`/`Decoder` pour le **streaming**.

Le format roi est **JSON** ; on lui consacre l'essentiel du chapitre, puis on survole `gob`,
`csv` et `xml`, et on termine par `regexp` pour l'extraction de texte. Le code complet est dans
[`code/ch42-encoding/`](../code/ch42-encoding/).

---

## `encoding/json` : le cœur

### Marshal / Unmarshal

```go
b, err := json.Marshal(v)        // valeur Go  -> []byte JSON
err = json.Unmarshal(b, &v)      // []byte JSON -> valeur Go (pointeur !)
```

`Unmarshal` exige un **pointeur** vers la destination. La correspondance se fait par **nom de
champ exporté** ; on l'ajuste avec des **tags de struct**.

```go
type Event struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Tags      []string  `json:"tags,omitempty"`      // omis si nil/vide
	CreatedAt time.Time `json:"created_at,omitzero"` // omis si zéro (🆕 1.24)
	Score     int64     `json:"score,string"`        // encodé "42" (chaîne)
	internal  string    // non exporté : JAMAIS sérialisé
	Secret    string    `json:"-"`                   // exclu explicitement
}
```

| Tag                     | Effet                                                             |
| ----------------------- | ----------------------------------------------------------------- |
| `json:"name"`           | renomme la clé JSON                                               |
| `json:"name,omitempty"` | omet si la valeur est **vide** (`0`, `""`, `nil`, slice/map vide) |
| `json:"name,omitzero"`  | 🆕 1.24 — omet si la valeur est **zéro** (gère `time.Time`, etc.) |
| `json:",string"`        | encode un nombre/booléen comme **chaîne** JSON (`"42"`)           |
| `json:"-"`              | champ **jamais** (dé)sérialisé                                    |

> ⚠️ **Seuls les champs exportés** (initiale majuscule) sont vus par `encoding/json`. Un champ
> non exporté (`internal`) est ignoré **silencieusement** — pas d'erreur, juste un trou.

**Pourquoi la réflexion, et pourquoi des tags ?** `Marshal`/`Unmarshal` acceptent un paramètre
`any` : la fonction ne connaît le type concret de `v` qu'à l'**exécution**, pas à la compilation.
Le seul moyen d'inspecter un type Go arbitraire — lister ses champs, lire leurs tags, lire ou
écrire leurs valeurs — sans écrire un cas particulier par type, est le package `reflect` (🔁
[Ch. 34](34-reflexion.md)). C'est la même contrainte qui justifie les tags : un identifiant Go
exporté doit commencer par une majuscule, alors que les API JSON suivent souvent une autre
convention (`snake_case`, `camelCase`). Le tag `json:"..."` découple le nom Go (imposé par le
compilateur) du nom de la clé JSON (convention du format) ; `encoding/json` le lit via
`reflect.StructTag.Get` et **met en cache** ces métadonnées par type pour ne pas reparcourir les
tags à chaque appel — la même stratégie de cache que recommande le Ch. 34 pour tout usage
réfléchi répété.

### `omitempty` vs `omitzero` (🆕 1.24)

Le piège historique : `omitempty` considère « vide » selon la valeur zéro de **bas niveau**.
Pour `time.Time`, le zéro (`0001-01-01…`) **n'est pas** une valeur vide au sens de `omitempty`,
donc le champ apparaissait toujours. `omitzero` (Go 1.24) corrige cela : il omet le champ s'il
**égale sa valeur zéro** (et appelle `IsZero()` si le type l'implémente).

```
  omitempty  : omet si len==0 / ==0 / ==nil / ==false   (notion "vide")
  omitzero   : omet si == valeur zéro du type            (notion "zéro", gère time.Time)
  les deux   : omet si vide OU zéro
```

### Encodage personnalisé : `Marshaler` / `Unmarshaler`

Un type peut contrôler totalement sa forme JSON en implémentant ces interfaces (🔁
[Ch. 9](09-interfaces.md)) :

```go
type Temperature float64

func (t Temperature) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", fmt.Sprintf("%.1f°C", float64(t)))), nil
}
func (t *Temperature) UnmarshalJSON(data []byte) error { /* parse "21.5°C" */ }
```

`Temperature(21.5)` s'encode alors en `"21.5°C"` au lieu de `21.5`.

> 💡 `time.Time` implémente lui-même `Marshaler`/`Unmarshaler` : c'est ce qui explique pourquoi
> le champ `CreatedAt` de `Event`, plus haut, s'encode en chaîne RFC 3339
> (`"2023-11-14T22:13:20Z"`) plutôt qu'en nombre. Sans cette implémentation, la représentation
> interne de `time.Time` (secondes, nanosecondes, fuseau) fuiterait telle quelle dans le JSON.

### Streaming : `Encoder` / `Decoder`

Pour lire/écrire un **flux** (fichier, corps HTTP, connexion) sans tout charger en mémoire, on
utilise `Encoder`/`Decoder` branchés sur un `io.Writer`/`io.Reader` (🔁
[Ch. 41 — I/O](41-io-flux.md)) :

```go
dec := json.NewDecoder(r)
dec.DisallowUnknownFields()   // un champ JSON inconnu => erreur (validation stricte)
var e Event
err := dec.Decode(&e)         // décode UN document ; rappeler en boucle pour un flux
```

`DisallowUnknownFields` est précieux côté API : un champ en trop devient une **erreur** au lieu
d'être ignoré en silence. `Decoder.Token()` permet en plus un parcours **token par token** (clés,
valeurs, `[`, `{`…) pour les très gros documents.

### Types utiles

- **`json.RawMessage`** — `[]byte` JSON **brut**, différé : on garde le fragment non décodé pour
  le traiter plus tard (utile quand le type dépend d'un champ « discriminant »).
- **`json.Number`** — préserve le nombre **tel quel** (chaîne), sans le convertir en `float64` ;
  active-le via `dec.UseNumber()` pour éviter la perte de précision sur les grands entiers.

Concrètement, le motif « champ discriminant » ressemble à ceci :

```go
type Envelope struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"` // décodage différé : dépend de Kind
}

var env Envelope
json.Unmarshal(b, &env) // Data reste un fragment JSON brut à ce stade
switch env.Kind {
case "temperature":
	var t Temperature
	json.Unmarshal(env.Data, &t) // décodage déclenché une fois Kind connu
}
```

### Décoder vers `any`

Sans type cible précis, JSON se décode vers des types **génériques** :

```
  null     -> nil
  true     -> bool
  nombre   -> float64        (⚠️ même pour un entier ! perte de précision > 2^53)
  chaîne   -> string
  tableau  -> []any
  objet    -> map[string]any (⚠️ ordre des clés NON garanti)
```

---

## ⚠️ Pièges JSON

- **Champ non exporté** non sérialisé — silence total. Exportez-le (et taguez-le).
- **`any` décode un nombre en `float64`** — un `int64` > 2^53 perd des chiffres. Décodez vers un
  champ `int64` typé, ou activez `UseNumber()`.
- **`nil` slice vs `[]`** (🔁 [Ch. 6](06-arrays-slices.md)) — une slice `nil` s'encode `null`, une
  slice **vide** (`[]T{}`) s'encode `[]`. Côté client, la différence compte (un consommateur peut
  faire `arr.length` sur `[]` mais planter sur `null`) ; choisissez et soyez constant. C'est un
  défaut de `encoding/json` v1 — `json/v2` (plus bas) change cette règle.
- **Correspondance de clé insensible à la casse en repli** — si aucun champ Go (nom exporté ou
  tag) ne correspond **exactement** à une clé JSON, `Unmarshal` retente une correspondance
  **insensible à la casse** avant d'abandonner. Un champ `Name` accepte donc `"name"`, `"NAME"`,
  voire `"naMe"` — pratique en cas d'à-peu-près, mais ça masque une faute de frappe côté client
  qui devrait plutôt produire un champ inconnu.
- **`Unmarshal` ne réinitialise pas la cible** — décoder un JSON partiel dans une struct déjà
  remplie **ne remet pas à zéro** les champs absents du document ; seuls les champs présents sont
  écrasés. Pratique pour un PATCH partiel, piégeux si l'on s'attend à un remplacement complet
  (repartez d'une valeur zéro explicite, ex. `e = Event{}`, avant de décoder).
- **Ordre des clés** d'un objet décodé en `map` : non déterministe (🔁 [Ch. 32](32-maps-hachage.md)).
  Pour un ordre stable, décodez vers une **struct**.
- **`Unmarshal` sans pointeur** → erreur `json: Unmarshal(non-pointer …)`.

---

## 🆕 `encoding/json/v2` (expérimental)

Go 1.25/1.26 embarquent une refonte majeure, **`encoding/json/v2`** (et son étage syntaxique
`encoding/json/jsontext`). Elle n'existe **que** si l'on compile avec
`GOEXPERIMENT=jsonv2` — c'est **expérimental**, hors promesse de compatibilité Go 1, et **non
utilisé** dans le code de ce chapitre. Activer le flag ne fait pas que révéler ces nouveaux
packages : en Go 1.26, il fait aussi tourner `encoding/json` v1 **par-dessus** v2 en interne
(même implémentation), avec un comportement annoncé comme identique dans l'immense majorité des
cas. C'est précisément ce qui permet à l'équipe Go de mesurer l'impact de v2 sur du code v1
existant, sans le réécrire.

Apports visés :

- **Performance** — décodage nettement plus rapide et moins d'allocations (streaming repensé,
  nouvelles interfaces `MarshalerTo`/`UnmarshalerFrom` qui écrivent/lisent directement sur un
  `jsontext.Encoder`/`Decoder` sans passer par un `[]byte` intermédiaire).
- **Options explicites** — les fonctions acceptent des `Options` variadiques
  (`json.Marshal(v, opts...)`) au lieu de comportements implicites ; on configure finement la
  sémantique (clés inconnues, casse, formats) et la syntaxe.
- **Défauts plus stricts** — clés JSON dupliquées et octets UTF-8 invalides deviennent des
  **erreurs** (v1 les tolère silencieusement) ; l'appariement nom de champ ↔ clé JSON en
  désérialisation est **sensible à la casse** par défaut (v1 retombe sur une correspondance
  insensible à la casse, 🔁 Pièges ci-dessus) ; une slice/map **`nil`** s'encode en `[]`/`{}` au
  lieu de `null` — le piège « `nil` slice vs `[]` » plus haut disparaît en v2.
- **API de flux dédiée** — `MarshalWrite`/`UnmarshalRead` (vers `io.Writer`/`io.Reader`),
  `MarshalEncode`/`UnmarshalDecode` (vers un `jsontext.Encoder`/`Decoder`).

> 💡 En production, restez sur `encoding/json` sans le flag (comportement par défaut inchangé,
> stable) jusqu'à stabilisation de v2 — v1 ne sera de toute façon **jamais retiré**, la migration
> est prévue comme une option, pas une obligation. Surveillez les notes de version pour suivre la
> décision d'adoption (ou d'abandon) du flag.

---

## Autres encodages (survol)

### `encoding/gob` — Go ↔ Go

Format **binaire auto-décrit**, idéal pour échanger des valeurs **entre deux programmes Go**
(cache, RPC maison, snapshot). Pas d'interopérabilité avec d'autres langages.

```go
var buf bytes.Buffer
gob.NewEncoder(&buf).Encode(in)   // sérialise
gob.NewDecoder(&buf).Decode(&out) // désérialise
```

### `encoding/csv` — tableaux

`Reader`/`Writer` pour le CSV. `FieldsPerRecord` (fixé par la 1re ligne) garantit un nombre de
colonnes constant — une ligne irrégulière devient une **erreur**.

```go
r := csv.NewReader(src)
rows, err := r.ReadAll() // [][]string
```

### `encoding/xml` — quand on n'a pas le choix

API jumelle de JSON (`Marshal`/`Unmarshal`, tags `xml:"..."`). Plus lourde et plus piégeuse
(espaces de noms, attributs vs éléments).

> 💡 Préférez **JSON** par défaut ; n'utilisez XML que pour **interopérer** avec un système qui
> l'impose (SOAP, configs héritées).

| Format | Interop            | Lisible | Taille   | Cas d'usage                  |
| ------ | ------------------ | ------- | -------- | ---------------------------- |
| JSON   | universelle        | oui     | moyenne  | API web, configs, logs       |
| gob    | Go uniquement      | non     | compacte | échange Go↔Go, cache binaire |
| csv    | tableurs           | oui     | petite   | données tabulaires, exports  |
| xml    | héritée/entreprise | oui     | grande   | SOAP, formats imposés        |

---

## `regexp` : expressions régulières

Le moteur de Go est **RE2** : **pas de références arrière** (`\1`) ni de lookaround, mais une
**garantie de temps linéaire** — pas d'« explosion catastrophique » comme avec les moteurs à
backtracking (⚡). Le mécanisme : RE2 compile le motif en un **automate fini**, parcouru en une
seule passe sur l'entrée (temps proportionnel à sa longueur). Un moteur à backtracking (PCRE, les
regex de Python ou Perl) explore au contraire **toutes** les combinaisons possibles par
récursion, ce qui peut dégénérer en temps **exponentiel** sur certains motifs pathologiques (ex.
`(a+)+b` appliqué à une longue chaîne sans `b`). Références arrière et lookaround exigent
justement cette capacité de backtracking — RE2 les exclut délibérément pour préserver sa
garantie. 🔁 voir aussi `strings`/`bytes` ([Ch. 7](07-maps-strings.md)).

### Compiler une fois

```go
// Compilée UNE SEULE FOIS, au niveau package.
var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)
```

`MustCompile` panique si le motif est invalide (idéal pour une constante de programme) ;
`Compile` renvoie une **erreur** (pour un motif venu de l'extérieur).

> ⚠️ **Ne jamais compiler une regexp dans une boucle chaude.** La compilation est coûteuse ;
> compilez-la une fois et réutilisez l'objet `*Regexp` (qui est **sûr** en concurrence).

### Méthodes courantes

```go
re.MatchString(s)              // bool : correspond ?
re.FindStringSubmatch(s)       // []string : [tout, groupe1, groupe2, ...]
re.FindAllString(s, -1)        // toutes les correspondances
re.ReplaceAllString(s, "-")    // remplace
re.ReplaceAllStringFunc(s, f)  // remplace via une fonction
```

### Groupes nommés

`(?P<name>…)` nomme un groupe ; `SubexpNames()` donne la correspondance indice → nom (l'indice 0
est la correspondance entière, donc `names[1]` est le 1er groupe).

```go
var kvPattern = regexp.MustCompile(`(?P<key>\w+)=(?P<value>\w+)`)
names := kvPattern.SubexpNames()              // ["", "key", "value"]
for _, m := range kvPattern.FindAllStringSubmatch(s, -1) {
	// m[0] = "env=prod", m[1] = "env", m[2] = "prod"
}
```

> 💡 **`regexp` n'est pas toujours la bonne réponse.** Pour un préfixe, un suffixe, une sous-chaîne
> ou un découpage simple, `strings.HasPrefix`/`Contains`/`Split`/`Cut` sont plus rapides et plus
> lisibles. Sortez `regexp` quand le motif est réellement variable.

---

## ⚡ Performance

- **JSON est cher** : `Marshal`/`Unmarshal` paient l'introspection par réflexion à **chaque
  champ**, à **chaque appel** — le Ch. 34 chiffre ce surcoût à environ **110× plus lent** qu'un
  code écrit à la main, avec des allocations là où le code direct n'en fait aucune (🔁
  [Ch. 34](34-reflexion.md)). `encoding/json` **atténue** ce coût en mettant en cache les
  métadonnées de champs par type, calculées une fois puis réutilisées — mais ne l'élimine pas : le
  parcours des valeurs reste réflexif à chaque appel. Pour un chemin ultra-chaud, un `MarshalJSON`
  manuel ou de la génération de code (🔁 Projet 6) bat la réflexion.
- **Streaming** (`Encoder`/`Decoder`) évite de matérialiser un gros `[]byte` intermédiaire.
- **`regexp`** : compiler une fois ; les variantes `…Bytes` évitent une conversion `string`↔`[]byte`
  inutile quand on travaille déjà sur des octets.
- **`json/v2`** (expérimental) vise précisément à réduire allocations et temps de décodage ; les
  annonces de l'équipe Go évoquent jusqu'à **10×** plus rapide en désérialisation sur certains
  projets ayant testé le flag — à vérifier sur son propre code avant toute généralisation.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch42-encoding
go test -race ./ch42-encoding/...
```

À essayer :

1. Retirez `omitzero` du champ `CreatedAt` et observez la date zéro `0001-01-01T00:00:00Z`
   réapparaître dans la sortie JSON.
2. Ajoutez un champ inconnu à l'entrée de `strictDecode` et vérifiez l'erreur renvoyée par
   `DisallowUnknownFields`.
3. Remplacez `kvPattern` par un découpage `strings.Cut` + `strings.Fields` et comparez la
   lisibilité.

---

## 📌 À retenir

- `encoding/json` : `Marshal`/`Unmarshal` (pointeur requis), tags `omitempty`/`omitzero`/`,string`/`-` ;
  seuls les **champs exportés** comptent.
- Pour un flux, utilisez `Encoder`/`Decoder` ; `DisallowUnknownFields` valide strictement les entrées.
- 🆕 **1.24** `omitzero` règle le cas `time.Time` ; 🆕 **json/v2** (expérimental, `GOEXPERIMENT=jsonv2`)
  préfigure une API plus rapide et explicite — pas encore pour la production.
- `gob` (Go↔Go), `csv` (tableaux), `xml` (interop imposée) complètent la famille ; **JSON par défaut**.
- `regexp` = RE2 (linéaire, pas de backref) : **compilez une fois**, et préférez `strings` quand il suffit.

## 🔁 Pour aller plus loin

- [Ch. 41 — Entrées/sorties & flux](41-io-flux.md) : les `io.Reader`/`Writer` derrière `Encoder`/`Decoder`.
- [Ch. 34 — Réflexion](34-reflexion.md) : ce qui fait fonctionner (et coûter) `encoding/json`.
- [Ch. 32 — Maps](32-maps-hachage.md) : pourquoi l'ordre des clés n'est pas garanti.
- Projet 2 — API REST : JSON aux frontières HTTP, validation, erreurs structurées.
- Projet 6 — Générateur de code : remplacer la réflexion JSON par du code généré.
