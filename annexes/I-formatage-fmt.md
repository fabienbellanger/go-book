# Annexe I — Verbes de formatage `fmt`

> **Objectif** — Une référence dense et complète des verbes du paquet `fmt` :
> verbes par catégorie, flags, largeur et précision, verbes spéciaux (`%w`, `%p`,
> `%T`) et pièges courants. À garder ouvert à côté du code. Tous les exemples
> sont compilables et testés dans `code/annexe-I-fmt/`.

---

## La famille `Print`

Tous les verbes ci-dessous s'utilisent avec les fonctions de formatage. Le suffixe
indique la destination et la signature :

| Fonction | Destination | Note |
|----------|-------------|------|
| `fmt.Printf` | sortie standard | format + arguments |
| `fmt.Sprintf` | renvoie une `string` | le plus utilisé hors I/O |
| `fmt.Fprintf` | n'importe quel `io.Writer` | fichiers, buffers, réseau (🔁 [Ch. 41](../chapitres/41-io-flux.md)) |
| `fmt.Errorf` | renvoie une `error` | **seule** à interpréter `%w` (🔁 [Ch. 10](../chapitres/10-erreurs.md)) |
| `fmt.Print` / `Println` | sortie standard | sans format ; espace entre opérandes |

`Sprint`, `Sprintln`, `Fprint`, `Fprintln`, `Sprintf`… déclinent la même logique.

---

## Verbes généraux (tout type)

| Verbe | Effet | Exemple (`point{3, 4}`) |
|-------|-------|--------------------------|
| `%v` | valeur, format par défaut | `{3 4}` |
| `%+v` | comme `%v`, **avec les noms de champs** | `{X:3 Y:4}` |
| `%#v` | syntaxe Go (re-collable dans du code) | `main.point{X:3, Y:4}` |
| `%T` | **type** de la valeur | `main.point` |
| `%%` | un signe pourcent littéral | `%` |

Si le type implémente `fmt.Stringer` (`String() string`) ou `error`
(`Error() string`), `%v` et `%s` passent par cette méthode :

```go
// code/annexe-I-fmt/main.go
fmt.Printf("Stringer -> %v / %s\n", green, green) // vert / vert
```

⚠️ Si `String()`/`Error()` est défini sur un **récepteur pointeur** (`func (c
*T) String() string`), seul `*T` implémente `Stringer` — pas `T` (method set,
🔁 [Ch. 09](../chapitres/09-interfaces.md)). `%v` sur une **valeur** ignore
alors la méthode ; `%v` sur `&valeur` l'utilise :

```go
// code/annexe-I-fmt/main.go
c := celsius(21.5) // String() défini sur *celsius
fmt.Printf("%%v valeur -> %v / %%v pointeur -> %v\n", c, &c) // 21.5 / 21.5°C
```

---

## Booléen

| Verbe | Effet | Exemple |
|-------|-------|---------|
| `%t` | `true` ou `false` | `true` |

---

## Entiers

| Verbe | Base / effet | `255` ou rune `中` |
|-------|--------------|---------------------|
| `%d` | décimal | `255` |
| `%b` | binaire | `11111111` |
| `%o` | octal | `377` |
| `%O` | octal préfixé `0o` | `0o377` |
| `%x` / `%X` | hexadécimal (min./MAJ.) | `ff` / `FF` |
| `%c` | le **caractère** Unicode du code | `中` |
| `%q` | le caractère, **quoté** et échappé | `'中'` |
| `%U` | notation Unicode | `U+4E2D` |

```go
// code/annexe-I-fmt/main.go
const han rune = 0x4E2D // le caractère 中
fmt.Printf("%c %q %U -> %c %q %U\n", han, han, han) // 中 '中' U+4E2D
```

⚠️ Pour `%q`, `%c` et `%U`, l'argument doit être un entier (idéalement un `rune`) :
`go vet` refuse `%q` sur un `int` non typé — d'où le `const ... rune` ci-dessus.

---

## Flottants & complexes

| Verbe | Effet | `1234.5678` |
|-------|-------|-------------|
| `%f` / `%F` | décimal sans exposant | `1234.567800` |
| `%e` / `%E` | notation scientifique | `1.234568e+03` |
| `%g` / `%G` | `%e` pour les grands exposants, sinon `%f` ; **précision minimale** | `1234.5678` |
| `%x` / `%X` | hexadécimal flottant | `0x1.34a456...p+10` |

La précision contrôle le nombre de décimales : `%.2f` → `1234.57`. Pour `%g`,
elle fixe le nombre de **chiffres significatifs**.

---

## Chaînes & tranches d'octets

| Verbe | Effet | `"Go café"` |
|-------|-------|-------------|
| `%s` | la chaîne telle quelle | `Go café` |
| `%q` | chaîne **quotée** et échappée (entre `"`) | `"Go café"` |
| `%x` / `%X` | chaque octet en hexa | `476f20636166c3a9` |

`%x` sur une chaîne montre l'encodage **UTF-8** réel (ici `café` → `63 61 66 c3 a9`).

---

## Pointeurs

| Verbe | Effet | Exemple |
|-------|-------|---------|
| `%p` | adresse en hexadécimal préfixée `0x` | `0xc000123456` |
| `%v` | **déréférence** un pointeur vers struct/array/slice/map et préfixe `&` | `&{3 4}` |

```go
// code/annexe-I-fmt/main.go
fmt.Printf("%%v ptr -> %v\n", &p) // &{3 4}
var pp *point
fmt.Printf("%%v nil ptr -> %v / %%p nil ptr -> %p\n", pp, pp) // <nil> / 0x0
```

💡 `%v` ne déréférence que les pointeurs vers struct/array/slice/map (préfixe
`&`) ; sur un pointeur vers un type scalaire (`*int`, `*string`…), il affiche
l'adresse brute comme `%p`. Sur un pointeur `nil`, `%v` affiche `<nil>` alors
que `%p` affiche `0x0`.

⚠️ L'adresse n'est **pas déterministe** : ne jamais l'asserter dans un test ni
s'en servir comme identité stable.

---

## Le verbe d'erreur `%w`

`%w` n'existe **que** dans `fmt.Errorf` : il enveloppe une erreur tout en restant
**dé-enveloppable** par `errors.Is` / `errors.As` (🔁 [Ch. 10](../chapitres/10-erreurs.md)).
**Pourquoi seulement `Errorf`** : c'est cette fonction, et elle seule, qui
reconnaît `%w` au moment du formatage et construit en retour une `error` qui
mémorise l'erreur enveloppée (via une méthode `Unwrap`) — les autres fonctions
de la famille `Print` ne renvoient pas d'`error` et n'ont donc rien à envelopper.

```go
// code/annexe-I-fmt/main.go
base := errors.New("disque plein")
wrapped := fmt.Errorf("écriture du cache : %w", base)
errors.Is(wrapped, base) // true
```

⚠️ `%w` ailleurs que dans `Errorf` (par ex. `Printf`) produit l'erreur de format
`%!w(...)`. Utiliser `%v` ou `%s` pour seulement **afficher** une erreur.

💡 Depuis **Go 1.20**, `fmt.Errorf` accepte **plusieurs `%w`** dans le même
appel : chaque erreur reste inspectable indépendamment par `errors.Is`/`As`
(🔁 [Ch. 10](../chapitres/10-erreurs.md)) :

```go
// code/annexe-I-fmt/main.go
both := fmt.Errorf("échec combiné : %w / %w", errA, errB)
errors.Is(both, errA) && errors.Is(both, errB) // true, true
```

---

## Flags (modificateurs)

Placés juste après le `%`, avant la largeur :

| Flag | Effet | Exemple |
|------|-------|---------|
| `+` | signe explicite pour les nombres (`+42`) ; champs nommés avec `%+v` | `%+d` → `+42` |
| `-` | **aligne à gauche** (sinon à droite) | `%-6d` → `42····` |
| `0` | remplit avec des **zéros** au lieu d'espaces | `%06d` → `000042` |
| `#` | forme alternative : `0x` pour `%#x`, `0` pour `%#o`, syntaxe Go pour `%#v` | `%#x` → `0xff` |
| (espace) | espace devant les nombres positifs / entre octets de `% x` | `% d` → `·42` |

---

## Largeur & précision

Forme générale : `%[flags][largeur][.précision]verbe`.

| Forme | Sens | Exemple |
|-------|------|---------|
| `%6d` | largeur **minimale** 6 (complétée à gauche) | `[····42]` |
| `%-6d` | largeur 6, aligné à gauche | `[42····]` |
| `%.2f` | précision : 2 décimales | `1234.57` |
| `%6.2f` | largeur 6 **et** 2 décimales | `[····3.14]`… |
| `%.3s` | tronque une chaîne à 3 caractères | `abc` |
| `%*d` | largeur **prise dans les arguments** | `Printf("%*d", 6, 42)` → `[····42]` |
| `%.*f` | précision prise dans les arguments | `Printf("%.*f", 2, x)` |
| `%[1]d` | **indexation explicite** des arguments | `Printf("%[2]d %[1]d", 7, 9)` → `9 7` |

L'indexation `%[n]` est précieuse pour réutiliser un argument ou réordonner sans
toucher à la liste — utile en i18n.

---

## ⚠️ Pièges courants

- **Verbe inadapté au type** → sortie `%!verbe(type=valeur)`, jamais une panique :

  ```go
  // code/annexe-I-fmt/main.go
  fmt.Sprintf("%d", "texte") // -> %!d(string=texte)
  ```

  `go vet` détecte ce cas quand le format est un **littéral** : garder les formats
  littéraux pour en bénéficier.
- **Argument manquant / en trop** → `%!d(MISSING)` ou `%!(EXTRA int=7)`.
- **`%v` vs `%+v` vs `%#v`** : en debug, préférer `%+v` (noms de champs) ou `%#v`
  (type + syntaxe Go) ; `%v` seul masque souvent l'information utile.
- **Récursion de `String()`/`Error()`** : si la méthode utilise `%v` sur son propre
  receveur, elle s'appelle elle-même à l'infini. Convertir vers le type sous-jacent
  (`fmt.Sprintf("%d", int(c))`) pour couper la récursion.
- **`%s` sur une `[]byte`** fonctionne (affiche le texte) ; **`%d` sur une `[]byte`**
  affiche la tranche d'entiers `[71 111 ...]`.
- **`nil` et vide sont indiscernables avec `%v`** : une slice/map `nil` et une
  slice/map vide s'affichent toutes les deux `[]` / `map[]`. Seul `== nil`
  distingue les deux à l'exécution — ne pas se fier au formatage pour ce test.

  ```go
  // code/annexe-I-fmt/main.go
  var nilSlice []int
  fmt.Printf("%%v nil slice -> %v / vide -> %v\n", nilSlice, []int{}) // [] / []
  ```
- **Les clés d'une `map` sont triées** avant affichage par `%v` (tri **déterministe**
  depuis Go 1.12) : `map[a:1 b:2 c:3]`, jamais un ordre aléatoire — pratique pour
  comparer une sortie dans un test.

---

## 🧪 À tester soi-même

L'annexe est rendue exécutable : la table de référence est vérifiée par un test.

```bash
cd code && go test ./annexe-I-fmt/...   # vérifie chaque verbe
go run ./annexe-I-fmt                    # affiche la démonstration complète
```

---

## 📌 À retenir

- `%v`/`%+v`/`%#v`/`%T` couvrent 90 % des besoins de debug ; `%+v` montre les noms
  de champs, `%#v` la syntaxe Go, `%T` le type.
- Entiers : `%d %b %o %x`, plus `%c`/`%q`/`%U` pour les runes (typer en `rune`).
- Flottants : `%f` (décimales fixes), `%e` (scientifique), `%g` (compact).
- Chaînes : `%s` brut, `%q` quoté, `%x` octets UTF-8.
- `%w` **uniquement** dans `fmt.Errorf` pour envelopper une erreur ré-inspectable
  (plusieurs `%w` possibles depuis Go 1.20).
- `%v` déréférence un pointeur vers struct/array/slice/map (`&{...}`) ; `nil` et
  vide sont indiscernables pour une slice/map.
- Largeur/précision/flags se combinent : `%-08.2f`, `%*d`, indexation `%[n]`.
- Un verbe inadapté donne `%!verbe(...)`, jamais une panique — et `go vet` le
  repère sur les formats littéraux.

## 🔁 Pour aller plus loin

- [Ch. 10 — Gestion des erreurs](../chapitres/10-erreurs.md) pour `%w`, `errors.Is`/`As`.
- [Ch. 09 — Interfaces](../chapitres/09-interfaces.md) et [Ch. 33](../chapitres/33-interfaces-profondeur.md) pour `fmt.Stringer`.
- [Ch. 41 — Entrées/sorties & flux](../chapitres/41-io-flux.md) pour `Fprintf` et les `io.Writer`.
- Documentation officielle : [pkg.go.dev/fmt](https://pkg.go.dev/fmt) (section « Printing »).
