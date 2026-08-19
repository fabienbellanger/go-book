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

## 🆕 `encoding/json/v2` (1.27)

Après deux versions derrière `GOEXPERIMENT=jsonv2`, la refonte de l'API JSON est **stable en
Go 1.27**. Elle arrive en deux packages :

- **`encoding/json/v2`** — l'étage **sémantique** : la correspondance entre valeurs Go et JSON
  (`Marshal`, `Unmarshal`, balises de struct, options).
- **`encoding/json/jsontext`** — l'étage **syntaxique** : le JSON vu comme une suite de
  *tokens*, sans aucune connaissance des types Go.

```
   valeur Go  <---- encoding/json/v2 ----> suite de tokens <---- jsontext ----> octets JSON
   (struct)         (sémantique :              (BeginObject,      (syntaxe :        []byte
                     tags, options)             String, …)         échappement,
                                                                   validation)
```

Le changement le plus important est invisible : **`encoding/json` v1 tourne désormais
par-dessus v2**. Même moteur, mais v1 conserve sa sémantique — elle est reconstituée par un jeu
d'options « héritées » (`json.DefaultOptionsV1()`, `json.CallMethodsWithLegacySemantics(…)`…).
Le décodage y gagne nettement en vitesse ; l'encodage est globalement à parité. Aucune
migration n'est requise, et v1 ne sera pas retiré.

> ⚠️ **Ce qui change quand même en v1** : les **messages d'erreur**. Le moteur n'est plus le
> même, donc le texte des erreurs diffère. Un test qui compare `err.Error()` à une chaîne
> littérale va casser — comparez avec `errors.Is`/`errors.As`, ou sur un sous-ensemble stable.
> En cas d'urgence, `GOEXPERIMENT=nojsonv2` restaure l'ancien moteur au build, mais c'est un
> sursis : l'opt-out disparaîtra.

### Ce que v2 change, concrètement

Les **défauts** sont plus stricts, et c'est le cœur de la refonte :

| Situation                        | `encoding/json` (v1)             | `encoding/json/v2`                     |
| -------------------------------- | -------------------------------- | -------------------------------------- |
| Clé JSON dupliquée               | silencieusement, la **dernière** | **erreur**                             |
| UTF-8 invalide dans une chaîne   | remplacé par U+FFFD              | **erreur**                             |
| Appariement clé ↔ champ          | insensible à la casse en dernier recours | **sensible à la casse**        |
| Slice/map `nil` à l'encodage     | `null`                           | `[]` / `{}`                            |
| Ordre des clés d'une **map**     | **trié**                         | **non déterministe** (`Deterministic(true)` pour trier) |
| Clé JSON inconnue au décodage    | ignorée                          | ignorée (option pour refuser)          |

```go
// code/ch42-encoding/jsonv2.go
import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
)

// Ni omitempty ni omitzero : c'est là que la différence se voit.
type Réponse struct {
	Items  []string          `json:"items"`
	Labels map[string]string `json:"labels"`
}

r := Réponse{} // Items et Labels sont nil

b, _ := jsonv2.Marshal(r, jsontext.WithIndent("  "))
// v2 : {"items": [],   "labels": {}}    -- nil s'encode en [] / {}
// v1 : {"items": null, "labels": null}  -- comportement historique préservé

err := jsonv2.Unmarshal([]byte(`{"name":"a","name":"b"}`), &v)
// jsontext: duplicate object member name "name"  (v1 accepte et garde "b")
```

Le premier point supprime un réflexe défensif très répandu : en v1, une API qui renvoie
`{"items": null}` oblige chaque client à distinguer `null` de `[]`, d'où le
`if items == nil { items = []T{} }` juste avant de sérialiser. En v2, il n'a plus lieu d'être.

Le piège « `nil` slice vs `[]` » vu plus haut disparaît donc en v2 — et avec lui le
`if s == nil { s = []T{} }` défensif qu'on écrivait avant de sérialiser une réponse d'API.

### Les options remplacent les comportements implicites

Toutes les fonctions v2 acceptent des `Options` variadiques. On règle explicitement ce qui
était auparavant figé :

```go
json.Marshal(v, jsontext.WithIndent("  "))            // indentation (côté syntaxe)
json.Marshal(v, json.Deterministic(true))             // ordre des clés de map stable
json.Unmarshal(b, &v, json.RejectUnknownMembers(true))  // refuse une clé inconnue
json.Unmarshal(b, &v, json.MatchCaseInsensitiveNames(true)) // retrouve la tolérance v1
```

⚠️ La ligne « ordre des clés de map » est le piège le plus sournois de la migration : v1 trie
les clés, v2 non. Un test à *golden file*, une signature calculée sur du JSON, ou un cache
indexé par la sérialisation d'une map deviennent instables. Passez `json.Deterministic(true)`
partout où l'octet exact compte.

`json.RejectUnknownMembers(true)` mérite une mention : c'est l'équivalent déclaratif de
`Decoder.DisallowUnknownFields()` en v1, et le réflexe à prendre pour valider une charge utile
entrante — une clé mal orthographiée devient une erreur au lieu d'être silencieusement perdue.

### Les quatre portes d'entrée

v2 nomme explicitement ce qui, en v1, se répartissait entre fonctions et types :

| Fonction          | Source / destination            | Équivalent v1                    |
| ----------------- | ------------------------------- | -------------------------------- |
| `Marshal`         | `[]byte`                        | `json.Marshal`                   |
| `MarshalWrite`    | `io.Writer`                     | `json.NewEncoder(w).Encode(v)`   |
| `MarshalEncode`   | `*jsontext.Encoder`             | —                                |
| `Unmarshal`       | `[]byte`                        | `json.Unmarshal`                 |
| `UnmarshalRead`   | `io.Reader`                     | `json.NewDecoder(r).Decode(&v)`  |
| `UnmarshalDecode` | `*jsontext.Decoder`             | —                                |

```go
// Écrire directement dans la réponse HTTP, sans buffer intermédiaire.
func écrireJSON(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.MarshalWrite(w, v)
}
```

Côté types, les interfaces `MarshalerTo` / `UnmarshalerFrom` (méthodes `MarshalJSONTo(*jsontext.Encoder)`
et `UnmarshalJSONFrom(*jsontext.Decoder)`) permettent d'écrire ou lire **directement sur le
flux**, sans allouer le `[]byte` intermédiaire qu'imposaient `MarshalJSON`/`UnmarshalJSON`. Les
anciennes interfaces restent reconnues.

### `jsontext` : le JSON sans les types

Quand on n'a pas besoin de mapper vers des structs — filtrer un flux, réindenter, valider,
transformer une clé — `jsontext` travaille sur les *tokens* :

```go
// code/ch42-encoding/jsonv2.go
dec := jsontext.NewDecoder(r)
for {
	tok, err := dec.ReadToken()
	if err == io.EOF {
		break
	}
	if tok.Kind() == jsontext.KindString {
		// dec.StackPointer() donne le JSON Pointer courant, ex. "/items/3/nom"
	}
}
```

⚠️ Au niveau syntaxique, **une clé d'objet est un token de type chaîne** au même titre qu'une
valeur : sur `{"a":"x"}`, `ReadToken` en livre deux. `dec.StackPointer()` permet de savoir où
l'on se trouve, et `dec.SkipValue()` de sauter un sous-arbre entier sans le construire.

`jsontext.Value` est un texte JSON brut (c'est le nouveau nom de `json.RawMessage`, dont il est
désormais l'alias) avec des méthodes de mise en forme : `Compact`, `Indent`, `Canonicalize`.

> 💡 **Que choisir ?** `encoding/json` pour du code existant et pour l'immense majorité des cas
> — il est maintenant rapide. `encoding/json/v2` pour du **code neuf** où l'on veut les défauts
> stricts et les options explicites. `jsontext` seulement quand la structure typée n'a pas de
> sens (proxy, filtre, outil de mise en forme).

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
- 🆕 **1.27** — le moteur **v2 sous v1** est un gain gratuit : la **désérialisation** accélère
  nettement, l'encodage reste à parité. Vous en profitez sans changer une ligne, simplement en
  recompilant. Mesurez chez vous plutôt que de reprendre un chiffre d'annonce : le gain dépend
  fortement de la forme des documents (profondeur, proportion de chaînes, taille des objets).
- Pour **écrire** un gros document, `MarshalWrite(w, v)` (v2) ou `Encoder` (v1) évitent le
  `[]byte` intermédiaire. Pour n'en lire qu'une partie, `jsontext.Decoder` + `SkipValue()` saute
  les sous-arbres sans jamais les construire.

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
- 🆕 **1.24** `omitzero` règle le cas `time.Time`.
- 🆕 **1.27** — **`encoding/json/v2` est stable** et fait tourner v1 par-dessous : décodage plus
  rapide, comportement v1 inchangé, mais **messages d'erreur différents** (ne les comparez pas
  littéralement). En code neuf, v2 apporte des défauts stricts (doublons et UTF-8 invalide
  refusés, casse respectée) et des **options explicites** ; attention, v2 **ne trie plus** les
  clés de map — `json.Deterministic(true)` si l'octet exact compte.
- 🆕 **1.27** — `encoding/json/jsontext` expose le JSON comme une suite de *tokens* : pour
  filtrer, réindenter ou valider un flux sans le typer.
- `gob` (Go↔Go), `csv` (tableaux), `xml` (interop imposée) complètent la famille ; **JSON par défaut**.
- `regexp` = RE2 (linéaire, pas de backref) : **compilez une fois**, et préférez `strings` quand il suffit.

## 🔁 Pour aller plus loin

- [Ch. 41 — Entrées/sorties & flux](41-io-flux.md) : les `io.Reader`/`Writer` derrière `Encoder`/`Decoder`.
- [Ch. 34 — Réflexion](34-reflexion.md) : ce qui fait fonctionner (et coûter) `encoding/json`.
- [Ch. 32 — Maps](32-maps-hachage.md) : pourquoi l'ordre des clés n'est pas garanti.
- Projet 2 — API REST : JSON aux frontières HTTP, validation, erreurs structurées.
- Projet 6 — Générateur de code : remplacer la réflexion JSON par du code généré.
