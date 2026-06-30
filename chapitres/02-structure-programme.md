# 2 — Structure d'un programme

> **Objectif** — Comprendre l'**unité de compilation** de Go (le package), le **point
> d'entrée** d'un programme, et l'**ordre d'initialisation**.
>
> **Prérequis** — [Ch. 1 — Installation, toolchain & premier programme](01-installation-toolchain.md)

---

## Introduction

En Go, on n'organise pas le code en classes mais en **packages**. Un package regroupe
des fichiers `.go` qui partagent un même espace de noms et se compilent ensemble. Bien
comprendre les packages, c'est comprendre **comment Go structure, isole et initialise**
le code.

## Le package : l'unité de compilation

Règles de base :

- Chaque fichier `.go` commence par une **clause `package`**.
- Tous les fichiers d'un **même dossier** appartiennent au **même package**.
- Un package est l'**unité de compilation et de visibilité** : ce qui est défini dedans
  est accessible partout dans le package, sans `import`.

```go
package greeting   // tous les .go de ce dossier déclarent « package greeting »
```

> ⚠️ Le **nom du package** (clause `package`) est indépendant du **nom du dossier**, mais
> par convention on les fait coïncider. Le **chemin d'import**, lui, suit l'arborescence
> du module.

Une seule exception confirme la règle « un dossier = un package » : les fichiers
`_test.go`. Ils peuvent déclarer le **même nom de package** que le reste du dossier (test
« boîte blanche », avec accès aux identifiants non exportés — c'est le choix fait par
`greeting_test.go` dans l'exemple du chapitre) ou un nom suffixé de `_test` (test « boîte
noire », `package greeting_test`, qui ne voit que l'API exportée, comme le ferait un
appelant extérieur). Le second style force à tester l'API publique telle qu'un utilisateur
la verrait ; à privilégier dès que c'est possible (détail au [Ch. 13](13-tests-outillage.md)).

## `package main` vs bibliothèque

Deux familles de packages :

|                        | `package main`            | bibliothèque (tout autre nom)           |
| ---------------------- | ------------------------- | --------------------------------------- |
| Rôle                   | produit un **exécutable** | code **réutilisable**, importé ailleurs |
| Doit contenir          | une fonction **`main()`** | (pas de `main`)                         |
| Résultat de `go build` | un binaire                | rien à exécuter directement             |

```go
package main

import "fmt"

func main() { // point d'entrée OBLIGATOIRE d'un exécutable
	fmt.Println("démarrage")
}
```

`func main()` ne prend **aucun paramètre** et ne renvoie **rien** — contrairement au
`int main(int argc, char **argv)` du C. Les arguments de la ligne de commande s'obtiennent
via `os.Args` (un `[]string`, le nom du programme en `os.Args[0]`), et le **code de
sortie** se règle avec `os.Exit(n)` (`0` implicite si `main` se termine normalement, `2`
en cas de `panic` non récupérée).

> 💡 Un module peut contenir **plusieurs** `package main` : typiquement un par sous-dossier
> de `cmd/` (`cmd/server`, `cmd/client`…), chacun produisant son **propre binaire**, nommé
> par défaut d'après son dossier (`go build ./cmd/server` produit l'exécutable `server`),
> sauf à imposer un nom avec `-o` (🔁 Ch. 12 pour l'organisation `cmd/`/`internal/`).

## `import` : chemins, regroupement, alias

On rend un package disponible avec `import`, en désignant son **chemin** (pas son nom) :

```go
import (
	"fmt"             // package de la bibliothèque standard
	"strings"         // idem

	"example.com/gobook/ch02-structure/greeting" // package local au module
)
```

- Le chemin d'un package **standard** est court (`fmt`, `net/http`).
- Le chemin d'un package **de votre module** part du chemin déclaré dans `go.mod`
  (`example.com/gobook`) suivi de l'arborescence des dossiers.
- On **regroupe** les imports dans un bloc `import ( … )` ; `gofmt`/`goimports` les
  trie et les sépare (standard / externes).

### Formes particulières d'import

```go
import (
	crand "crypto/rand"  // ALIAS : on renomme pour éviter une collision avec math/rand
	_ "image/png"        // IMPORT BLANK : pour ses seuls effets de bord (son init())
	. "fmt"              // DOT IMPORT : expose les symboles sans préfixe — À ÉVITER
)
```

| Forme   | Syntaxe                | Usage                                                         |
| ------- | ---------------------- | ------------------------------------------------------------- |
| Normale | `import "fmt"`         | accès via `fmt.Println`                                       |
| Alias   | `import f "fmt"`       | accès via `f.Println` (résoudre une collision)                |
| Blank   | `import _ "image/png"` | n'importe que pour exécuter l'`init()` du package             |
| Dot     | `import . "fmt"`       | `Println` sans préfixe — ⚠️ nuit à la lisibilité, à proscrire |

L'import blank ci-dessus n'est pas qu'un exemple d'école : `image/png` enregistre son
décodeur auprès du registre global du package `image` depuis son `init()` (section
suivante). Sans cet import, `image.Decode` ne reconnaîtrait pas les fichiers PNG — alors
même qu'aucune fonction de `png` n'est appelée explicitement. Le même mécanisme sert à
enregistrer un driver `database/sql` : `import _ "github.com/lib/pq"` rend le driver
PostgreSQL disponible sans exposer le moindre symbole.

> ⚠️ Un import **inutilisé** est une **erreur de compilation** (pas un simple avertissement).
> C'est volontaire : le code reste propre. `goimports` retire automatiquement les imports
> superflus.

## Identifiants exportés : la règle de la majuscule

La visibilité **hors du package** ne dépend ni d'un mot-clé `public`/`private`, ni de
fichiers d'en-tête. Elle tient à **une seule règle** :

> Un identifiant est **exporté** (visible depuis un autre package) **si et seulement si
> son nom commence par une majuscule**.

```go
func Greet(lang, name string) string { … } // EXPORTÉ : appelable via greeting.Greet
func hello(lang string) string        { … } // privé : visible seulement dans le package

var DefaultLang = "fr" // exportée
var defaultLang = "fr" // privée
```

Cela vaut pour **tout** : fonctions, types, variables, constantes, champs de struct,
méthodes. C'est simple, mécanique, et immédiatement visible à la lecture.

Précision : la règle porte sur la catégorie Unicode **« lettre majuscule » (Lu)**, pas
seulement sur `A`-`Z`. Un identifiant commençant par `É` ou `Ω` est donc exporté ; un
identifiant commençant par un chiffre, un `_`, ou une lettre sans casse (la plupart des
idéogrammes) ne l'est jamais, quelle que soit l'intention du développeur.

> 💡 Cette règle a des conséquences au-delà de la simple visibilité : `encoding/json` (et la
> plupart des bibliothèques de (dé)sérialisation) **ignorent silencieusement les champs non
> exportés** d'une struct. Un champ `total` minuscule oublié dans un `json.Marshal` ne
> provoque ni erreur ni avertissement : il disparaît simplement du JSON produit.

## `main`, `init` et l'ordre d'initialisation

Deux fonctions ont un rôle spécial :

- **`main()`** — le point d'entrée, uniquement dans `package main`. Sa fin termine le
  programme.
- **`init()`** — fonction **sans argument ni retour**, exécutée **automatiquement** au
  chargement du package. Un package (voire un fichier) peut en avoir **plusieurs**. On
  ne l'appelle jamais à la main.

L'ordre d'initialisation est **déterministe** :

1. Les **packages importés** sont initialisés **en premier**, récursivement (un package
   est prêt avant celui qui l'importe).
2. Dans un package : les **variables de package** sont initialisées dans l'ordre de
   **leurs dépendances** — pas l'ordre du texte : une variable qui en référence une autre
   est initialisée après elle, même si elle est déclarée avant dans le fichier. **Puis**
   les fonctions **`init()`** s'exécutent, dans l'**ordre lexicographique des noms de
   fichiers** du package, et dans l'ordre d'apparition au sein d'un même fichier.
3. Enfin, pour un exécutable, **`main()`** démarre.

```
   Ordre d'initialisation d'un exécutable
   --------------------------------------

   import greeting
        |
        v
   [greeting]  vars de package  -->  init()        (1) package importé d'abord
        |
        v
     [main]    vars de package  -->  init()        (2) puis le package main
        |
        v
     [main]    main()                              (3) enfin le point d'entrée
```

Dans l'exemple du chapitre, la sortie commence par `[init main] version 1.0` **avant**
toute ligne de `main()` — preuve que les `init()` tournent avant `main`.

> 💡 Usage typique d'`init()` : enregistrer un driver (`database/sql`), préparer une
> table de correspondance, valider une configuration. À garder **court et sans effet de
> bord surprenant** — du code difficile à tester sinon.

> ⚠️ Contrairement à toute autre fonction, `init` peut être **déclarée plusieurs fois**
> dans un même fichier, voire un même package (le compilateur les exécute toutes, dans
> l'ordre où elles apparaissent) — c'est la seule exception du langage à l'interdiction de
> redéclarer un identifiant. N'en abusez pas pour autant : un seul `init()` par fichier,
> centré sur une responsabilité, reste plus lisible et plus simple à déboguer.

> ⚠️ `os.Exit(code)` termine le programme **immédiatement**, sans exécuter les `defer` en
> attente (🔁 Ch. 16). Réservez-le à la toute fin de `main` ; appelé en profondeur dans la
> pile, il peut laisser des ressources ouvertes (fichiers, verrous, connexions).

## Commentaires de documentation & `go doc`

La documentation **est** dans le code, sous forme de commentaires placés **juste avant**
l'élément documenté. Convention : commencer la phrase par le **nom** de l'élément.

```go
// Greet renvoie une salutation pour name dans la langue lang.
func Greet(lang, name string) string { … }
```

Cette convention n'est pas qu'une question de style : `go doc` et `pkg.go.dev` extraient la
**première phrase** du commentaire comme résumé synthétique (affiché dans les listes et les
résultats de recherche), et `staticcheck` (règle `ST1020`) signale un commentaire de
fonction exportée qui ne commence pas par son nom. La respecter rend la documentation
générée immédiatement exploitable, sans effort supplémentaire.

Le commentaire d'un **package** commence par `Package <nom>` et se place avant la clause
`package` (souvent dans un fichier `doc.go`).

Consultez la doc en ligne de commande :

```bash
go doc fmt                    # synthèse du package fmt
go doc fmt.Println            # un symbole précis
go doc ./ch02-structure/greeting   # un package local
```

> 🆕 **Go 1.25** : `go doc -http` lance un **serveur de doc local** pour parcourir
> l'ensemble dans le navigateur, hors-ligne.

## Schéma : graphe de dépendances

```
        +----------------+        importe        +-------------------------+
        |  package main  | --------------------> |  package greeting       |
        |  (exécutable)  |                       |  (bibliothèque)         |
        +----------------+                       +-------------------------+
              |  importe                                |  importe
              v                                         v
        +----------------+                       +----------------+
        |  fmt (stdlib)  |                       |  fmt (stdlib)  |
        +----------------+                       +----------------+

   Règle : les dépendances forment un graphe ACYCLIQUE.
   Les imports circulaires sont INTERDITS (erreur de compilation).
```

> ⚠️ Go **interdit les imports circulaires** (A importe B qui importe A). C'est une
> contrainte de conception : elle force à clarifier les responsabilités, parfois via un
> troisième package commun.

En pratique, deux techniques pour casser un cycle : **extraire un package de plus bas
niveau** partagé par les deux côtés (types et constantes communs, sans dépendance vers
eux), ou **inverser la dépendance via une interface** — le package de haut niveau définit
une interface décrivant ce dont il a besoin, et c'est le package de bas niveau qui
l'implémente sans le connaître (🔁 Ch. 9 — Interfaces). Un cycle d'imports est souvent le
signe qu'un type ou une fonction est mal placé : un signal d'architecture, pas seulement un
obstacle technique à contourner.

## ⚠️ Pièges

- **Dossier ≠ package, sauf une exception** — les fichiers `_test.go` peuvent déclarer
  `package nomDuPaquet` (boîte blanche, accès aux non exportés) ou `package
nomDuPaquet_test` (boîte noire, API publique seulement). C'est la seule entorse à
  « un dossier = un package ».
- **Dot import** (`. "fmt"`) économise un préfixe mais rend le code illisible dès que le
  fichier grossit : on ne sait plus d'où vient un symbole. À réserver à de rares DSL de
  test.
- **Import inutilisé ou circulaire** : erreur de **compilation**, jamais un simple
  avertissement — rien ne compile tant que ce n'est pas corrigé.
- **Plusieurs `init()` dans un même fichier** sont autorisés (seule exception du langage à
  l'interdiction de redéclarer un identifiant) : à éviter en code de production, où un seul
  `init()` lisible vaut mieux.
- **`os.Exit()` ignore les `defer`** : un appel en profondeur d'appel peut laisser des
  ressources non libérées (fichiers, verrous) — à réserver à la toute fin de `main`.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch02-structure     # observez l'ordre : [init main] avant les saluts
go test ./ch02-structure/...
go doc ./ch02-structure/greeting Greet
```

À essayer :

1. Ajoutez une langue (`"de": "Hallo"`) dans `greeting` et un appel correspondant.
2. Renommez `Greet` en `greet` (minuscule) : observez l'**erreur de compilation** dans
   `main` (symbole non exporté).
3. Ajoutez un second `init()` dans `greeting` qui affiche un message : vérifiez qu'il
   tourne avant `main`.

---

## 📌 À retenir

- Le **package** est l'unité de compilation ; tous les fichiers d'un dossier le partagent.
- **`package main` + `func main()`** = exécutable ; tout autre nom = bibliothèque.
- **Majuscule = exporté**, minuscule = privé au package. Une seule règle, partout.
- `init()` s'exécute **automatiquement** ; ordre : packages importés → vars → `init` → `main`.
- Imports **inutilisés interdits**, imports **circulaires interdits**.
- La **doc** vit dans les commentaires (préfixés du nom) ; lisible via `go doc`.

## 🔁 Pour aller plus loin

- [Ch. 3 — Variables, constantes & types de base](03-variables-constantes-types.md).
- [Ch. 12 — Packages, modules & organisation du code](12-packages-modules.md) : `go.mod`, `internal/`, workspaces.
- Annexe F — Idiomes & style (nommage des packages).
