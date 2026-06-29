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
2. Dans un package : les **variables de package** sont initialisées (dans l'ordre de
   leurs dépendances), **puis** les fonctions **`init()`** s'exécutent (dans l'ordre des
   fichiers, puis d'apparition).
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
   [main]      vars de package  -->  init()         (2) puis le package main
        |
        v
   [main]      main()                               (3) enfin le point d'entrée
```

Dans l'exemple du chapitre, la sortie commence par `[init main] version 1.0` **avant**
toute ligne de `main()` — preuve que les `init()` tournent avant `main`.

> 💡 Usage typique d'`init()` : enregistrer un driver (`database/sql`), préparer une
> table de correspondance, valider une configuration. À garder **court et sans effet de
> bord surprenant** — du code difficile à tester sinon.

## Commentaires de documentation & `go doc`

La documentation **est** dans le code, sous forme de commentaires placés **juste avant**
l'élément documenté. Convention : commencer la phrase par le **nom** de l'élément.

```go
// Greet renvoie une salutation pour name dans la langue lang.
func Greet(lang, name string) string { … }
```

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
        |  package main  | --------------------> |  package greeting        |
        |  (exécutable)  |                       |  (bibliothèque)          |
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
