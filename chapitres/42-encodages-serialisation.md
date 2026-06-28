# Ch. 42 — Encodages & sérialisation : `encoding/json`, `gob`/`csv`/`xml`, `regexp`

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
- **`nil slice` vs `[]`** — une slice `nil` s'encode `null`, une slice **vide** (`[]T{}`) s'encode
  `[]`. Côté client, la différence compte ; choisissez et soyez constant.
- **Ordre des clés** d'un objet décodé en `map` : non déterministe (🔁 [Ch. 32](32-maps-hachage.md)).
  Pour un ordre stable, décodez vers une **struct**.
- **`Unmarshal` sans pointeur** → erreur `json: Unmarshal(non-pointer …)`.

---

## 🆕 `encoding/json/v2` (expérimental)

Go 1.25/1.26 embarquent une refonte majeure, **`encoding/json/v2`** (et son étage syntaxique
`encoding/json/jsontext`). Elle n'existe **que** si l'on compile avec
`GOEXPERIMENT=jsonv2` — c'est **expérimental**, hors promesse de compatibilité Go 1, et **non
utilisé** dans le code de ce chapitre.

Apports visés :

- **Performance** — décodage nettement plus rapide et moins d'allocations (streaming repensé).
- **Options explicites** — les fonctions acceptent des `Options` variadiques
  (`json.Marshal(v, opts...)`) au lieu de comportements implicites ; on configure finement la
  sémantique (clés inconnues, casse, formats) et la syntaxe.
- **Défauts plus sains** — ex. les map sont triées par clé à l'encodage, gestion plus stricte.
- **API de flux dédiée** — `MarshalWrite`/`UnmarshalRead` (vers `io.Writer`/`io.Reader`),
  `MarshalEncode`/`UnmarshalDecode` (vers un `jsontext.Encoder`/`Decoder`).

> 💡 En production, restez sur `encoding/json` (v1, stable) jusqu'à stabilisation de v2. Surveillez
> les notes de version : l'objectif à terme est que v1 délègue à v2.

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
backtracking (⚡). 🔁 voir aussi `strings`/`bytes` ([Ch. 7](07-maps-strings.md)).

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

- **JSON est cher** : réflexion à chaque champ (🔁 [Ch. 34](34-reflexion.md)). Pour un chemin
  ultra-chaud, un `MarshalJSON` manuel ou de la génération de code (🔁 Projet 6) bat la réflexion.
- **Streaming** (`Encoder`/`Decoder`) évite de matérialiser un gros `[]byte` intermédiaire.
- **`regexp`** : compiler une fois ; les variantes `…Bytes` évitent une conversion `string`↔`[]byte`
  inutile quand on travaille déjà sur des octets.
- **`json/v2`** (expérimental) vise précisément à réduire allocations et temps de décodage.

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
- [Ch. 32 — Maps : tables de hachage](32-maps-hachage.md) : pourquoi l'ordre des clés n'est pas garanti.
- Projet 2 — API REST : JSON aux frontières HTTP, validation, erreurs structurées.
- Projet 6 — Générateur de code : remplacer la réflexion JSON par du code généré.
