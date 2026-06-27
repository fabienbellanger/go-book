# Ch. 12 — Packages, modules & organisation du code

> **Objectif** — Structurer un projet Go : du **package** (unité de compilation et de
> visibilité) au **module** (unité de version et de distribution), avec `go.mod`, `internal/`,
> les workspaces et la documentation.
>
> **Prérequis** — [Ch. 2 — Structure d'un programme](02-structure-programme.md), [Ch. 1 — Toolchain](01-installation-toolchain.md)

---

## Introduction

Deux niveaux d'organisation, à ne pas confondre :

- un **package** est l'unité de **compilation** et de **visibilité** — un répertoire, un nom, des
  identifiants exportés (majuscule) ou non ([Ch. 2](02-structure-programme.md)) ;
- un **module** est l'unité de **version** et de **distribution** — un arbre de packages décrit
  par un `go.mod`, qu'on publie et que d'autres importent.

Un module contient un ou plusieurs packages ; un programme combine des packages issus de
plusieurs modules. L'exemple est dans [`code/ch12-packages/`](../code/ch12-packages/).

---

## Le fichier `go.mod`

`go.mod` déclare le **chemin du module** (préfixe de tous ses packages), la **version de Go**
minimale, et les **dépendances** :

```
module github.com/alice/billing   // chemin d'import racine

go 1.26                            // version de langage minimale

require (
	github.com/google/uuid v1.6.0  // dépendance directe
	golang.org/x/text v0.20.0      // (in)directe
)
```

- **`go mod init <chemin>`** crée le fichier ; **`go mod tidy`** ajoute les dépendances
  réellement importées et **retire** les inutilisées (à lancer avant chaque commit).
- **`go.sum`** enregistre les **empreintes** cryptographiques des versions téléchargées
  (intégrité). On le **commit**, on ne l'édite pas à la main.

## Versions : SemVer & sélection

Go utilise le **versionnage sémantique** (`vMAJEUR.MINEUR.CORRECTIF`) et la **sélection de
version minimale** (MVS) : la version retenue est la **plus petite** qui satisfait toutes les
exigences — résultat **reproductible**, sans « dernière en date » surprise.

```bash
go get github.com/google/uuid@v1.6.0   # version précise
go get github.com/google/uuid@latest   # dernière version stable
go get -u ./...                        # met à jour les dépendances (mineures/correctifs)
```

> ⚠️ **Major version = chemin d'import.** À partir de `v2`, le suffixe fait **partie** du chemin :
> `github.com/foo/bar/v2`. Deux versions majeures peuvent ainsi coexister dans un même build.

## `internal/` : la visibilité à l'échelle du projet

Un package sous un répertoire **`internal/`** n'est importable **que** par le code situé sous le
**parent** de cet `internal/`. C'est le seul mécanisme de visibilité **au-delà** du couple
exporté/non-exporté.

```
   github.com/alice/billing/
   +-- cmd/billing/        main.go        -> peut importer internal/...  ✅
   +-- internal/
   |   +-- store/          store.go
   |   +-- money/          money.go       (API "publique" du projet, mais privée au module)
   +-- api/                api.go         -> peut importer internal/...  ✅

   github.com/bob/autre-projet/  ...      -> NE PEUT PAS importer billing/internal/... ❌
```

> 💡 `internal/` est l'outil pour exposer une API **stable aux yeux de vos utilisateurs** tout en
> gardant **toute liberté** de réorganiser l'intérieur : personne d'extérieur ne peut s'y
> accrocher.

## Organiser les packages (pragmatique)

- **Nommez par responsabilité**, pas par couche technique : `store`, `billing`, `auth` — pas
  `models`, `utils`, `helpers`. Un package `util` finit en fourre-tout et crée des cycles.
- **`cmd/<nom>/`** pour chaque binaire (`package main`) ; la logique réutilisable vit dans des
  packages importables (souvent sous `internal/`).
- **Plat tant que possible.** N'introduisez une hiérarchie que lorsqu'elle clarifie. Un petit
  projet peut tenir en un seul package.
- Le **nom du package** doit être court et sans redondance : `package money`, appelé
  `money.Amount` — jamais `money.MoneyAmount`.

> ⚠️ **Cycles d'imports interdits.** Si `a` importe `b` et `b` importe `a`, ça ne compile pas.
> C'est souvent le signe qu'une **troisième** abstraction (ou une interface, [Ch. 9](09-interfaces.md))
> doit être extraite.

## Workspaces : `go work` (multi-modules en local)

Pour développer **plusieurs modules ensemble** (ex. une lib et l'appli qui la consomme) sans
publier ni bricoler `replace`, un **workspace** les relie localement :

```bash
go work init ./billing ./billing-lib   # crée go.work
go work use ./nouveau-module           # ajoute un module
```

Le `go.work` (non commité en général) prend le pas sur les `go.mod` : les imports croisés
pointent vers le code **local**.

## Dépendances d'outils dans `go.mod` (🆕 1.24)

Avant 1.24, épingler la version d'un outil (linter, générateur) imposait un fichier `tools.go`
avec des imports blancs. Désormais, une directive **`tool`** s'en charge :

```bash
go get -tool golang.org/x/tools/cmd/stringer   # ajoute `tool ...` dans go.mod
go tool stringer -help                          # exécute l'outil épinglé
go tool                                         # liste les outils du module
```

```
go.mod :
   tool golang.org/x/tools/cmd/stringer
```

L'outil est **versionné** comme une dépendance ordinaire : toute l'équipe utilise la **même**
version.

## Documenter : doc comments & `Example`

La documentation Go **est** le commentaire qui **précède** une déclaration (sans ligne vide) ; il
**commence par le nom** de l'élément. `go doc` (et `pkg.go.dev`) l'affichent.

```go
// Amount représente un montant en centimes d'euro. Travailler en entiers évite
// les erreurs d'arrondi des float64.
type Amount int64
```

Un **`Example`** est une fonction de test qui sert **à la fois** de doc et de test exécuté
(détail [Ch. 13](13-tests-outillage.md)) :

```go
func ExampleAmount_String() {
	fmt.Println(money.Euros(12, 50))
	// Output: 12,50 €
}
```

Le commentaire `// Output:` est **vérifié** par `go test` : si la sortie change, le test échoue.
La doc et le code restent ainsi toujours d'accord.

---

## 🆕 Go 1.2x

- **1.24** — **dépendances d'outils** dans `go.mod` (directive `tool`, `go get -tool`, `go tool`),
  remplaçant le motif `tools.go`.
- **1.25** — directive **`ignore`** dans `go.mod` (répertoires ignorés par les motifs `./...`,
  utile pour des assets ou du généré non compilable) ; **`go doc -http`** lance un serveur de
  documentation local.

```
go.mod :
   ignore ./web/dist    // assets buildés, ignorés par go build ./...
```

## ⚠️ Pièges

- **`replace` oublié** dans `go.mod` au moment de publier : pointe vers un chemin local, casse le
  build des autres. À réserver au dev (ou utilisez `go work`).
- **Suffixe `/v2` manquant** après un changement de version majeure : les imports ne se résolvent
  pas.
- **Cycle d'imports** : refactorez vers une interface ou un package commun de plus bas niveau.
- **Package `utils`/`common`** : aimant à dépendances et à cycles. Nommez par domaine.
- **Ne pas commiter `go.sum`** → builds non vérifiables. Toujours le versionner.

## ⚡ Performance (de build)

- Go **met en cache** les packages compilés (`go env GOCACHE`) : seul le code modifié est
  recompilé. Découper en packages **cohérents** améliore l'incrémental.
- Un **graphe de dépendances** plus plat et sans cycles compile plus vite et se teste par morceaux
  (`go test ./store/...`).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch12-packages
go test ./ch12-packages/...
go doc ./ch12-packages/internal/money    # lit les doc comments
```

À essayer :

1. Tentez d'importer `ch12-packages/internal/money` depuis un autre chapitre (ex. `ch11-generics`)
   et observez l'erreur « use of internal package not allowed ».
2. Cassez la sortie de `ExampleAmount_String` (ex. `12.50`) et constatez l'échec de `go test`.
3. Lancez `go mod tidy` dans `code/` : rien ne change (aucune dépendance externe), mais c'est le
   réflexe avant commit.

---

## 📌 À retenir

- **Package** = unité de compilation/visibilité ; **module** (`go.mod`) = unité de
  version/distribution.
- `go mod tidy` synchronise les dépendances ; `go.sum` (commité) garantit l'intégrité ; MVS rend
  les builds reproductibles.
- **`internal/`** restreint l'import au sous-arbre parent : API publique côté projet, privée côté
  monde extérieur.
- Nommez les packages **par responsabilité**, gardez le graphe **sans cycles**, utilisez **`go work`**
  pour le multi-module local.
- 🆕 1.24 : outils épinglés via la directive **`tool`**. La doc, ce sont les **commentaires** ;
  les **`Example`** la rendent exécutable.

## 🔁 Pour aller plus loin

- [Ch. 13 — Tests & outillage](13-tests-outillage.md) : `Example`, couverture, `go vet`.
- [Ch. 1 — Toolchain](01-installation-toolchain.md) : `go build/run/test/env`, `GOTOOLCHAIN`.
- Projet 4 — Bibliothèque générique : SemVer, compatibilité, doc + `Example`, CI.
- Annexe B — Antisèche des commandes `go`.
