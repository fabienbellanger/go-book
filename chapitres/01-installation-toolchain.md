# 1 — Installation, toolchain & premier programme

> **Objectif** — Avoir un environnement Go fonctionnel, comprendre la toolchain, et
> exécuter son premier programme.
>
> **Prérequis** — [Ch. 0 — Pourquoi Go ?](00-pourquoi-go.md)

---

## Introduction

Go se distingue par une **toolchain unique et intégrée** : une seule commande, `go`,
fait tout — compiler, exécuter, tester, formater, documenter, analyser. Pas de
`Makefile` obligatoire, pas de gestionnaire de paquets tiers. Là où un projet
JavaScript assemble typiquement `npm` + `eslint` + `prettier` + `jest`, et un projet
Python `pip`/`poetry` + `black` + `mypy` + `pytest`, Go regroupe ces rôles dans **un
seul binaire** : une seule syntaxe de sous-commande, une seule version à synchroniser
entre développeurs. Ce chapitre installe cet environnement et exécute un premier
programme.

## Installation

Téléchargez l'installeur depuis **[go.dev/dl](https://go.dev/dl/)** (Windows, macOS,
Linux), ou utilisez un gestionnaire de paquets :

```bash
# macOS (Homebrew)
brew install go

# Linux (archive officielle, recommandé)
# extraire dans /usr/local puis ajouter /usr/local/go/bin au PATH
```

Vérifiez l'installation :

```bash
go version
# go version go1.26.4 darwin/arm64
```

La sortie indique la **version** et la cible **`OS/architecture`** courante.

## Variables d'environnement

Go fonctionne « out of the box », mais quelques variables méritent d'être connues.
Affichez-les avec `go env` :

```bash
go env GOROOT GOPATH GOBIN GOTOOLCHAIN
# /usr/local/go
# /Users/vous/go
#                      <- GOBIN vide : c'est normal, voir ci-dessous
# auto
```

| Variable          | Rôle                                                                                                                                                                                                                                               |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GOROOT`          | où Go est **installé** (réglé automatiquement, ne pas y toucher en général).                                                                                                                                                                       |
| `GOPATH`          | dossier des binaires et du cache (`~/go` par défaut). _Héritage_ de l'ère pré-modules : on n'y met plus son code source.                                                                                                                           |
| `GOBIN`           | où `go install` dépose les exécutables (`$GOPATH/bin` par défaut). **Vide** tant qu'on ne l'a pas réglé explicitement (`go env GOBIN` n'affiche alors rien) : c'est normal, le repli sur `$GOPATH/bin` s'applique quand même. À ajouter au `PATH`. |
| `GOTOOLCHAIN`     | gère le **téléchargement automatique** de la bonne version de Go selon `go.mod` (voir ci-dessous).                                                                                                                                                 |
| `GOOS` / `GOARCH` | système et architecture **cibles** de la compilation (cross-compilation).                                                                                                                                                                          |

> 💡 Depuis les **modules** (Go 1.11+), `GOPATH` n'est plus l'endroit où l'on range son
> code. Vos projets vivent **où vous voulez**, du moment qu'ils contiennent un `go.mod`.
> Avant 1.11, **tout** le code — y compris les dépendances tierces — devait vivre sous
> `$GOPATH/src`, organisé en miroir de son chemin d'import : un seul `GOPATH` actif ne
> permettait pas de faire cohabiter deux projets exigeant deux versions différentes
> d'une même dépendance. Les modules règlent ce problème en épinglant chaque dépendance
> à une version précise dans `go.mod`, indépendamment de l'emplacement du code sur disque.

### Gestion automatique de la toolchain

Depuis Go 1.21, `go.mod` distingue deux directives :

- **`go 1.26`** — la version **minimale** requise ; elle fixe aussi les fonctionnalités
  du langage disponibles (un module en `go 1.20` ne peut pas utiliser `range` sur un
  entier ni les built-ins `min`/`max`, apparus en 1.21-1.22).
- **`toolchain go1.26.4`** — la version **précise préférée** pour construire ce module ;
  elle doit être supérieure ou égale à la ligne `go`. Absente, elle vaut implicitement
  `go<version de la ligne go>`.

Si le `go` local est trop ancien pour l'une de ces deux exigences, la commande
**télécharge et lance automatiquement** la bonne version, sans rien installer de façon
permanente : le binaire téléchargé est un module comme un autre
(`golang.org/toolchain`), récupéré via `GOPROXY` et mis en cache dans `GOMODCACHE`
(`~/go/pkg/mod` par défaut). Ce mécanisme est piloté par `GOTOOLCHAIN` (`auto` par
défaut, raccourci de `local+auto`) et reste **strictement ascendant** : il bascule vers
une version plus récente si besoin, mais ne « downgrade » jamais silencieusement une
toolchain locale déjà plus récente que celle exigée. Conséquence : plus de « ça compile
chez moi mais pas chez toi » dû à une version différente.

```
   go local = 1.24              go.mod exige go 1.26 (ligne go ou toolchain)
        |                                |
        +------------- comparaison ------+
                        |
              1.24 < 1.26 : version locale insuffisante
                        |
                        v
       téléchargement de golang.org/toolchain@go1.26.x
       (via GOPROXY, vérifié par GOSUMDB, mis en cache dans GOMODCACHE)
                        |
                        v
              cette toolchain 1.26.x exécute la commande go
```

> ⚠️ Ce téléchargement suppose un accès réseau au proxy de modules. Derrière un proxy
> d'entreprise qui ne mirrore pas `golang.org/toolchain`, ou sur une machine sans accès
> Internet, la bascule échoue. Solutions : épingler `GOTOOLCHAIN=local` (jamais de
> bascule, on garde la toolchain installée telle quelle) ou `GOTOOLCHAIN=go1.26.4+path`
> (cherche uniquement dans le `PATH`, sans télécharger) — utile en CI ou sur un poste
> isolé où la version exacte est déjà pré-installée.

## Les commandes cœur

Tout passe par le sous-commandes de `go` :

| Commande     | Rôle                                                                  |
| ------------ | --------------------------------------------------------------------- |
| `go run`     | compile **et exécute** (idéal pour itérer, ne laisse pas de binaire). |
| `go build`   | compile en **binaire** (sans l'exécuter).                             |
| `go test`    | lance les **tests** et benchmarks.                                    |
| `go vet`     | **analyse statique** : détecte les erreurs probables.                 |
| `go fmt`     | **formate** le code selon le style canonique (`gofmt`).               |
| `go doc`     | affiche la **documentation** d'un package ou symbole.                 |
| `go fix`     | **modernise** le code (réécritures automatiques).                     |
| `go env`     | affiche/règle la **configuration**.                                   |
| `go mod`     | gère le **module** et ses dépendances (voir Ch. 12).                  |
| `go install` | compile et **installe** un exécutable dans `GOBIN`.                   |

## Premier programme

Créons le traditionnel « Hello ». Dans ce livre, les exemples vivent dans le module
`code/`. Le programme du chapitre est dans
[`code/ch01-hello/`](../code/ch01-hello/) :

```go
// code/ch01-hello/main.go
package main

import "fmt"

// greet construit le message de bienvenue pour name.
// On isole la logique dans une fonction pour pouvoir la tester (voir main_test.go).
func greet(name string) string {
	return fmt.Sprintf("Bonjour, %s ! 👋", name) // commentaire en français
}

func main() {
	fmt.Println(greet("Go")) // affiche : Bonjour, Go ! 👋
}
```

Exécutez-le :

```bash
cd code
go run ./ch01-hello
# Bonjour, Go ! 👋
```

Décortiquons chaque élément :

- **`package main`** — un programme exécutable appartient au package spécial `main`
  (les bibliothèques ont un autre nom, voir [Ch. 2](02-structure-programme.md)).
- **`import "fmt"`** — on importe le package `fmt` (formatage des entrées/sorties),
  issu de la **bibliothèque standard**.
- **`func greet(name string) string`** — une fonction **séparée** de `main`, qui prend
  une chaîne en paramètre et en renvoie une (signatures détaillées au
  [Ch. 3](03-variables-constantes-types.md) et au [Ch. 5](05-fonctions.md)). Elle ne
  fait **aucune** entrée/sortie : c'est ce qui la rend testable sans capturer la sortie
  standard — voir `main_test.go` et le 🧪 plus bas.
- **`fmt.Sprintf("...%s...", name)`** — construit une chaîne formatée et la **renvoie**,
  contrairement à `fmt.Printf` qui l'affiche directement. Le verbe `%s` insère `name`
  tel quel ; 🔁 [Annexe I — Verbes de formatage `fmt`](../annexes/I-formatage-fmt.md)
  pour le catalogue complet (`%d`, `%v`, `%q`…).
- **`func main()`** — le **point d'entrée** : l'exécution démarre ici. Ce rôle spécial
  n'existe que dans `package main` ; une fonction `main()` ailleurs serait une fonction
  ordinaire, sans signification particulière pour le compilateur.
- **`fmt.Println`** — affiche le résultat de `greet` sur la sortie standard, suivi d'un
  retour à la ligne.

> ⚠️ Les **identifiants** (variables, fonctions, types) sont en **anglais** dans tous les
> exemples du livre, mais les **commentaires** sont en **français**. C'est une convention
> du livre, et une bonne pratique pour du code partagé.

## Anatomie d'un binaire & cross-compilation

`go build` produit un **exécutable autonome** : il embarque le runtime Go (ordonnanceur,
ramasse-miettes, tables de types pour la réflexion) et toutes les dépendances, en
**liaison statique** par défaut. Aucun interpréteur ni bibliothèque partagée à installer
sur la machine cible — on copie un seul fichier.

```bash
cd code
go build -o hello ./ch01-hello
./hello            # Bonjour, Go ! 👋
ls -lh hello       # 1,5 à 2,5 Mo selon la plateforme : runtime Go inclus
```

> 💡 Même un programme aussi minimal pèse plusieurs mégaoctets : le runtime est
> **inclus dans chaque binaire**, quelle que soit la taille du code applicatif. Ce coût
> fixe ne grandit ensuite que lentement avec le code — un programme dix fois plus gros
> ne pèse pas dix fois plus lourd.

La **cross-compilation** est triviale : il suffit de fixer `GOOS` et `GOARCH`.
Depuis un Mac, on peut produire un binaire Linux/arm64 sans rien installer :

```bash
GOOS=linux GOARCH=arm64 go build -o hello-linux-arm64 ./ch01-hello
```

```
   Machine de build (macOS/arm64)            Cible (Linux/arm64)
   +---------------------------+             +-------------------+
   |  go build                 |  GOOS=linux |  hello-linux-     |
   |  GOOS=linux GOARCH=arm64  | ----------> |  arm64 (autonome) |
   +---------------------------+  GOARCH=... +-------------------+
        un seul outil, aucune toolchain croisée à installer
```

Listez toutes les cibles possibles avec `go tool dist list` (plusieurs dizaines de
combinaisons : Linux, macOS, Windows, BSD, WebAssembly...).

> ⚠️ Cette simplicité tient tant qu'**aucune dépendance n'utilise cgo** (l'interfaçage
> avec du code C). Dès que `GOOS`/`GOARCH` diffère de la machine hôte, Go désactive
> automatiquement cgo (`CGO_ENABLED=0`) sauf à configurer un compilateur C croisé
> (`CC`) : le binaire compile quand même, mais bascule silencieusement sur
> l'implémentation pure Go quand elle existe (ex. résolution de noms réseau), ou échoue
> si un package exige strictement cgo (ex. certains drivers SQLite). 🔁 Annexe B détaille
> `CGO_ENABLED=0`, qui force un binaire 100 % statique même en compilation native.

## Formatage & éditeurs

- **`gofmt`** (via `go fmt`) impose **un seul style** : indentation par tabulations,
  accolades placées automatiquement. Contrairement à `prettier` ou `black`, `gofmt` n'a
  **aucune option de style** (pas de largeur de ligne, pas d'emplacement d'accolade
  configurable) : il n'y a littéralement rien à régler, donc rien à débattre en revue
  de code.
- **`goimports`** (outil tiers) fait comme `gofmt` et gère **les imports** (ajoute ceux
  qui manquent, retire les inutiles).
- **`gopls`** est le **serveur de langage** officiel (Language Server Protocol) : il
  apporte autocomplétion, navigation, refactoring et diagnostics dans VS Code, GoLand,
  Neovim, etc. Installez l'extension Go de votre éditeur, elle s'occupe du reste.

> 💡 Configurez votre éditeur pour **formater à la sauvegarde**. Le code Go est, par
> convention, toujours passé à `gofmt` — un diff non formaté est immédiatement repéré.

## 🆕 Nouveautés récentes de la toolchain

- **🆕 Go 1.25** : directive **`ignore`** dans `go.mod` (exclut des dossiers du module) ;
  **`go doc -http`** lance un serveur local pour parcourir la doc dans le navigateur.
- **🆕 Go 1.26** : **`go fix`** intègre des _modernizers_ — il réécrit automatiquement le
  code vers des formes plus modernes (ex. boucles `for range N`, usage de `min`/`max`,
  `any` au lieu de `interface{}`). Lancez-le avec `go fix ./...`.

## 🧪 À tester soi-même

Le programme est accompagné d'un test _table-driven_
(`code/ch01-hello/main_test.go`, motif détaillé au [Ch. 13](13-tests-outillage.md)) qui
vérifie `greet` sur plusieurs entrées, dont des cas limites (chaîne vide, accents).
Lancez :

```bash
cd code
go test ./ch01-hello/...   # ok   example.com/gobook/ch01-hello
go vet ./ch01-hello/...    # (aucune sortie = tout va bien)
```

> ⚡ Relancez `go test ./ch01-hello/...` sans rien changer : la sortie devient
> `ok ... (cached)`. Le résultat n'est pas rejoué, juste retrouvé dans le **cache de
> build** (`GOCACHE`), indexé par un hash des sources et des options. `go build`
> partage ce même cache — c'est lui qui rend l'aller-retour « modifier → tester » quasi
> instantané, même sur de gros modules.

Puis expérimentez :

1. Modifiez `greet` pour saluer **votre** prénom et relancez `go run ./ch01-hello`.
2. Produisez un binaire pour Windows : `GOOS=windows GOARCH=amd64 go build ./ch01-hello`.
3. Cassez volontairement le format (enlevez une indentation) puis lancez `go fmt ./...`.
4. Cassez volontairement un test (changez un `want`) et relancez `go test` : le cache ne
   masque pas un échec, il ne s'applique qu'à des résultats positifs inchangés.

---

## 📌 À retenir

- Une seule commande, **`go`**, fait tout : compiler, exécuter, tester, formater, documenter.
- `go run` pour itérer, `go build` pour produire un **binaire autonome** (un seul fichier).
- La **cross-compilation** se fait avec `GOOS`/`GOARCH`, sans toolchain croisée — tant
  qu'aucune dépendance n'utilise cgo (sinon `CC` doit pointer un compilateur croisé).
- `GOTOOLCHAIN` télécharge **automatiquement** la version de Go exigée par `go.mod`,
  jamais en downgrade, et nécessite un accès au proxy de modules.
- Le code Go est **toujours formaté** par `gofmt` — formatez à la sauvegarde.

## 🔁 Pour aller plus loin

- [Ch. 2 — Structure d'un programme](02-structure-programme.md).
- [Ch. 12 — Packages, modules & organisation du code](12-packages-modules.md) pour `go.mod`/`go.sum`.
- [Annexe B — Antisèche des commandes `go`](../annexes/B-antiseche-go.md).
