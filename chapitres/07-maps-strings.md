# 7 — Maps & strings (usage)

> **Objectif** — Manipuler les **tables associatives** (`map`) et le **texte** (`string`),
> en intégrant le modèle « une string est une suite d'**octets** UTF-8, pas de caractères ».
>
> **Prérequis** — [Ch. 6 — Arrays & slices](06-arrays-slices.md)

---

## Introduction

Deux types omniprésents du quotidien Go : la **map** (dictionnaire clé → valeur) et la
**string** (texte). Tous deux cachent une subtilité que ce chapitre démystifie : l'itération
d'une map est **volontairement désordonnée**, et une string est une séquence d'**octets**
encodés en **UTF-8** — pas une suite de caractères indexables. Les internals (tables de
hachage, immutabilité) viendront aux [Ch. 32](32-maps-hachage.md) et
[Ch. 31](31-strings-profondeur.md).

L'exemple complet est dans [`code/ch07-maps-strings/`](../code/ch07-maps-strings/).

---

## Partie 1 — Les maps

Une **map** associe des **clés** uniques à des **valeurs** : `map[K]V`. Accès, insertion et
suppression sont en **O(1) amorti**.

```
   map[string]int   (vue logique — internals au Ch. 32)
   ---------------------------------------------------
   "alice" --hash--> [ "alice" : 30 ]
   "bob"   --hash--> [ "bob"   : 25 ]

   - clé K : doit être d'un type COMPARABLE (== défini) ; pas de slice, map ni func.
   - ordre d'itération : RANDOMISÉ à dessein -> ne jamais en dépendre.
```

### Créer une map

```go
var m map[string]int        // nil : lecture OK, mais ÉCRITURE -> panique !
m = map[string]int{}        // map vide, prête à l'emploi
m = map[string]int{"a": 1}  // littéral
m = make(map[string]int)    // équivalent à {}
m = make(map[string]int, 8) // pré-dimensionne pour ~8 entrées (évite des réallocations)
```

> ⚠️ **`nil` map** : `var m map[string]int` vaut `nil`. On peut la **lire** (`m["x"]` renvoie
> la zero value) et l'itérer (0 fois), mais **écrire dedans panique**. Initialisez-la avec
> `{}` ou `make` avant toute écriture. (Différent du `nil` slice, où `append` fonctionne.)

### Lire, écrire, et le test `comma-ok`

Lire une clé **absente** ne provoque jamais d'erreur : on récupère la **zero value** du type
de valeur. Pour distinguer « absente » de « présente avec la valeur zéro », utilisez la forme
à **deux résultats** (`comma-ok`) :

```go
v := m["alice"]        // 30 si présent, 0 (zero value) sinon — ambigu !
v, ok := m["alice"]    // ok == true si la clé existe
if v, ok := m["zoe"]; !ok {
	// "zoe" n'est pas dans la map
}
```

Conséquence pratique très idiomatique — incrémenter un compteur sans initialiser :

```go
counts := map[string]int{}
counts[word]++   // clé absente -> part de 0, puis 1, 2, ...
```

### `delete`, `clear`, `len`

```go
delete(m, "bob")   // supprime la clé "bob" (no-op si absente, jamais d'erreur)
clear(m)           // 🆕 1.21 : vide TOUTES les entrées (la map reste non-nil)
n := len(m)        // nombre d'entrées
```

### Itération : **non ordonnée**

`range` sur une map visite les paires dans un ordre **aléatoire**, qui **change à chaque
exécution** (Go randomise volontairement pour empêcher tout code de dépendre d'un ordre) :

```go
for k, v := range m {
	// ordre NON garanti, différent à chaque run
}
```

**Parade** quand un ordre stable est requis (affichage, tests) : extraire les clés et les
trier. L'idiome moderne tient en une ligne (🆕 1.21/1.23) :

```go
for _, k := range slices.Sorted(maps.Keys(m)) {
	fmt.Println(k, m[k])   // parcours déterministe, trié par clé
}
```

> 💡 `maps.Keys(m)` renvoie un **itérateur** (`iter.Seq`, voir [Ch. 18](18-iterateurs.md)),
> que `slices.Sorted` consomme pour produire une tranche triée. Pas besoin de boucle
> intermédiaire.

### L'idiome **set** : `map[T]struct{}`

Go n'a pas de type ensemble dédié. On utilise une map dont la valeur est `struct{}` — un type
qui n'occupe **aucun octet** : seule la clé compte.

```go
seen := map[string]struct{}{}
seen["go"] = struct{}{}       // ajouter
_, ok := seen["go"]            // tester l'appartenance
delete(seen, "go")            // retirer
```

> 💡 `struct{}` (zéro octet) signale clairement « la valeur ne porte aucune information ». On
> verra `map[T]bool` comme alternative plus lisible quand on a besoin de `m[x]` directement en
> condition — au prix d'un octet par entrée.

### Le package `maps` (🆕 1.21)

Helpers génériques (voir [Ch. 11](11-genericite.md)) :

```go
maps.Clone(m)        // copie de surface, indépendante
maps.Equal(a, b)     // égalité clé par clé
maps.Copy(dst, src)  // fusionne src dans dst
maps.DeleteFunc(m, func(k string, v int) bool { return v == 0 })
maps.Keys(m)         // itérateur sur les clés (1.23) — cf. slices.Sorted ci-dessus
```

---

## Partie 2 — Les strings

Une **string** est une **séquence d'octets en lecture seule** (immuable). Son contenu textuel
est, par convention, encodé en **UTF-8**. C'est le point qui surprend le plus en venant
d'autres langages : une string n'est **pas** un tableau de caractères.

### `byte`, `rune` et UTF-8

- **`byte`** = alias de `uint8` : un **octet** brut.
- **`rune`** = alias de `int32` : un **point de code Unicode** (un « caractère »).
- **UTF-8** encode chaque rune sur **1 à 4 octets**. L'ASCII (U+0000–U+007F) tient sur 1 octet :
  une string ASCII a donc autant d'octets que de caractères — d'où la confusion fréquente.

```
   UTF-8 : nombre d'octets selon le point de code
   ------------------------------------------------------------------
   U+0000   .. U+007F   : 1 octet    0xxxxxxx                            (ASCII : 'A', '0', '?')
   U+0080   .. U+07FF   : 2 octets   110xxxxx 10xxxxxx                   ('é', 'ç', 'µ')
   U+0800   .. U+FFFF   : 3 octets   1110xxxx 10xxxxxx 10xxxxxx          ('世', '€')
   U+10000  .. U+10FFFF : 4 octets   11110xxx 10xxxxxx 10xxxxxx 10xxxxxx (emoji '🚀')
```

Exemple concret — la string `"café"` (le `é` n'est **pas** de l'ASCII) :

```
   "café"   ->  len = 5 OCTETS, mais seulement 4 runes

   index octet :   0      1      2      3      4
   octet (hex) :  0x63   0x61   0x66   0xC3   0xA9
                 [ 'c' ][ 'a' ][ 'f' ][    'é'     ]   <- 'é' (U+00E9) = 2 octets : 0xC3 0xA9
   index rune  :   0      1      2          3
```

### `len`, indexation, et `range`

| Opération                   | Renvoie                                         | Unité      |
| --------------------------- | ----------------------------------------------- | ---------- |
| `len(s)`                    | nombre d'**octets**                             | octet      |
| `s[i]`                      | l'**octet** à la position `i` (`byte`)          | octet      |
| `for i, r := range s`       | `i` = index d'**octet**, `r` = **rune** décodée | octet→rune |
| `utf8.RuneCountInString(s)` | nombre de **runes**                             | rune       |

```go
s := "café"
len(s)                      // 5  (octets !)
utf8.RuneCountInString(s)   // 4  (runes)
s[3]                        // 0xC3 -> le PREMIER octet de 'é', pas 'é' !
```

⚠️ **Indexer une string donne un octet, pas un caractère.** Pour parcourir des **caractères**,
utilisez `range` (qui décode l'UTF-8) :

```go
for i, r := range "café" {
	fmt.Printf("i=%d %q\n", i, r)
}
// i=0 'c'  | i=1 'a'  | i=2 'f'  | i=3 'é'
//                                  ^ i saute ensuite à 5 : 'é' occupe les octets 3 et 4
```

### Conversions `[]byte` et `[]rune`

```go
b := []byte(s)   // copie les octets bruts        -> [99 97 102 195 169]
r := []rune(s)   // décode l'UTF-8 en points de code -> [99 97 102 233]
string(b)        // reconstruit la string depuis des octets
string(r)        // reconstruit la string depuis des runes
```

> ⚠️ `string(i)` où `i` est un **entier** convertit le **point de code** en string, pas le
> nombre en son écriture décimale : `string(65)` vaut `"A"`, **pas** `"65"` ! Pour
> « 65 » → `"65"`, utilisez `strconv.Itoa(65)`. (Le compilateur `vet` met d'ailleurs en garde
> contre `string(int)`.)

### Immutabilité

Le contenu d'une string ne peut **pas** être modifié en place :

```go
s := "café"
s[0] = 'C'   // ERREUR de compilation : cannot assign to s[0]
```

Pour « modifier », on construit une **nouvelle** string — via `[]rune`/`[]byte` ou
`strings.Builder` (voir plus bas). C'est cette immutabilité qui rend le partage de strings
sûr et bon marché (le détail au [Ch. 31](31-strings-profondeur.md)).

### Les packages du texte

**`strings`** — manipulation de strings (toutes renvoient de **nouvelles** strings) :

```go
strings.ToUpper("go")               // "GO"
strings.Contains("golang", "lang")  // true
strings.HasPrefix("gopher", "go")   // true
strings.Split("a,b,c", ",")         // ["a" "b" "c"]
strings.Fields("  go  rust ")       // ["go" "rust"]  (découpe sur les espaces)
strings.Join([]string{"x", "y"}, "/")  // "x/y"
strings.ReplaceAll("a.b", ".", "/") // "a/b"
strings.TrimSpace("  hi  ")         // "hi"
```

**`strconv`** — conversions texte ↔ nombres/booléens (avec **gestion d'erreur**) :

```go
n, err := strconv.Atoi("42")          // 42, nil    (string -> int)
s := strconv.Itoa(255)                // "255"      (int -> string)
f, err := strconv.ParseFloat("3.14", 64)
b, err := strconv.ParseBool("true")
q := strconv.Quote("a\tb")            // "\"a\\tb\""  (string échappée et entre guillemets)
```

**`unicode/utf8`** — inspection de l'encodage :

```go
utf8.RuneCountInString(s)   // nombre de runes
utf8.ValidString(s)         // l'UTF-8 est-il valide ?
r, size := utf8.DecodeRuneInString("é")  // r='é', size=2 (octets consommés)
```

**`unicode`** — classification de runes : `unicode.IsLetter`, `IsDigit`, `IsSpace`,
`ToUpper`, `ToLower`…

### `strings.Builder` : concaténer efficacement

Concaténer avec `+=` dans une boucle est un **piège de performance** : chaque `+` alloue une
**nouvelle** string et recopie tout (les strings sont immuables). `strings.Builder` accumule
dans un buffer **réutilisé** :

```go
var b strings.Builder
b.Grow(32)            // optionnel : pré-alloue si la taille finale est connue
for i := range 3 {
	fmt.Fprintf(&b, "ligne%d;", i)  // ou b.WriteString / b.WriteByte / b.WriteRune
}
result := b.String()  // "ligne0;ligne1;ligne2;"
```

### Dates & heures : `time.Time`, formatage & parsing

Une date est un `time.Time` (package `time`). Sa conversion en texte est l'un des
points les plus **déroutants** de Go pour les nouveaux venus : au lieu des codes
classiques (`%Y-%m-%d`, `dd/MM/yyyy`…), Go décrit un format en **écrivant une date
de référence précise**, toujours la même :

```
  Mon Jan  2 15:04:05 MST 2006
       1   2  3  4  5       6      (et le fuseau -0700 = 7)
      mois jour h min sec  année
```

La disposition se lit donc comme un **exemple** : « voici à quoi ressemble cette
date-là, mets-y la mienne ». Le moyen mnémotechnique est la suite **1 2 3 4 5 6 7** :
mois (`01`), jour (`02`), heure 12 h (`03`), minute (`04`), seconde (`05`), année
(`06`), fuseau (`-0700`) — l'heure sur 24 h étant `15`.

```go
t := time.Date(2025, time.June, 28, 15, 4, 5, 0, time.UTC)

t.Format("2006-01-02 15:04:05")   // "2025-06-28 15:04:05"
t.Format("02/01/2006")            // "28/06/2025"  (format français)
t.Format(time.RFC3339)            // "2025-06-28T15:04:05Z"

// Parse fait l'inverse : la disposition décrit le format de l'ENTRÉE.
when, err := time.Parse("2006-01-02", "2025-06-28")
// ParseInLocation(layout, valeur, loc) pour fixer le fuseau d'interprétation.
```

> ⚠️ **Seules les valeurs de la date de référence sont « magiques ».** Tout autre
> chiffre est recopié **littéralement** : `t.Format("2024-01-02")` rend
> `"2024-06-28"` (le `2024` est du texte !) — seul `2006`/`06` désigne l'année.

**Tableau exhaustif des éléments** (les seuls reconnus ; rendus pour la référence
`2006-01-02 15:04:05.123456789 -0700`) :

| Composant              | Jeton                             | Rendu                             | Remarque                                       |
| ---------------------- | --------------------------------- | --------------------------------- | ---------------------------------------------- |
| **Année**              | `2006`                            | `2006`                            | 4 chiffres                                     |
|                        | `06`                              | `06`                              | 2 chiffres                                     |
| **Mois**               | `January`                         | `January`                         | nom complet (anglais)                          |
|                        | `Jan`                             | `Jan`                             | nom abrégé (3 lettres)                         |
|                        | `01`                              | `01`                              | numéro, zéro initial                           |
|                        | `1`                               | `1`                               | numéro, sans zéro                              |
| **Jour du mois**       | `02`                              | `02`                              | quantième, zéro initial                        |
|                        | `2`                               | `2`                               | quantième, sans zéro                           |
|                        | `_2`                              | `« 2»`                            | cadré à droite sur 2 colonnes (espace)         |
| **Jour de l'année**    | `002`                             | `002`                             | quantième annuel, zéro initial (3 chiffres)    |
|                        | `__2`                             | `«  2»`                           | quantième annuel, cadré sur 3 colonnes         |
| **Jour de la semaine** | `Monday`                          | `Monday`                          | nom complet                                    |
|                        | `Mon`                             | `Mon`                             | nom abrégé                                     |
| **Heure**              | `15`                              | `15`                              | sur **24 h** (00–23) — **seul** jeton 24 h     |
|                        | `03`                              | `03`                              | sur 12 h, zéro initial                         |
|                        | `3`                               | `3`                               | sur 12 h, sans zéro                            |
| **Minute**             | `04`                              | `04`                              | zéro initial                                   |
|                        | `4`                               | `4`                               | sans zéro                                      |
| **Seconde**            | `05`                              | `05`                              | zéro initial                                   |
|                        | `5`                               | `5`                               | sans zéro                                      |
| **Fraction**           | `.000` / `.000000` / `.000000000` | `.123` / `.123456` / `.123456789` | 3/6/9 décimales, **zéros de fin conservés**    |
|                        | `.9` / `.99` / `.999` …           | `.1` / `.12` / `.123`             | décimales, **zéros de fin supprimés**          |
|                        | `,000` / `,999`                   | `,123`                            | idem, séparateur **virgule**                   |
| **Méridien**           | `PM`                              | `PM`                              | AM/PM majuscules                               |
|                        | `pm`                              | `pm`                              | am/pm minuscules                               |
| **Fuseau**             | `-0700`                           | `-0700`                           | décalage ±hhmm                                 |
|                        | `-07:00`                          | `-07:00`                          | décalage ±hh:mm                                |
|                        | `-07`                             | `-07`                             | décalage ±hh                                   |
|                        | `-070000`                         | `-070000`                         | décalage ±hhmmss                               |
|                        | `-07:00:00`                       | `-07:00:00`                       | décalage ±hh:mm:ss                             |
|                        | `Z0700`                           | `Z` ou `-0700`                    | comme `-0700`, mais **« Z » si UTC**           |
|                        | `Z07:00`                          | `Z` ou `-07:00`                   | comme `-07:00`, mais « Z » si UTC (← RFC 3339) |
|                        | `Z07` / `Z070000` / `Z07:00:00`   | `Z` ou `±…`                       | variantes « Z si UTC »                         |
|                        | `MST`                             | `MST`                             | abréviation (nom) du fuseau                    |

Plutôt que de réécrire ces dispositions, on réutilise les **constantes
prédéfinies** du package `time` :

| Constante                                                | Disposition                     | Exemple de rendu                |
| -------------------------------------------------------- | ------------------------------- | ------------------------------- |
| `time.DateOnly`                                          | `2006-01-02`                    | `2025-06-28`                    |
| `time.TimeOnly`                                          | `15:04:05`                      | `15:04:05`                      |
| `time.DateTime`                                          | `2006-01-02 15:04:05`           | `2025-06-28 15:04:05`           |
| `time.RFC3339`                                           | `2006-01-02T15:04:05Z07:00`     | `2025-06-28T15:04:05Z`          |
| `time.RFC3339Nano`                                       | `…05.999999999Z07:00`           | `2025-06-28T15:04:05.5Z`        |
| `time.RFC1123`                                           | `Mon, 02 Jan 2006 15:04:05 MST` | `Sat, 28 Jun 2025 15:04:05 UTC` |
| `time.Kitchen`                                           | `3:04PM`                        | `3:04PM`                        |
| `time.ANSIC`                                             | `Mon Jan _2 15:04:05 2006`      | `Sat Jun 28 15:04:05 2025`      |
| `time.UnixDate`                                          | `Mon Jan _2 15:04:05 MST 2006`  | `Sat Jun 28 15:04:05 UTC 2025`  |
| `time.Stamp` / `StampMilli` / `StampMicro` / `StampNano` | `Jan _2 15:04:05[.000…]`        | `Jun 28 15:04:05`               |

(Existent aussi : `RFC822`, `RFC822Z`, `RFC850`, `RFC1123Z`, `RubyDate`.)

> 💡 **Le bon réflexe** : pour un format d'API ou de stockage, préférer
> `time.RFC3339` (tri lexicographique = tri chronologique). Le `Format`/`Parse`
> rendent et lisent dans le **fuseau** porté par le `time.Time` ; ajuster avec
> `t.UTC()`, `t.Local()` ou `t.In(loc)` avant d'afficher.

---

## 🆕 Go 1.2x

- **1.21** — built-in `clear` (maps et slices) ; packages `maps` et `slices`.
- **1.23** — `maps.Keys`/`maps.Values` et `slices.Sorted`/`slices.Collect` (basés sur les
  **itérateurs**, [Ch. 18](18-iterateurs.md)).
- **1.24** — variantes **itérateur** sans allocation intermédiaire : `strings.FieldsSeq`,
  `strings.SplitSeq`, `strings.Lines` — `for w := range strings.FieldsSeq(text)` ne
  matérialise pas la tranche de mots.
- **1.24** — les maps reposent désormais sur les **Swiss Tables** (gains mémoire/CPU,
  transparents ; détail [Ch. 32](32-maps-hachage.md)).

## ⚠️ Pièges

- **Écrire dans une `nil` map** → panique. Toujours `make`/`{}` avant d'écrire.
- **Dépendre de l'ordre d'itération** d'une map → l'ordre est randomisé. Triez les clés.
- **Indexer une string** (`s[i]`) croyant obtenir un caractère → on obtient un **octet**.
  Utilisez `range` ou `[]rune`.
- **`len(s)` ≠ nombre de caractères** dès qu'il y a du non-ASCII. Utilisez
  `utf8.RuneCountInString`.
- **`string(monInt)`** ne formate pas le nombre : c'est une conversion de **point de code**.
  Utilisez `strconv.Itoa` / `fmt.Sprint`.
- **Concaténer avec `+=` en boucle** → O(n²) en copies. Utilisez `strings.Builder`.
- **Disposition de date « inventée »** (`YYYY-MM-DD`, `%Y`…) → ne formate rien : Go
  n'a qu'**une** date de référence, `2006-01-02 15:04:05`. Confondre le mois (`01`)
  avec la minute (`04`), ou écrire `12` pour l'heure (c'est `15` en 24 h) = bugs muets.
- **Map partagée entre goroutines** → accès concurrent non protégé = data race (voir
  [Ch. 21](21-synchronisation.md) et [Ch. 32](32-maps-hachage.md)).

## ⚡ Performance

- **Pré-dimensionner** : `make(map[K]V, n)` et `b.Grow(n)` évitent des réallocations quand la
  taille est connue à l'avance.
- **`strings.Builder`** au lieu de `+=` : une seule croissance amortie au lieu d'une copie par
  concaténation.
- Les conversions `[]byte(s)` / `string(b)` **copient** (immutabilité oblige) — les éviter dans
  les boucles chaudes ; optimisations sans copie au [Ch. 31](31-strings-profondeur.md).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch07-maps-strings
go test ./ch07-maps-strings/...
```

À essayer :

1. Exécutez plusieurs fois `go run` et observez l'**ordre changeant** de l'« itération brute »
   de la map (puis la stabilité de la version triée).
2. Remplacez `[]rune(s)` par `[]byte(s)` dans `reverseString` et constatez que `"café"`
   inversé devient de l'**UTF-8 invalide** (le test échoue).
3. Écrivez `wordFrequencies(text string) []struct{ Word string; N int }` triées par fréquence
   décroissante (indice : `slices.SortFunc`).

---

## 📌 À retenir

- Une **map** `map[K]V` : clés **comparables**, accès O(1), itération **randomisée** ; `nil`
  en lecture mais **jamais en écriture**.
- `comma-ok` (`v, ok := m[k]`) distingue absent de zero value ; `delete`/`clear` nettoient.
- `slices.Sorted(maps.Keys(m))` = parcours **déterministe** ; `map[T]struct{}` = **set**.
- Une **string** est une suite d'**octets** UTF-8 **immuable** : `len` et `s[i]` sont en
  octets, `range` et `[]rune` en **runes**.
- `strings.Builder` pour concaténer ; `strconv` pour nombres ↔ texte ; `string(int)` = point
  de code, pas formatage.
- **Dates** : `time.Time.Format`/`time.Parse` décrivent le format avec la **date de
  référence** `2006-01-02 15:04:05` (mnémo **1 2 3 4 5 6 7**) ; privilégier les
  constantes (`time.RFC3339`, `time.DateOnly`…).

## 🔁 Pour aller plus loin

- [Ch. 8 — Structs, méthodes & composition](08-structs-methodes.md).
- [Ch. 31 — Strings en profondeur](31-strings-profondeur.md) : header, conversions sans copie,
  `unique`, `strings.Builder` amorti.
- [Ch. 32 — Maps](32-maps-hachage.md) : buckets, Swiss Tables, croissance.
- [Ch. 18 — Itérateurs](18-iterateurs.md) pour `maps.Keys` / `slices.Sorted` / `FieldsSeq`.
- [Ch. 11 — Généricité](11-genericite.md) pour les packages `maps` et `slices`.
