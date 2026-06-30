# 4 — Flux de contrôle

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

Ce minimalisme est assumé, pas accidentel : moins de constructions équivalentes, c'est
moins de débats de style, et un `gofmt` qui produit un résultat quasi unique pour une même
logique. L'absence de ternaire (`cond ? a : b`) en particulier n'est pas un oubli — les
abus qu'il permet en C ou en JavaScript (ternaires imbriqués, illisibles) ont pesé dans le
choix de n'avoir qu'un `if` complet : un peu plus long à écrire, mais qui se relit sans
ambiguïté.

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

> 💡 **Ce n'est pas qu'une question de style.** Le lexer de Go insère automatiquement un
> point-virgule en fin de ligne après certains tokens — dont une accolade fermante `}`. Si
> `else` n'était pas obligatoirement sur la **même ligne** que le `}` qui le précède, ce
> point-virgule invisible terminerait le `if` tout seul, et `else` se retrouverait orphelin
> (erreur de compilation). Rendre les accolades obligatoires élimine au passage l'ambiguïté
> du _dangling else_ qui existe en C (« à quel `if` se rattache cet `else` ? »).

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

> ⚠️ Comme tout `:=` (Ch. 3), une variable déclarée dans l'init d'un `if` **masque** une
> variable de même nom dans la portée englobante. `if err := f(); err != nil { … }` juste
> après un `err := g()` crée un **second** `err`, indépendant du premier — un piège
> classique si l'intention était de réutiliser ou de mettre à jour la variable externe.

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

La clause `post` (comme l'`init`) est une **instruction simple**, pas seulement `i++` :
une **affectation multiple** y fonctionne aussi, ce qui permet de faire avancer deux
indices à la fois — utile pour parcourir un slice par les deux bouts :

```go
s := []int{1, 2, 3, 4, 5}
for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
	s[i], s[j] = s[j], s[i] // inverse en place
}
// s == [5 4 3 2 1]
```

> 💡 `i++`/`i--` sont des **instructions**, pas des expressions : impossible d'écrire
> `a[i++]` comme en C. C'est volontaire — Go supprime ainsi toute une classe de bugs liés
> à l'ordre d'évaluation des effets de bord dans une expression. Il n'existe pas non plus
> de forme préfixée `++i` : la forme postfixe est la seule.

Go n'a pas de `do…while`, mais la 3ᵉ forme (boucle infinie) plus un `break` **en fin de
corps** le reconstruit pour les cas où le corps doit s'exécuter **au moins une fois** :

```go
for {
	// … corps exécuté au moins une fois …
	if !continuer {
		break
	}
}
```

## `range` : parcourir une séquence

`range` itère sur **slices/arrays, maps, strings, channels** et, depuis 1.22, **entiers**.
Les valeurs renvoyées dépendent de la source :

| Source               | `for k, v := range x` donne…                                     |
| -------------------- | ---------------------------------------------------------------- |
| slice / array        | `k` = index, `v` = **copie** de l'élément                        |
| map                  | `k` = clé, `v` = **copie** de la valeur (**ordre aléatoire** ⚠️) |
| string               | `k` = **index octet**, `v` = `rune` (point de code)              |
| channel              | `v` = valeur reçue (jusqu'à fermeture)                           |
| entier `n` (🆕 1.22) | `k` = 0, 1, …, n−1                                               |

```go
for i, r := range "héllo" {
	fmt.Printf("[%d:%c] ", i, r)
}
// [0:h] [1:é] [3:l] [4:l] [5:o]   <-- 'é' fait 2 octets : l'index saute de 1 à 3
```

> ⚠️ **L'expression de `range` est évaluée une seule fois**, avant le début de la boucle.
> Sur un **array** (type valeur, voir Ch. 6), cela signifie que `range` travaille sur une
> **copie complète** faite au départ : modifier l'array original _pendant_ la boucle ne
> change pas les valeurs déjà vues par les itérations restantes.
>
> ```go
> arr := [3]int{1, 2, 3}
> for i, v := range arr {
> 	if i == 0 {
> 		arr[1] = 99 // modifie l'array original...
> 	}
> 	fmt.Println(v) // ... mais range travaille sur sa copie : affiche 1 2 3, pas 1 99 3
> }
> ```
>
> Ranger sur un **slice** (en-tête de 3 mots, pas de copie des éléments) ou sur un
> `*[N]T` évite cette copie — voir ⚡ Performance plus bas.

On **ignore** une valeur dont on n'a pas besoin avec `_`, ou on omet la seconde :

```go
for _, v := range s { … } // juste les valeurs
for i := range s { … }    // juste les index
for range s { … }         // juste le nombre d'itérations
```

> ⚠️ **`v` est une copie.** Modifier `v` dans `for _, v := range s` ne change **pas** le
> slice. Pour muter en place, indexez : `s[i] = …`.

> 🔁 Depuis Go 1.23, `range` accepte aussi une **fonction** comme source (itérateurs
> `iter.Seq`/`iter.Seq2`) — un mécanisme à part entière, détaillé au [Ch. 18 — Itérateurs
> par fonction](18-iterateurs.md).

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

- `break` sort de la boucle (ou du `switch`/`select`) **la plus interne**.
- `continue` passe à l'**itération suivante** de la boucle **la plus interne**.
- Une **étiquette** (un identifiant suivi de `:`, placé juste avant un `for`) permet de
  viser une boucle **externe** — indispensable pour sortir de boucles imbriquées en une
  fois, ce qu'un `break` nu ne sait pas faire :

```go
search:
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c] == target {
				break search // sort des DEUX boucles d'un coup
			}
		}
	}
```

`continue` accepte aussi une étiquette, avec un effet **différent** de `break` étiqueté :
il n'arrête **pas** la boucle externe, il la fait seulement passer à l'**itération
suivante**, en abandonnant le reste de la boucle interne en cours :

```go
rows:
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c] < 0 {
				continue rows // abandonne le reste de la ligne r, passe à r+1
			}
			fmt.Println(grid[r][c])
		}
	}
```

`goto Label` saute vers une étiquette **dans la même fonction** (en avant ou en arrière),
mais reste **rare** : presque tout ce qu'il permet s'exprime mieux avec une boucle, une
condition ou un `defer`. Deux règles limitent les sauts dangereux — on ne peut pas sauter
**dans** un bloc englobant depuis l'extérieur, ni sauter par-dessus une déclaration de
variable de façon à entrer dans sa portée :

```go
goto next
x := 5 // jamais exécutée, mais le compilateur refuse quand même de compiler :
       // "goto next jumps over declaration of x"
next:
	fmt.Println(x)
```

Cette restriction garantit qu'à l'endroit où le `goto` atterrit, toute variable en
portée a bien été initialisée — pas de saut vers une variable « à moitié déclarée ». Le
même exemple, sans `goto`, est d'ailleurs plus clair avec un simple `if` — ce qui résume
bien pourquoi `goto` reste si peu utilisé en Go idiomatique.

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

> ⚠️ Ce changement est piloté par la **version de langage déclarée dans `go.mod`**
> (directive `go 1.22` ou supérieure), pas seulement par la version de la toolchain
> installée. Un module qui déclare `go 1.20` conserve l'**ancien** comportement (variable
> partagée) même compilé avec une toolchain 1.26 — mettre à jour `go.mod`, pas seulement
> `go`, est donc nécessaire pour bénéficier du correctif.

> 🔁 Ce piège historique et sa disparition sont détaillés au [Ch. 15 — Closures](15-closures.md),
> et ses implications pour les goroutines au [Ch. 19](19-goroutines.md).

## ⚠️ Pièges

- **Valeur de `range` copiée** — `for _, v := range s { v.x = … }` ne modifie rien ;
  indexez `s[i]`. Vrai aussi pour une **map** : `for k, v := range m { v.x = … }` ne
  modifie pas la map.
- **`range` sur un array copie tout l'array une fois**, au départ — un array volumineux
  modifié pendant la boucle ne change pas les valeurs déjà vues par les itérations
  restantes (préférez un slice ou un `*[N]T`, voir plus haut).
- **Ordre d'itération d'une map aléatoire** — ne dépendez jamais de l'ordre (c'est
  volontaire, voir Ch. 7). Triez les clés si l'ordre compte.
- **`break` dans un `select`/`switch`** — il sort du `switch`/`select`, pas de la boucle
  englobante : utilisez une **étiquette** pour cette dernière.
- **`continue` étiqueté ne sort pas de la boucle externe** — il la fait seulement passer
  à l'itération suivante ; pour sortir complètement, il faut `break Label`.
- **Index octet vs rune** — sur une string, l'index de `range` est en **octets**, pas en
  caractères (voir Ch. 7).

## ⚡ Performance

- **Bornes éliminées par le compilateur (BCE)** — une boucle idiomatique
  `for i := 0; i < len(s); i++ { _ = s[i] }` permet au compilateur de prouver que `i`
  reste dans les bornes et de **supprimer le test de dépassement** à chaque accès `s[i]`.
  Indexer avec une variable non corrélée à `len(s)` réintroduit ce test. On observe ces
  décisions avec `go build -gcflags="-d=ssa/check_bce/debug=1"`.
- **`range` sur un array copie tout l'array une fois** (voir ⚠️ ci-dessus) — pour un grand
  array, ce coût est réel et se répète à chaque appel de la fonction qui contient la
  boucle. Ranger sur un **slice** (en-tête de 3 mots, indépendant de la taille des
  données) ou un `*[N]T` l'évite.
- **`for range n` (entier, 🆕 1.22) ne coûte rien de plus** qu'un `for i := 0; i < n; i++`
  écrit à la main : le compilateur génère le même code, c'est un sucre syntaxique pur.
- **Ne micro-optimisez pas à l'aveugle** — ces détails comptent dans une boucle exécutée
  des millions de fois (parsing, calcul numérique) ; ailleurs, la clarté du code prime.
  Mesurez avec `go test -bench` avant de réécrire une boucle pour la performance
  (Ch. 36/40).

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
4. Écrivez un petit programme avec un `goto` qui saute par-dessus une déclaration de
   variable, et lisez le message d'erreur exact du compilateur.
5. Reprenez l'exemple `arr := [3]int{1, 2, 3}` ci-dessus et remplacez `range arr` par
   `range &arr` : la mutation faite pendant la boucle devient-elle visible ?

---

## 📌 À retenir

- Un seul mot-clé de boucle : **`for`**, en trois formes (complète, condition, infinie) ;
  la clause `post` accepte une affectation multiple, pas seulement `i++`.
- `if`/`switch`/`for` acceptent une **instruction d'initialisation** à portée locale —
  attention au _shadowing_ d'une variable existante (`:=` crée toujours du neuf).
- `range` parcourt slices, maps, strings, channels et **entiers** (🆕 1.22), et depuis
  1.23 des fonctions (Ch. 18) ; sa source est **évaluée une seule fois**, et la valeur
  obtenue est une **copie**.
- `switch` **ne tombe pas** d'un cas à l'autre ; sans condition, il remplace `if/else if`.
- **Étiquettes** : `break Label` sort de la boucle visée, `continue Label` passe à son
  itération suivante — deux effets différents. `goto` existe (sauts restreints) mais
  reste rare.
- 🆕 1.22 : `for range N` et **portée par itération** (fin du piège de capture) — ce
  dernier point dépend de la version déclarée dans `go.mod`, pas seulement de la
  toolchain.

## 🔁 Pour aller plus loin

- [Ch. 5 — Fonctions](05-fonctions.md).
- [Ch. 14 — `switch` & sélection de cas](14-switch.md) : type switch, jump tables.
- [Ch. 15 — Closures](15-closures.md) : la capture de variables de boucle en détail.
- [Ch. 18 — Itérateurs par fonction](18-iterateurs.md) : `range` sur une fonction (1.23).
