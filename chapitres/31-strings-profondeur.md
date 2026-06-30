# 31 — Strings en profondeur

> **Objectif** — Comprendre qu'une `string` est un **header de 2 mots** sur un backing **immuable**,
> savoir ce que **coûtent** (ou non) les conversions `string`↔`[]byte`, maîtriser `strings.Builder`
> pour concaténer en **O(n)**, décoder l'**UTF-8** (octets vs runes) et **interner** des chaînes avec
> le package `unique`.
>
> **Prérequis** — [Ch. 7](07-maps-strings.md), [Ch. 30](30-slices-profondeur.md)

---

## Introduction

Une `string` Go ressemble à un slice — un pointeur et une longueur — mais avec une différence
fondamentale : son backing est **immuable**. Cette immuabilité explique tout : pourquoi les chaînes se
partagent sans danger, pourquoi modifier impose une **copie**, et pourquoi la concaténation naïve est un
**piège de performance**. Code dans [`code/ch31-strings-profondeur/`](../code/ch31-strings-profondeur/).

---

## Le header de string : 2 mots immuables

Une `string` fait **16 octets** sur une machine 64 bits : un **pointeur** vers les octets et une
**longueur**. Pas de `cap` — une string ne grandit pas. Le backing est en **lecture seule** (souvent en
mémoire constante du binaire pour les littéraux).

```
  s := "héllo"

  s  (string header = 2 mots = 16 octets)
  +---------+---------+
  |  ptr    |  len=6  |          <- 6 OCTETS (pas 5 : 'é' en prend 2)
  +----+----+---------+
       |
       v   backing IMMUABLE (UTF-8)
   octets : 68 c3 a9 6c 6c 6f     ('h'  'é'(2o)  'l' 'l' 'o')
```

Copier une string copie **16 octets** (le header), jamais les données — d'où des passages par valeur
bon marché. Deux chaînes peuvent **partager** le même backing en toute sécurité, justement parce qu'aucune
ne peut le modifier.

💡 Pour un **littéral** (`"héllo"` écrit dans le source), ce backing vit dans la section lecture seule
du binaire (`.rodata`) : y écrire via `unsafe` ne corrompt pas juste une valeur logique, ça **segfault**
au niveau matériel. Une string **construite à l'exécution** (conversion depuis `[]byte`, `fmt.Sprintf`…)
vit, elle, sur le tas ou la pile ordinaires ; son immutabilité n'est garantie que par le **typage** — rien
au niveau mémoire n'empêche `unsafe` de la modifier, d'où la prudence requise au [Ch. 35](35-unsafe-cgo.md).

### Sous-chaînes : zéro copie, mais rétention possible

`s[i:j]` ne copie **rien** : Go construit un **nouveau header** (pointeur décalé de `i`, `len = j-i`) qui
vise le **même** backing — sûr uniquement parce que ce backing est immuable. C'est ce qui rend
`strings.Cut`, `strings.TrimPrefix` ou un simple slicing **O(1)** en mémoire, quelle que soit la taille
de `s` :

```
  s     := lireFichier()        // backing de 50 Mo
  ligne := s[120:180]           // header DISTINCT, MEME backing

  s      +-----+-----+              ligne  +-----+-----+
         | ptr | len |                     | ptr | len |
         +--|--+-----+                     +--|--+-----+
            |                                  |
            v                                  v
  backing  [0 ............. 120 ........... 180 ............. 50 Mo]
                              ^----- ligne (60 o) -----^
```

⚠️ Tant que `ligne` existe, les **50 Mo entiers** restent en mémoire, même si `ligne` ne fait que 60
octets — exactement le piège de rétention des slices ([Ch. 30](30-slices-profondeur.md)). La parade est
la même : forcer une **copie indépendante**. Pour les strings, c'est **`strings.Clone`** (1.18) :

```go
// code/ch31-strings-profondeur/strdeep.go
func DetachSubstring(s string, start, end int) string {
	return strings.Clone(s[start:end])
}
```

## Octets vs runes : l'UTF-8

`len(s)` compte les **octets**, pas les caractères. Un point de code Unicode (une **rune**) occupe 1 à
4 octets en UTF-8. `range` sur une string **décode** l'UTF-8 : il livre l'**index d'octet** et la
**rune**.

Ce choix de représentation n'est pas arbitraire : stocker un `[]rune` (4 octets fixes par caractère)
gaspillerait 3 octets sur 4 pour du texte majoritairement ASCII — le cas le plus courant — alors que
l'UTF-8 reste **compatible octet à octet** avec l'ASCII (les 128 premiers points de code s'encodent sur
1 octet, identique à leur valeur ASCII). Le coût se déplace : indexer un caractère est O(1) avec
`[]rune`, O(n) avec une string (il faut décoder depuis le début).

```go
// code/ch31-strings-profondeur/strdeep.go
func ByteVsRune(s string) (bytes, runes int) {
	return len(s), utf8.RuneCountInString(s)
}
```

```
$ go run ./ch31-strings-profondeur
"héllo, 日本" : 14 octets, 9 runes
range -> (index octet, largeur) : (0,1) (1,2) (3,1) (4,1) (5,1) (6,1) (7,1) (8,3) (11,3)
```

L'index d'octet **saute** (0, 1, **3**, …) car `é` occupe 2 octets et `日` en occupe 3. Pour indexer par
caractère, convertissez en `[]rune` (au prix d'une allocation). ⚠️ `s[i]` renvoie un **octet** (`byte`),
jamais une rune.

## Conversions `string`↔`[]byte` : quand y a-t-il copie ?

Une string est immuable, un `[]byte` est mutable : convertir doit, **en général**, **copier** — sinon on
pourrait muter une string. Mais le compilateur **élide** la copie dans des cas prouvés sûrs où le résultat
est **consommé sur place** :

| Conversion                      | Copie ? | alloc/op |
| ------------------------------- | ------- | -------- |
| `string(b)` stockée/renvoyée    | **oui** | **1**    |
| `[]byte(s)` stockée/renvoyée    | **oui** | **1**    |
| `m[string(b)]` (lookup map)     | **non** | **0**    |
| `string(b) == s` (comparaison)  | **non** | **0**    |
| `for range string(b)`, `switch` | **non** | **0**    |

Ces trois lignes « non » ne sont pas le fruit d'une analyse générale du flux de données : le compilateur
reconnaît un **nombre fini de formes syntaxiques** précises, écrites **telles quelles** dans
l'expression. Introduire une étape intermédiaire suffit à perdre l'optimisation :

```go
key := string(b)  // copie : la conversion est maintenant stockée dans une variable
v := m[key]        // ce lookup ne « voit » plus le motif m[string(b)]
```

⚠️ Le motif doit apparaître **littéralement** au bon endroit (`m[string(b)]`, `string(b) == s`,
`for range string(b)`, `switch string(b)`) — un refactoring anodin (extraire dans une variable, passer
par une fonction intermédiaire) réintroduit silencieusement l'allocation.

```go
// Modifier une string OBLIGE à passer par []byte : 2 copies (aller + retour).
func ToUpperASCII(s string) string {
	b := []byte(s) // copie n°1
	for i := range b {
		if c := b[i]; c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b) // copie n°2
}
```

💡 Sur un chemin chaud, gardez vos données en `[]byte` d'un bout à l'autre pour éviter les
allers-retours. Le **vrai** zéro-copie (partager le backing) demande `unsafe.String`/`unsafe.Slice`
([Ch. 35](35-unsafe-cgo.md)) — à réserver aux cas mesurés.

## Concaténer : `strings.Builder`, pas `+`

Comme chaque `+` crée une **nouvelle** string et **recopie** tout l'accumulateur, concaténer dans une
boucle est **O(n²)**. `strings.Builder` accumule dans un `[]byte` qui **croît en amorti** (comme
[`append`](30-slices-profondeur.md)), puis livre la string finale **sans copie** :

```go
// code/ch31-strings-profondeur/strdeep.go
func JoinCSV(items []string) string {
	var b strings.Builder
	b.Grow(size) // réserve la taille finale -> une seule montée mémoire
	for i, s := range items {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s)
	}
	return b.String()
}
```

Mesuré pour 500 fragments (`-benchmem`) :

| Variante         | ns/op       | B/op          | allocs/op |
| ---------------- | ----------- | ------------- | --------- |
| `concatPlus` (+) | **117 042** | **1 068 781** | **499**   |
| `concatBuilder`  | **3 861**   | **12 536**    | **12**    |

**30× plus rapide**, **85× moins de mémoire** : le `+` recopie ~1 Mo pour produire 4 Ko. Avec `Grow`,
on tombe encore plus bas. 📌 `strings.Builder` (ou `bytes.Buffer`) est la **règle** dès qu'il y a une
boucle.

## Interning : le package `unique` (1.23)

Beaucoup de chaînes **identiques** en mémoire (clés, symboles, en-têtes) gaspillent de la place et
ralentissent les comparaisons. **`unique.Make`** renvoie un **handle canonique** : deux contenus égaux
donnent le **même** handle, comparable en comparant **un seul pointeur**.

```go
// code/ch31-strings-profondeur/strdeep.go
type Symbol = unique.Handle[string] // 8 octets (1 pointeur)

func Intern(s string) Symbol { return unique.Make(s) }
```

```
Intern("event.created") x2 : handles == ? true ; taille handle = 8 o
```

Gains : **mémoire** (un seul backing partagé par valeur distincte) et **vitesse** (comparer/hacher un
pointeur, pas toute la chaîne). `unique.Make` n'est cependant **pas gratuit** : chaque appel hache le
contenu et consulte une table d'internement globale (verrouillée) — un coût comparable à une écriture de
map. L'opération ne se rentabilise que si le **même** handle sert ensuite **plusieurs fois**
(comparaisons répétées, clé de map réutilisée) ; interner une chaîne qui ne sert qu'une fois coûte
strictement plus cher qu'une comparaison directe.

Idéal comme **clés de map** quand les mêmes chaînes reviennent en masse. Le package gère lui-même la
collecte des valeurs devenues inutilisées (via des références faibles,
[Ch. 27](27-garbage-collector.md)).

---

## 🆕 Go 1.2x

- **1.23** — le package **`unique`** (`Make`, `Handle[T].Value()`) : interning générique et thread-safe,
  pas seulement pour les strings. Remplace les tables d'interning maison.
- **1.24** — `unique` s'appuie sur les **références faibles** internes ; les handles non utilisés sont
  récupérés par le GC, sans fuite.
- **continuité** — les optimisations **sans copie** de conversion (`m[string(b)]`, `string(b)==s`,
  `for range`) restent garanties par le compilateur ; profitez-en au lieu de recourir à `unsafe`.

## ⚠️ Pièges

- **`len(s)` = octets, pas caractères** — pour compter des caractères, `utf8.RuneCountInString`. `s[i]`
  donne un **octet**.
- **`string(monInt)`** — renvoie le **caractère** de ce point de code, pas la représentation décimale !
  Utilisez `strconv.Itoa` (le vet `stringintconv` le signale).
- **Concaténer par `+` en boucle** — O(n²). `strings.Builder` systématiquement.
- **`[]rune(s)` par réflexe** — alloue et copie tout ; n'en faites que si vous indexez vraiment par rune.
- **`unsafe.String` pour gagner une copie** — ne le faites que si vous **garantissez** que le `[]byte`
  source ne sera plus jamais modifié ([Ch. 35](35-unsafe-cgo.md)).
- **Garder une petite sous-chaîne d'un grand texte** — elle retient **tout** le backing en mémoire
  (même piège que les slices, [Ch. 30](30-slices-profondeur.md)). `strings.Clone` pour détacher.
- **Compter sur l'optimisation sans copie après un refactoring** — le motif (`m[string(b)]`...) doit
  rester **syntaxiquement identique** ; le stocker dans une variable intermédiaire la fait disparaître.

## ⚡ Performance

- **Préallouez** le `Builder` avec `Grow(n)` si vous connaissez (même approximativement) la taille finale.
- Restez en **`[]byte`** sur le chemin chaud pour éviter les conversions ; convertissez en `string` une
  seule fois, au bout.
- **Internez** (`unique`) les chaînes répétées **plusieurs fois** : moins de mémoire, comparaisons en
  O(1) — mais seulement si le handle est réutilisé, sinon l'internement coûte plus qu'il ne rapporte.
- Les conversions **consommées sur place** (lookup, comparaison, `range`) sont **gratuites** — le
  compilateur élide la copie. 🔁 [Ch. 35](35-unsafe-cgo.md) pour le zéro-copie explicite.
- **`strings.Clone`** pour libérer un grand backing retenu par une petite sous-chaîne — symétrique de
  `slices.Clone` ([Ch. 30](30-slices-profondeur.md)).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch31-strings-profondeur
go test ./ch31-strings-profondeur/...
go test -bench=. -benchmem -run=^$ ./ch31-strings-profondeur/...
```

À essayer :

1. Mesurez `JoinCSV` **avec** et **sans** `b.Grow(...)` (`-benchmem`) : combien d'allocations en moins ?
2. Comparez `string(b) == s` (0 alloc) et `bytes.Equal(b, []byte(s))` (la 2ᵉ conversion alloue).
3. Internez 1 million de chaînes tirées d'un petit ensemble et comparez la mémoire (`ReadMemStats`) avec/sans `unique`.
4. Construisez une string de plusieurs Mo, gardez-en une sous-chaîne de quelques octets **avec** et
   **sans** `strings.Clone`, forcez un `runtime.GC()` puis comparez `HeapAlloc` (`ReadMemStats`) : sans
   `Clone`, le backing complet reste compté.

---

## 📌 À retenir

- Une `string` = **header de 2 mots** (ptr/len, 16 o) sur un backing **immuable** ; pas de `cap`. Copier
  une string copie le header, jamais les octets.
- **Slicer** (`s[i:j]`) ne copie rien — nouveau header, même backing. Pratique et gratuit, mais une
  petite sous-chaîne **retient** tout un grand backing ; `strings.Clone` (1.18) détache.
- `len` et l'indexation comptent des **octets** (UTF-8) ; `range` **décode** en runes (index d'octet +
  rune). `utf8` pour compter/valider.
- Convertir `string`↔`[]byte` **copie** quand le résultat survit (**1 alloc**), mais est **gratuit**
  quand il est consommé sur place (lookup, comparaison, `range`) — à condition que le motif reste
  **syntaxiquement** celui que le compilateur reconnaît.
- **`strings.Builder`** rend la concaténation **O(n)** : ~30× plus rapide que `+` en boucle. `Grow` pour
  préallouer.
- **`unique.Make`** (1.23) interne : handle canonique de 8 o, comparaison/hachage en O(1), mémoire
  partagée — rentable seulement si le handle est réutilisé.

## 🔁 Pour aller plus loin

- [Ch. 30 — Slices en profondeur](30-slices-profondeur.md) : le même header à 3 mots, mutable cette fois.
- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : pourquoi une copie de conversion coûte.
- [Ch. 32 — Maps](32-maps-hachage.md) : le hachage des clés string, et `unique` comme clé.
- [Ch. 35 — `unsafe` & interop](35-unsafe-cgo.md) : `unsafe.String`/`unsafe.Slice`, le zéro-copie assumé.
- Doc : `go doc strings.Builder` ; `go doc unique` ; `go doc unicode/utf8`.
