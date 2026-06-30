# 12 — Packages, modules & organisation du code

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

Avant les modules (Go 1.11, généralisés en 1.13), tout le code source vivait sous un unique
`$GOPATH/src`, organisé en miroir du chemin d'import : un seul exemplaire de chaque dépendance
sur la machine, partagé par **tous** les projets — impossible de figer deux versions
différentes d'une même bibliothèque selon le projet en cours (`GOPATH`, réduit aujourd'hui au
rôle de cache, est détaillé au [Ch. 1](01-installation-toolchain.md)). `go.mod` règle ce
problème en déplaçant le versionnage **dans le module lui-même** : chaque projet fige ses
propres dépendances, indépendamment des autres présents sur la machine — d'où des builds
**reproductibles** et **isolés**.

Le fichier déclare le **chemin du module** (préfixe de tous ses packages), la **version de Go**
minimale, et les **dépendances** :

```
module github.com/alice/billing   // chemin d'import racine

go 1.26                            // version de langage minimale, vérifiée strictement

require (
	github.com/google/uuid v1.6.0  // dépendance directe : importée par votre code
	golang.org/x/text v0.20.0      // indirect : importée par une dépendance, pas par vous
)
```

- **`go mod init <chemin>`** crée le fichier ; **`go mod tidy`** ajoute les dépendances
  réellement importées, **retire** les inutilisées, et tient à jour l'étiquette `// indirect`
  (à lancer avant chaque commit).
- **`go.sum`** enregistre, pour chaque version retenue, deux **empreintes** SHA-256 (le contenu
  du module et son `go.mod`) : `go build`/`go mod verify` les recontrôlent et refusent de
  continuer en cas de divergence. On le **commit**, on ne l'édite **jamais** à la main.
- La ligne **`go 1.26`** n'est pas qu'indicative : depuis Go 1.21, elle est **vérifiée
  strictement**. Un module qui l'exige refuse de compiler avec une toolchain plus ancienne, sauf
  bascule automatique pilotée par `GOTOOLCHAIN` (🔁 [Ch. 1](01-installation-toolchain.md)).

## Versions : SemVer & sélection

Go utilise le **versionnage sémantique** (`vMAJEUR.MINEUR.CORRECTIF`) et la **sélection de
version minimale** (MVS) : la version retenue est la **plus petite** qui satisfait toutes les
exigences — résultat **reproductible**, sans « dernière en date » surprise.

Concrètement, chaque dépendance exige une version _minimale_ des siennes ; Go assemble toutes
ces exigences en un **graphe** puis retient, pour chaque module présent plusieurs fois, le
**maximum des minimums demandés** — qui reste la plus petite version satisfaisant tout le monde
**à la fois** (d'où « minimale ») :

```
   billing (votre module)
     --requiert--> A >= v1.2.0  --requiert--> C >= v1.0.0
     --requiert--> B >= v1.5.0  --requiert--> C >= v1.3.0

   build list retenue : A v1.2.0, B v1.5.0, C v1.3.0
   (pour C : max(v1.0.0, v1.3.0) = v1.3.0 -- la plus petite version qui
    satisfait À LA FOIS l'exigence de A et celle de B)
```

> 💡 Contrairement à des résolveurs comme celui de npm (qui retiennent souvent la **plus
> récente** version satisfaisant un intervalle), MVS ne fait **jamais** monter une version sans
> qu'une exigence explicite du graphe ne le demande. Pas de solveur à contraintes : un seul
> passage du graphe suffit, donc un résultat déterministe. Pour monter délibérément une version,
> il faut un `go get` explicite.

```bash
go get github.com/google/uuid@v1.6.0   # version précise
go get github.com/google/uuid@latest   # dernière version stable
go get -u ./...                        # met à jour les dépendances (mineures/correctifs)
```

> 💡 Un commit sans tag de version reçoit un **pseudo-version** déduit de l'historique Git :
> `v0.0.0-20240115120000-abcdef123456` (horodatage UTC + hash court). Go les ordonne
> chronologiquement comme de vraies versions ; vous les croiserez dans `go.sum` dès qu'une
> dépendance n'a jamais publié de tag.

> ⚠️ **Major version = chemin d'import.** À partir de `v2`, le suffixe fait **partie** du chemin :
> `github.com/foo/bar/v2`. Deux versions majeures peuvent ainsi coexister dans un même build.
> En dessous de `v2` (y compris **`v0`**, qui signale une API **instable**, sans garantie de
> compatibilité), le chemin d'import ne change jamais.

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

Ce n'est **pas une notion du langage** (pas de mot-clé) mais une règle de **chemin d'import**,
appliquée par l'outillage (`go build`, `go vet`, `golang.org/x/tools`) à la compilation : dès
qu'un segment du chemin s'appelle littéralement `internal`, seul le code dont le chemin **part
du parent** de ce segment a le droit de l'importer ; tout autre import échoue avec « use of
internal package … not allowed ». La règle est purement **positionnelle** et s'applique
indépendamment à chaque `internal/` : un module peut en contenir plusieurs, à des profondeurs
différentes — `billing/internal/store` est visible depuis tout `billing/...`, mais un autre
`billing/api/internal/cache` ne serait visible que depuis `billing/api/...`, pas depuis
`billing/cmd/...`.

> 💡 `internal/` est l'outil pour exposer une API **stable aux yeux de vos utilisateurs** tout en
> gardant **toute liberté** de réorganiser l'intérieur : personne d'extérieur ne peut s'y
> accrocher.

## Organiser les packages (pragmatique)

- **Nommez par responsabilité**, pas par couche technique : `store`, `billing`, `auth` — pas
  `models`, `utils`, `helpers`. Un package `util` finit en fourre-tout et crée des cycles.
- **`cmd/<nom>/`** pour chaque binaire (`package main`) ; la logique réutilisable vit dans des
  packages importables (souvent sous `internal/`). Raison concrète : un `package main` ne peut
  pas être **importé** par un test ou un autre programme — tout ce qui reste dedans n'est
  testable qu'en exécutant le binaire. Un `main()` réduit à l'assemblage (lire la config, câbler
  les dépendances, appeler la logique) garde le gros du code dans des packages `go test`-ables
  isolément.
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

```
go.work :
   go 1.26

   use (
   	./billing
   	./billing-lib
   )
```

Le `go.work` (non commité en général) prend le pas sur les `go.mod` : les imports croisés
pointent vers le code **local**, sans toucher au contenu d'aucun `go.mod`.

> 💡 **`go work` vs `replace`** — les deux redirigent un import vers du code local, mais
> `replace` vit **dans** `go.mod` : oublié au moment de publier, il casse le build de tous les
> autres utilisateurs (⚠️ ci-dessous). `go work` vit dans un fichier **séparé**, propre à votre
> poste, qui ne part jamais dans un commit ou une release : c'est l'option à préférer pour du
> développement multi-modules au quotidien.

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

Le **nom** n'est pas libre : `go doc` l'analyse pour rattacher l'exemple au bon symbole.

| Nom                        | Rattaché à                                                   |
| -------------------------- | ------------------------------------------------------------ |
| `Example`                  | le package entier (doc en tête de page)                      |
| `ExampleAmount`            | le type `Amount`                                             |
| `ExampleAmount_String`     | la méthode `String` du type `Amount`                         |
| `ExampleAmount_String_neg` | variante supplémentaire (suffixe libre après le dernier `_`) |

Un nom qui ne correspond à aucun identifiant exporté du package reste un test exécuté, mais
n'apparaît **rattaché à rien** dans la documentation générée.

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
  Signe avant-coureur fréquent : deux packages « métier » qui s'appellent mutuellement parce
  qu'aucun des deux n'est clairement en dessous de l'autre — c'est souvent qu'un troisième
  package (types/interfaces partagés) doit être extrait.
- **Package `utils`/`common`** : aimant à dépendances et à cycles. Nommez par domaine.
- **Ne pas commiter `go.sum`** → builds non vérifiables. Toujours le versionner.
- **Tout mettre sous `internal/` par réflexe** : pratique au départ, mais si une partie de ce
  code doit un jour devenir une bibliothèque publique (la vôtre, ou partagée avec un autre
  module du même dépôt), il faut **déplacer** les fichiers — donc changer leur chemin d'import
  partout où ils sont utilisés. `internal/` doit protéger des **détails d'implémentation**
  réellement instables, pas servir de position par défaut pour tout ce qui n'est pas `main`.
- **Package trop gros** (un seul `internal/core` qui concentre toute la logique) : il devient le
  nœud que **tout** importe, donc celui qui invalide le plus de cache à chaque modification
  (⚡ ci-dessous), et il est impossible à tester par sous-domaine (`go test ./core/...` retombe
  toujours sur l'ensemble). Un signe qu'il faut le scinder : des noms de fonctions préfixés par
  thème (`UserCreate`, `UserDelete`, `OrderCreate`…) qui annoncent des sous-packages.

## ⚡ Performance (de build)

- Go **met en cache** les packages compilés (`go env GOCACHE`) : seul le code modifié — et tout
  ce qui en **dépend en amont dans le graphe** — est recompilé. Un package importé par la moitié
  du projet invalide la moitié du cache à chaque changement ; découper en packages **cohérents
  et peu interdépendants** améliore l'incrémental bien plus qu'un découpage par couche.
- Un **graphe de dépendances** plus plat et sans cycles compile plus vite et se teste par morceaux
  (`go test ./store/...`).
- `go test` **met aussi en cache** les résultats : un test dont le code et les dépendances n'ont
  pas changé depuis la dernière exécution réussie n'est pas relancé (`go test -count=1` force la
  réexécution). Des packages plus petits et plus stables profitent davantage de ce cache en CI.

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
