# Annexe J — Expressions régulières (`regexp`)

> **Objectif** — Une référence dense du paquet `regexp` : le moteur **RE2** et ses
> garanties, la compilation, l'API de recherche, les sous-groupes nommés, le
> remplacement et le découpage, la table de syntaxe et les pièges courants. À
> garder ouvert à côté du code. Tous les exemples sont compilables et testés dans
> `code/annexe-J-regexp/`.

---

## Le moteur : RE2, pas PCRE

`regexp` n'implémente **pas** les expressions régulières « Perl » (PCRE). Il repose
sur **RE2** : la recherche est faite par un automate fini, garantissant un temps
**linéaire** en la taille de l'entrée — jamais d'explosion exponentielle.

| Propriété | RE2 (`regexp`) | PCRE (Perl, Python `re`, PHP, JS) |
|-----------|----------------|-----------------------------------|
| Complexité | **linéaire** garantie, `O(n·m)` | peut être **exponentielle** (backtracking) |
| ReDoS (déni de service par regex) | **impossible** par construction | risque réel sur entrée hostile |
| Backreferences (`\1`) | ❌ non supportées | oui |
| Lookahead / lookbehind (`(?=…)`, `(?<=…)`) | ❌ non supportés | oui |
| Sous-groupes, alternance, classes, quantifieurs | ✔ | ✔ |

C'est un choix **délibéré** : en abandonnant backreferences et lookaround (qui
imposent le backtracking), Go obtient une garantie de complexité qui rend `regexp`
**sûr sur une entrée non maîtrisée** (formulaire, en-tête HTTP, log). Si un motif
« qui marche ailleurs » est refusé à la compilation, c'est presque toujours un
lookaround ou une backreference.

---

## Compiler : `MustCompile` vs `Compile`

Un `*regexp.Regexp` est **sûr en concurrence** et se **réutilise** : compilez le
motif **une seule fois**, idéalement dans une variable au niveau du paquet.

| Fonction | Renvoie | Quand |
|----------|---------|-------|
| `regexp.MustCompile(expr)` | `*Regexp` (**panique** si invalide) | motif **littéral**, fixé dans le code — l'erreur serait un bug |
| `regexp.Compile(expr)` | `(*Regexp, error)` | motif **dynamique** (venu de l'utilisateur, d'un fichier) |
| `regexp.CompilePOSIX` / `MustCompilePOSIX` | idem | sémantique POSIX *leftmost-longest* |

```go
// code/annexe-J-regexp/main.go
var reDate = regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
```

⚠️ **Ne jamais compiler dans une boucle ni à chaque appel de fonction** : la
compilation est coûteuse, la recherche ne l'est pas. Un motif recompilé à chaque
itération est l'erreur de performance n°1 avec `regexp` (voir ⚡).

---

## API de recherche

Toutes les méthodes existent en deux familles : sur `string` (suffixe `String`) et
sur `[]byte` (sans suffixe). Choisir selon la donnée en main pour éviter une
conversion (🔁 [Ch. 31](../chapitres/31-strings-profondeur.md)).

| Méthode (`string`) | Renvoie | Rôle |
|--------------------|---------|------|
| `MatchString(s)` | `bool` | y a-t-il **au moins** une correspondance ? |
| `FindString(s)` | `string` | la **1re** correspondance (`""` si aucune) |
| `FindStringIndex(s)` | `[]int{début, fin}` | positions de la 1re correspondance (`nil` si aucune) |
| `FindAllString(s, n)` | `[]string` | les `n` premières (**`-1`** = toutes ; `nil` si aucune) |
| `FindStringSubmatch(s)` | `[]string` | correspondance **+ sous-groupes** (`[0]`=tout) |
| `FindStringSubmatchIndex(s)` | `[]int` | positions de chaque sous-groupe |
| `FindAllStringSubmatch(s, n)` | `[][]string` | toutes les correspondances avec leurs sous-groupes |

L'équivalent `[]byte` : `Match`, `Find`, `FindIndex`, `FindAll`, `FindSubmatch`…
Et le raccourci hors objet `regexp.MatchString(pattern, s)` **compile à chaque
appel** — pratique pour un one-shot, à proscrire sur un chemin chaud.

```go
// code/annexe-J-regexp/main.go
reWord := regexp.MustCompile(`\w+`)
reWord.FindString("  Go 1.26 !") // "Go"
reWord.FindAllString("un deux trois", -1) // ["un" "deux" "trois"]
```

💡 **`nil` vs `""`** : les méthodes `Find…` qui renvoient un slice renvoient `nil`
en l'absence de correspondance ; celles qui renvoient une `string` renvoient `""`.
Testez `== nil` (slice) plutôt que la longueur si la distinction compte.

---

## Sous-groupes nommés

`(?P<name>…)` nomme un groupe. `FindStringSubmatch` renvoie `m[0]` (toute la
correspondance) puis un élément par groupe, dans l'ordre. `SubexpNames()` donne le
nom associé à chaque index (`""` pour les groupes anonymes et l'index 0).

```go
// code/annexe-J-regexp/main.go
m := reDate.FindStringSubmatch("release 2026-07-04 ok") // [2026-07-04 2026 07 04]
names := reDate.SubexpNames()                            // ["" "year" "month" "day"]
```

Une fois combinés, on obtient un accès par nom robuste au réordonnancement des
groupes dans le motif :

```go
// code/annexe-J-regexp/main.go
byName := map[string]string{}
for i, name := range names {
	if name != "" {
		byName[name] = m[i]
	}
}
// byName["year"] == "2026"
```

---

## Ancrage : chercher vs valider

`MatchString` cherche une **sous-chaîne** : `\d+` matche `"abc42"`. Pour **valider
tout le champ**, ancrez avec `^…$` — sinon vous validez « contient », pas « est ».

```go
// code/annexe-J-regexp/main.go
var reLogLevel = regexp.MustCompile(`(?i)^(debug|info|warn|error)$`)
reLogLevel.MatchString("INFO")  // true  (?i) = casse ignorée
reLogLevel.MatchString("info ") // false : l'espace final tombe hors de ^…$
```

---

## Remplacement

| Méthode | Remplacement | Note |
|---------|--------------|------|
| `ReplaceAllString(s, repl)` | chaîne, avec `$1` / `${name}` | expansion des références de groupe |
| `ReplaceAllLiteralString(s, repl)` | chaîne **littérale** | `$` non interprété |
| `ReplaceAllStringFunc(s, f)` | `func(match string) string` | logique arbitraire par correspondance |

Dans le motif de remplacement, `$1` ou `${name}` réinjecte un groupe capturé.
Préférez la forme **`${name}`** (accolades) pour éviter l'ambiguïté quand un
chiffre ou une lettre suit la référence.

```go
// code/annexe-J-regexp/main.go
reDate.ReplaceAllString("2026-07-04", "${day}/${month}/${year}") // "04/07/2026"
```

Pour une transformation qui n'est pas une simple réécriture de gabarit,
`ReplaceAllStringFunc` reçoit chaque correspondance :

```go
// code/annexe-J-regexp/main.go
reDigits := regexp.MustCompile(`\d+`)
reDigits.ReplaceAllStringFunc("carte 4242 42", func(m string) string {
	return strings.Repeat("*", len(m)) // "carte **** **"
})
```

---

## Découpage

`Split(s, n)` découpe `s` sur chaque correspondance du motif (`-1` = pas de
limite) — l'inverse de `FindAll`.

```go
// code/annexe-J-regexp/main.go
var reSpaces = regexp.MustCompile(`\s+`)
reSpaces.Split("a  b   c", -1) // ["a" "b" "c"]
```

💡 Pour découper sur **une** espace simple, `strings.Fields` suffit et évite la
regex (voir ⚡).

---

## Table de syntaxe

| Élément | Signifie |
|---------|----------|
| `.` | n'importe quel caractère **sauf `\n`** (voir `(?s)`) |
| `^` `$` | début / fin de **texte** (ou de **ligne** avec `(?m)`) |
| `\b` `\B` | frontière / non-frontière de mot |
| `\d` `\D` | chiffre / non-chiffre |
| `\w` `\W` | caractère de mot (`[0-9A-Za-z_]`) / son complément |
| `\s` `\S` | espace / non-espace |
| `[abc]` `[^abc]` | classe / classe négative |
| `[a-z]` | intervalle |
| `x*` `x+` `x?` | 0+, 1+, 0 ou 1 (**gourmands**) |
| `x{n}` `x{n,}` `x{n,m}` | répétition exacte / au moins / bornée |
| `x*?` `x+?` `x??` | versions **paresseuses** (le moins possible) |
| `a\|b` | alternance |
| `( … )` | groupe **capturant** |
| `(?: … )` | groupe **non capturant** (regroupe sans mémoriser) |
| `(?P<name> … )` | groupe **nommé** |
| `(?i)` | insensible à la **casse** |
| `(?m)` | mode **multiligne** : `^`/`$` collent à chaque ligne |
| `(?s)` | *dotall* : `.` matche **aussi** `\n` |

Les flags `(?ims)` se placent en tête du motif (portée globale) ou dans un groupe
`(?i:…)` (portée locale).

⚠️ Dans un littéral Go, préférez les **backquotes** `` `\d+` `` aux guillemets
`"\\d+"` : sans elles, chaque `\` doit être doublé.

💡 **Gourmand par défaut.** `<.*>` sur `"<a><b>"` capture `"<a><b>"` en entier ;
`<.*?>` (paresseux) capture `"<a>"`. Autre parade RE2-compatible : une classe
négative `<[^>]*>`.

---

## Sémantique de correspondance

Par défaut, `regexp` renvoie la correspondance **la plus à gauche** puis, à
position égale, celle que l'automate atteint en premier (sémantique *leftmost-first*,
proche de Perl). `Regexp.Longest()` (ou `CompilePOSIX`) bascule en *leftmost-longest*
(POSIX) : `(a|ab)` sur `"ab"` renvoie `"ab"` au lieu de `"a"`. Le paquet
`regexp/syntax` expose l'arbre du motif pour l'outillage — usage rare.

---

## ⚠️ Pièges courants

- **Recompiler en boucle** : `regexp.MustCompile(p)` à chaque itération domine le
  temps d'exécution. Compilez une fois en variable de paquet.
- **Attendre des backreferences / lookaround** : absents de RE2. Un motif PCRE
  copié tel quel peut être **refusé à la compilation** — réécrivez sans lookaround.
- **`.` qui ne prend pas `\n`** : par défaut `.` s'arrête aux sauts de ligne.
  Utilisez `(?s)` pour un texte multiligne.
- **Oublier d'ancrer** : `MatchString` teste « contient une correspondance », pas
  « la chaîne entière correspond ». Pour valider un champ : `^…$`.
- **Regex là où `strings` suffit** : `strings.Contains`, `HasPrefix`, `HasSuffix`,
  `Index`, `Fields` (🔁 [Ch. 07](../chapitres/07-maps-strings.md),
  [Ch. 31](../chapitres/31-strings-profondeur.md)) sont plus lisibles **et** bien
  plus rapides pour un besoin fixe (pas de motif).

---

## ⚡ Performance

- **Compiler une fois** (variable de paquet, `init`) : c'est l'optimisation qui
  compte le plus (🔁 [Ch. 40](../chapitres/40-methodologie-performance.md)).
- **Préférer les variantes `Index` / `[]byte`** : `FindStringIndex` renvoie des
  positions sans allouer de sous-chaînes ; travailler en `[]byte` évite une
  conversion quand la donnée est déjà des octets (🔁 [Ch. 26](../chapitres/26-allocation-escape.md)).
- **`regexp` vs `strings`** : pour une recherche fixe (préfixe, sous-chaîne
  littérale), `strings.*` est souvent **plusieurs fois** plus rapide — mesurez
  avant de dégainer une regex (🔁 [Ch. 36](../chapitres/36-tests-benchmarks-fuzzing.md)).
- **`Longest()` est plus lent** : ne l'activez que si la sémantique POSIX est requise.

---

## 🧪 À tester soi-même

L'annexe est rendue exécutable : chaque exemple est vérifié par un test.

```bash
cd code && go test ./annexe-J-regexp/...   # match, sous-groupes, remplacement, split
go run ./annexe-J-regexp                   # affiche la démonstration
```

---

## 📌 À retenir

- `regexp` = **RE2** : temps **linéaire** garanti, **pas** de backtracking, donc
  **pas** de backreferences ni de lookaround — mais aucun risque de ReDoS.
- **Compiler une fois** avec `MustCompile` (motif littéral) ou `Compile` (motif
  dynamique) ; le `*Regexp` est réutilisable et sûr en concurrence.
- Recherche : `MatchString` (bool), `FindString`/`FindAllString` (texte),
  `FindStringSubmatch` (+ sous-groupes), variantes `[]byte` sans suffixe.
- **Groupes nommés** `(?P<name>…)` + `SubexpNames()` pour un accès robuste.
- Remplacement `ReplaceAllString` (`${name}`) ou `ReplaceAllStringFunc` (logique) ;
  découpage `Split`.
- Ancrer avec `^…$` pour **valider** ; flags `(?ims)` ; `.` ignore `\n` sauf `(?s)`.
- Quand un besoin est **fixe**, `strings.*` est plus lisible et plus rapide qu'une regex.

## 🔁 Pour aller plus loin

- [Ch. 07 — Maps & strings](../chapitres/07-maps-strings.md) et [Ch. 31 — Strings en profondeur](../chapitres/31-strings-profondeur.md) : les alternatives `strings.*`.
- [Ch. 40 — Méthodologie de performance](../chapitres/40-methodologie-performance.md) et [Ch. 36 — Tests, benchmarks & fuzzing](../chapitres/36-tests-benchmarks-fuzzing.md) : mesurer avant d'optimiser un motif.
- [Annexe I — Verbes de formatage `fmt`](I-formatage-fmt.md) : l'autre référence « à garder ouverte ».
- Documentation officielle : [pkg.go.dev/regexp](https://pkg.go.dev/regexp) et [la syntaxe RE2](https://pkg.go.dev/regexp/syntax).
