# 1 — Installation, toolchain & premier programme

> **Objectif** — Avoir un environnement Go fonctionnel, comprendre la toolchain, et
> exécuter son premier programme.
>
> **Prérequis** — [Ch. 0 — Pourquoi Go ?](00-pourquoi-go.md)

---

## Introduction

Go se distingue par une **toolchain unique et intégrée** : une seule commande, `go`,
fait tout — compiler, exécuter, tester, formater, documenter, analyser. Pas de
`Makefile` obligatoire, pas de gestionnaire de paquets tiers. Ce chapitre installe
cet environnement et exécute un premier programme.

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
```

| Variable          | Rôle                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `GOROOT`          | où Go est **installé** (réglé automatiquement, ne pas y toucher en général).                                             |
| `GOPATH`          | dossier des binaires et du cache (`~/go` par défaut). _Héritage_ de l'ère pré-modules : on n'y met plus son code source. |
| `GOBIN`           | où `go install` dépose les exécutables (`$GOPATH/bin` par défaut). À ajouter au `PATH`.                                  |
| `GOTOOLCHAIN`     | gère le **téléchargement automatique** de la bonne version de Go selon `go.mod` (voir ci-dessous).                       |
| `GOOS` / `GOARCH` | système et architecture **cibles** de la compilation (cross-compilation).                                                |

> 💡 Depuis les **modules** (Go 1.11+), `GOPATH` n'est plus l'endroit où l'on range son
> code. Vos projets vivent **où vous voulez**, du moment qu'ils contiennent un `go.mod`.

### Gestion automatique de la toolchain

Depuis Go 1.21, le fichier `go.mod` peut exiger une version minimale (`go 1.26`) voire
une toolchain précise (`toolchain go1.26.4`). Si votre `go` local est trop ancien,
**il télécharge et utilise automatiquement** la bonne version. Ce comportement est piloté
par `GOTOOLCHAIN` (`auto` par défaut). Conséquence : plus de « ça compile chez moi mais
pas chez toi » dû à une version différente.

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
- **`func main()`** — le **point d'entrée** : l'exécution démarre ici.
- **`fmt.Println`** — affiche une ligne sur la sortie standard.

> ⚠️ Les **identifiants** (variables, fonctions, types) sont en **anglais** dans tous les
> exemples du livre, mais les **commentaires** sont en **français**. C'est une convention
> du livre, et une bonne pratique pour du code partagé.

## Anatomie d'un binaire & cross-compilation

`go build` produit un **exécutable autonome** : il embarque le runtime Go et toutes les
dépendances. Aucun interpréteur ni bibliothèque partagée à installer sur la machine
cible — on copie un seul fichier.

```bash
cd code
go build -o hello ./ch01-hello
./hello            # Bonjour, Go ! 👋
ls -lh hello       # ~1-2 Mo : runtime Go inclus
```

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

Listez toutes les cibles possibles avec `go tool dist list`.

## Formatage & éditeurs

- **`gofmt`** (via `go fmt`) impose **un seul style** : indentation par tabulations,
  accolades placées automatiquement. Il n'y a **pas de débat** — on formate, c'est tout.
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

Le programme est accompagné d'un test (`code/ch01-hello/main_test.go`). Lancez :

```bash
cd code
go test ./ch01-hello/...   # ok   example.com/gobook/ch01-hello
go vet ./ch01-hello/...    # (aucune sortie = tout va bien)
```

Puis expérimentez :

1. Modifiez `greet` pour saluer **votre** prénom et relancez `go run ./ch01-hello`.
2. Produisez un binaire pour Windows : `GOOS=windows GOARCH=amd64 go build ./ch01-hello`.
3. Cassez volontairement le format (enlevez une indentation) puis lancez `go fmt ./...`.

---

## 📌 À retenir

- Une seule commande, **`go`**, fait tout : compiler, exécuter, tester, formater, documenter.
- `go run` pour itérer, `go build` pour produire un **binaire autonome** (un seul fichier).
- La **cross-compilation** se fait avec `GOOS`/`GOARCH`, sans toolchain croisée.
- `GOTOOLCHAIN` télécharge **automatiquement** la version de Go exigée par `go.mod`.
- Le code Go est **toujours formaté** par `gofmt` — formatez à la sauvegarde.

## 🔁 Pour aller plus loin

- [Ch. 2 — Structure d'un programme](02-structure-programme.md).
- [Ch. 12 — Packages, modules & organisation du code](12-packages-modules.md) pour `go.mod`/`go.sum`.
- Annexe B — Antisèche des commandes `go`.
