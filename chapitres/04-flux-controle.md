# Ch. 4 — Flux de contrôle

> **Objectif** — Maîtriser les branchements (`if`, `switch`) et les boucles (`for`,
> `range`) idiomatiques de Go.
>
> **Prérequis** — [Ch. 3 — Variables, constantes & types de base](03-variables-constantes-types.md)

---

## Introduction

Go réduit le flux de contrôle à l'essentiel : **un seul mot-clé de boucle** (`for`), un
`if` sans parenthèses, et un `switch` particulièrement souple. Pas de `while`, pas de
`do…while`, pas d'opérateur ternaire. Moins de formes à connaître, donc moins
d'ambiguïté à la lecture.

L'exemple complet est dans [`code/ch04-controlflow/`](../code/ch04-controlflow/).

## `if` / `else`

Pas de parenthèses autour de la condition, **accolades obligatoires** (même pour une
seule ligne) :

```go
if score >= 60 {
	fmt.Println("réussi")
} else {
	fmt.Println("échoué")
}
```

### `if` avec instruction d'initialisation

On peut exécuter une **instruction courte** avant la condition ; la variable déclarée
n'existe que dans le `if`/`else` :

```go
if v := classify(85); v == "B" {
	fmt.Println(v) // v visible ici…
}
// … mais plus ici : v est hors de portée.
```

> 💡 C'est l'idiome pour tester une erreur sans polluer la portée englobante :
>
> ```go
> if err := doSomething(); err != nil {
> 	return err
> }
> ```

## `for` : l'unique boucle

`for` prend **trois formes**, qui couvrent tous les besoins :

```go
// 1. Forme complète : init ; condition ; post
for i := 0; i < n; i++ { … }

// 2. Condition seule (le « while » des autres langages)
for sum < 10 { … }

// 3. Sans clause : boucle infinie (sortie par break/return)
for { … }
```

```
   Table de décision — quelle forme de for ?
   -----------------------------------------
   Compter / indexer .................. for i := 0; i < n; i++ { }
   Boucler tant qu'une condition tient . for cond { }
   Boucler N fois (sans indice utile) .. for range n { }       (🆕 1.22)
   Parcourir une collection ........... for i, v := range coll { }
   Boucle événementielle / serveur .... for { select { … } }   (voir Ch. 20)
```

## `range` : parcourir une séquence

`range` itère sur **slices/arrays, maps, strings, channels** et, depuis 1.22, **entiers**.
Les valeurs renvoyées dépendent de la source :

| Source               | `for k, v := range x` donne…                        |
| -------------------- | --------------------------------------------------- |
| slice / array        | `k` = index, `v` = **copie** de l'élément           |
| map                  | `k` = clé, `v` = valeur (**ordre aléatoire** ⚠️)    |
| string               | `k` = **index octet**, `v` = `rune` (point de code) |
| channel              | `v` = valeur reçue (jusqu'à fermeture)              |
| entier `n` (🆕 1.22) | `k` = 0, 1, …, n−1                                  |

```go
for i, r := range "héllo" {
	fmt.Printf("[%d:%c] ", i, r)
}
// [0:h] [1:é] [3:l] [4:l] [5:o]   <-- 'é' fait 2 octets : l'index saute de 1 à 3
```

On **ignore** une valeur dont on n'a pas besoin avec `_`, ou on omet la seconde :

```go
for _, v := range s { … } // juste les valeurs
for i := range s { … }    // juste les index
for range s { … }         // juste le nombre d'itérations
```

> ⚠️ **`v` est une copie.** Modifier `v` dans `for _, v := range s` ne change **pas** le
> slice. Pour muter en place, indexez : `s[i] = …`.

## `switch` : la sélection souple

Le `switch` de Go est plus puissant qu'ailleurs. Points clés :

- **Pas de `fallthrough` implicite** : chaque cas s'arrête tout seul (pas besoin de
  `break`).
- Les cas peuvent être des **valeurs quelconques** (pas seulement des constantes).
- Un cas peut lister **plusieurs valeurs**.

```go
switch jour {
case "samedi", "dimanche": // plusieurs valeurs
	fmt.Println("week-end")
default:
	fmt.Println("semaine")
}
```

### `switch` sans condition

Sans expression, `switch` équivaut à une **cascade `if / else if`** plus lisible — chaque
cas est une condition booléenne (utilisé par `classify` dans l'exemple) :

```go
switch {
case score >= 90:
	return "A"
case score >= 80:
	return "B"
default:
	return "F"
}
```

### `fallthrough` & init

`fallthrough` force le passage **au cas suivant** (rare). On peut aussi initialiser :

```go
switch v := f(); { // instruction d'init, puis switch sans condition
case v > 0:
	fmt.Print("positif ")
	fallthrough // exécute AUSSI le cas suivant
case v > -10:
	fmt.Print("> -10")
}
```

> 🔁 Le **type switch** (`switch x.(type)`) et la génération de code (jump table vs
> comparaisons) sont détaillés au [Ch. 14](14-switch.md).

## `break`, `continue`, étiquettes, `goto`

- `break` sort de la boucle (ou du `switch`) **la plus interne**.
- `continue` passe à l'**itération suivante**.
- Une **étiquette** permet de viser une boucle externe — indispensable pour sortir de
  boucles imbriquées :

```go
search:
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c] == target {
				break search // sort des DEUX boucles
			}
		}
	}
```

- `goto` existe (saut vers une étiquette **dans la même fonction**), mais reste **rare** ;
  on ne peut pas sauter par-dessus une déclaration de variable vers sa portée.

## 🆕 Go 1.22 : deux changements majeurs

### `for range N` — itérer sur un entier

```go
for i := range 5 { // i : 0,1,2,3,4
	fmt.Print(i, " ")
}
```

Plus besoin de `for i := 0; i < 5; i++` quand l'indice suffit.

### Portée **par itération** de la variable de boucle

Avant 1.22, la variable de boucle était **unique** et **partagée** par toutes les
itérations — source du bug le plus célèbre de Go, avec les closures et les goroutines.
Depuis 1.22, **chaque itération a sa propre copie** :

```go
funcs := make([]func() int, 0, 3)
for i := range 3 {
	funcs = append(funcs, func() int { return i })
}
for _, f := range funcs {
	fmt.Print(f(), " ") // 1.22+ : 0 1 2     (avant : 3 3 3)
}
```

> 🔁 Ce piège historique et sa disparition sont détaillés au [Ch. 15 — Closures](15-closures.md),
> et ses implications pour les goroutines au [Ch. 19](19-goroutines.md).

## ⚠️ Pièges

- **Valeur de `range` copiée** — `for _, v := range s { v.x = … }` ne modifie rien ;
  indexez `s[i]`.
- **Ordre d'itération d'une map aléatoire** — ne dépendez jamais de l'ordre (c'est
  volontaire, voir Ch. 7). Triez les clés si l'ordre compte.
- **`break` dans un `select`/`switch`** — il sort du `switch`/`select`, pas de la boucle
  englobante : utilisez une **étiquette** pour cette dernière.
- **Index octet vs rune** — sur une string, l'index de `range` est en **octets**, pas en
  caractères (voir Ch. 7).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch04-controlflow
go test ./ch04-controlflow/...
```

À essayer :

1. Réécrivez `classify` avec un `switch score / 10 { case 10, 9: … }` (switch d'expression).
2. Supprimez l'étiquette de `firstPair` et remplacez `break search` par `break` :
   observez que seule la boucle interne est quittée (le résultat devient faux).
3. Ajoutez un `fallthrough` dans `classify` et constatez l'effet en cascade.

---

## 📌 À retenir

- Un seul mot-clé de boucle : **`for`**, en trois formes (complète, condition, infinie).
- `if`/`switch`/`for` acceptent une **instruction d'initialisation** à portée locale.
- `range` parcourt slices, maps, strings, channels et **entiers** (🆕 1.22) ; la valeur
  est une **copie**.
- `switch` **ne tombe pas** d'un cas à l'autre ; sans condition, il remplace `if/else if`.
- **Étiquettes** pour sortir de boucles imbriquées ; `goto` existe mais reste rare.
- 🆕 1.22 : `for range N` et **portée par itération** (fin du piège de capture).

## 🔁 Pour aller plus loin

- [Ch. 5 — Fonctions](05-fonctions.md).
- [Ch. 14 — `switch` & sélection de cas](14-switch.md) : type switch, jump tables.
- [Ch. 15 — Closures](15-closures.md) : la capture de variables de boucle en détail.
