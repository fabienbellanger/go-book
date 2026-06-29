# 46 — Embarquer & déployer

> **Objectif** — Transformer un programme Go en **artefact déployable** : embarquer
> ses ressources dans le binaire (`//go:embed`), compiler conditionnellement (build
> constraints), produire un **binaire statique** minimal et reproductible, puis le
> glisser dans une image conteneur réduite à l'essentiel.
>
> **Prérequis** — [Ch. 1 — Toolchain & cross-compilation](01-installation-toolchain.md),
> [Ch. 12 — Packages & modules](12-packages-modules.md)

---

## Introduction

Un binaire Go est déjà **autonome** : pas de runtime à installer, pas de VM, une seule
copie de fichier suffit ([Ch. 1](01-installation-toolchain.md)). Ce chapitre franchit la
dernière étape — faire en sorte qu'il embarque **tout ce dont il a besoin** (templates,
assets web, migrations SQL, sa propre version), qu'il se compile **au plus juste** pour la
cible, et qu'il se déploie dans un conteneur **minuscule**.

Le fil rouge tourne autour d'une idée simple : à la livraison, **un seul fichier**.

```
   Sources + assets                Build                 Déploiement
   +----------------+         +--------------+         +-----------------+
   | main.go        |         | CGO_ENABLED=0|         | image scratch / |
   | version.txt    | ----->  | -ldflags     | ----->  | distroless      |
   | templates/...  |  embed  | -trimpath    |  copie  |  +-----------+  |
   | migrations/... |         +--------------+         |  | 1 binaire |  |
   +----------------+              |                   |  +-----------+  |
                                   v                   +-----------------+
                            1 binaire statique
                            (assets DEDANS)
```

L'exemple complet est dans [`code/ch46-embed-build/`](../code/ch46-embed-build/).

---

## Embarquer des fichiers : `//go:embed`

La directive `//go:embed` copie des fichiers **dans le binaire à la compilation**. Plus de
« fichier introuvable » en production parce qu'on a oublié de livrer un dossier `templates/`.

Trois formes, selon le type de la variable juste **en dessous** de la directive :

```go
import _ "embed" // ⚠️ requis pour les cas string et []byte

//go:embed version.txt
var version string // tout le contenu du fichier, en texte

//go:embed logo.png
var logo []byte // contenu binaire brut
```

```go
import "embed" // pour le cas FS, l'import est NORMAL (pas blanc)

//go:embed templates
var templatesFS embed.FS // un sous-arbre entier, en lecture seule
```

Un `embed.FS` implémente `fs.FS` : tout l'écosystème `io/fs`
fonctionne dessus (`fs.ReadFile`, `fs.WalkDir`, `fs.Sub`, `template.ParseFS`, et même
`http.FileServerFS` — 🔁 [Ch. 45](45-net-http.md)).

```go
//go:embed templates
var templatesFS embed.FS

func renderWelcome(name string) (string, error) {
	t, err := template.ParseFS(templatesFS, "templates/welcome.txt")
	// ... Execute sur le gabarit embarqué
}
```

### Motifs

Après `//go:embed`, on liste un ou plusieurs **motifs** (`path.Match`), séparés par des
espaces ; les dossiers sont embarqués **récursivement** :

```go
//go:embed static templates/*.html migrations/*.sql
var assets embed.FS
```

### ⚠️ Règles à connaître

- La variable doit être **au niveau du package** (pas dans une fonction).
- Cas `string`/`[]byte` : il faut l'**import blanc** `_ "embed"` même si `embed` n'est pas
  référencé autrement. Le compilateur le rappelle si on l'oublie.
- **Pas de `..`** ni de chemin absolu : on n'embarque que des fichiers **sous le package**.
- Par défaut, les fichiers **cachés** (commençant par `.` ou `_`, et `testdata`) sont
  **exclus** d'un motif de dossier. Le préfixe `all:` force leur inclusion :
  `//go:embed all:static`.
- Le chemin d'accès au runtime garde **toujours des `/`** (jamais `\`), même sous Windows.

> 💡 **Cas d'usage typiques** : gabarits HTML/texte, assets d'un front (CSS/JS/images servis
> en HTTP), **migrations SQL**, fichiers de configuration par défaut, et le **numéro de
> version** versionné dans le dépôt.

---

## Compilation conditionnelle : build constraints

Une **build constraint** décide si un fichier participe — ou non — à la compilation, selon
l'OS, l'architecture ou des **tags** maison.

### Deux mécanismes

1. **Par nom de fichier** — un suffixe `_GOOS`, `_GOARCH` (ou les deux) restreint
   automatiquement le fichier :

   ```
   net_linux.go        compilé seulement sous Linux
   net_windows.go      compilé seulement sous Windows
   simd_amd64.go       compilé seulement sur amd64
   foo_test.go         compilé seulement par `go test`
   ```

2. **Par directive `//go:build`** — une expression booléenne en tête de fichier :

   ```go
   //go:build linux && amd64
                          <-- ⚠️ LIGNE VIDE OBLIGATOIRE ici
   package main
   ```

   Opérateurs : `&&`, `||`, `!`, parenthèses. Les termes sont des OS (`linux`, `darwin`,
   `windows`…), des archs (`amd64`, `arm64`…), `cgo`, la version de Go (`go1.26`), ou un
   **tag personnalisé**.

### 🆕 La syntaxe `//go:build`

Depuis Go 1.17, `//go:build` **remplace** l'ancienne `// +build`, plus fragile (logique
implicite, espaces signifiants). Les outils (`gofmt`) maintiennent les deux synchronisées
sur le code ancien, mais **pour du neuf : uniquement `//go:build`**.

```go
// ANCIEN (obsolète, ne plus écrire) :
// +build prod

// NOUVEAU :
//go:build prod
```

> ⚠️ Le piège n°1 : **oublier la ligne vide** après `//go:build`. Sans elle, la ligne est
> rattachée au commentaire de doc du package et **n'a plus aucun effet** — le fichier est
> compilé en permanence, sans avertissement.

### Tags personnalisés

On active un tag avec `-tags` :

```bash
go build -tags prod ./...
go test  -tags "prod integration" ./...
```

Dans [`code/ch46-embed-build/`](../code/ch46-embed-build/), deux fichiers exposent la même
fonction `featureName()` :

```go
//go:build !prod

func featureName() string { return "dev (build par défaut)" }
```

```go
//go:build prod

func featureName() string { return "prod (build -tags prod)" }
```

Les contraintes `prod` / `!prod` sont **exclusives** : exactement un des deux fichiers est
compilé, jamais les deux, jamais zéro. C'est le motif idiomatique pour des variantes
(édition libre/pro, stubs de tests d'intégration, implémentations spécifiques à un OS).

---

## Produire un binaire statique et minimal

### `CGO_ENABLED=0` : le vrai statique

Par défaut, certains packages standard (résolution DNS, `os/user`) peuvent passer par des
bibliothèques **C** du système via cgo, ce qui crée une **dépendance dynamique** (`libc`).
Pour un binaire **100 % statique**, portable d'une distribution à l'autre et exécutable dans
une image vide :

```bash
CGO_ENABLED=0 go build -o app ./cmd/app
```

Go bascule alors sur ses implémentations **pures Go**. C'est la base d'un déploiement
`scratch` (voir plus bas).

### Réduire la taille : `-ldflags="-s -w"`

L'éditeur de liens peut retirer la **table des symboles** (`-s`) et les **informations de
débogage DWARF** (`-w`). Le binaire perd la possibilité d'être inspecté par un débogueur,
mais gagne souvent **20 à 30 %** de taille — ⚡ appréciable pour une image conteneur.

```bash
go build -ldflags="-s -w" -o app ./cmd/app
```

| Flag `-ldflags`           | Effet                                                      |
| ------------------------- | ---------------------------------------------------------- |
| `-s`                      | retire la table des symboles                               |
| `-w`                      | retire les infos de débogage DWARF                         |
| `-X importpath.var=value` | **injecte** une valeur dans une variable `string` exportée |

### Injecter la version : `-ldflags -X`

`-X` affecte une variable `string` de **package** à la compilation. Convention :

```go
package main

// Cible de l'injection ; valeur de repli pendant le développement.
var version = "dev"
```

```bash
go build -ldflags="-X main.version=1.2.3" -o app ./cmd/app
```

> ⚠️ `-X` ne fonctionne que sur une **`var string`** (jamais une `const`, ni un autre type),
> et le chemin doit être **complet** (`main.version`, ou `example.com/app/build.Version`).

### Reproductibilité : `-trimpath`

`-trimpath` retire les **chemins absolus** de la machine de build (`/Users/alice/...`) des
binaires. Le résultat ne dépend plus de l'emplacement du code : indispensable pour des
**builds reproductibles** (même source → même binaire octet pour octet).

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=1.2.3" -o app ./cmd/app
```

### Cross-compilation (rappel 🔁 [Ch. 1](01-installation-toolchain.md))

`GOOS`/`GOARCH` produisent un binaire pour une **autre** plateforme, sans toolchain croisée :

```bash
GOOS=linux   GOARCH=amd64 go build -o app-linux-amd64   ./cmd/app
GOOS=linux   GOARCH=arm64 go build -o app-linux-arm64   ./cmd/app
GOOS=darwin  GOARCH=arm64 go build -o app-macos-arm64   ./cmd/app
GOOS=windows GOARCH=amd64 go build -o app-windows.exe   ./cmd/app
```

| `GOOS`    | `GOARCH`          | Cible courante                |
| --------- | ----------------- | ----------------------------- |
| `linux`   | `amd64` / `arm64` | serveurs, conteneurs          |
| `darwin`  | `arm64` / `amd64` | macOS (Apple Silicon / Intel) |
| `windows` | `amd64` / `arm64` | postes Windows                |
| `js`      | `wasm`            | navigateur (WebAssembly)      |

> `go tool dist list` énumère **tous** les couples supportés.

---

## Lire ses propres infos de build : `runtime/debug`

`-X` n'est pas la seule façon de connaître sa version : `go build` **injecte
automatiquement** les métadonnées du module et du VCS, lisibles via
`debug.ReadBuildInfo()` — sans aucun flag.

```go
func vcsRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" { // hash de commit injecté par `go build`
			return s.Value
		}
	}
	return ""
}
```

`BuildInfo` expose :

- `GoVersion`, `Path`, `Main.Version` (version du module, ex. `v1.4.0` ou `(devel)`) ;
- `Deps` — toutes les dépendances et leurs versions ;
- `Settings` — réglages de build, dont les clés VCS injectées **automatiquement** :
  `vcs.revision` (hash de commit), `vcs.time` (date du commit), `vcs.modified`
  (`true` si l'arbre de travail était sale).

> 💡 **`-X` vs `BuildInfo`.** `BuildInfo` est « gratuit » et idiomatique pour le **commit**
> et l'état VCS. `-X` reste utile pour une **étiquette sémantique** (`1.2.3`) issue d'un tag
> CI qui n'est pas le hash brut. Beaucoup d'outils combinent les deux.

> ⚠️ En `go test` / `go run`, les settings `vcs.*` ne sont **pas** renseignés (build hors
> mode release) : prévoir le cas `""`.

---

## Déployer en conteneur : image minimale

Un binaire statique se met dans une image **quasi vide**. Le motif standard est le
**Dockerfile multi-stage** : on compile dans une image `golang` complète, puis on **copie le
seul binaire** dans une image finale minuscule.

```dockerfile
# --- étape 1 : build ---
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

# --- étape 2 : image finale minimale ---
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
```

Pourquoi une image **réduite** (`scratch` = totalement vide, ou `distroless` = juste les
fichiers système indispensables) :

- **surface d'attaque minimale** : pas de shell, pas de gestionnaire de paquets, rien à
  exploiter ;
- **taille** : quelques Mo (le binaire) au lieu de centaines ;
- **démarrage** instantané.

> ⚠️ **Pièges de `scratch`.** Une image totalement vide ne contient **ni certificats CA**
> (les appels HTTPS échouent) **ni base de fuseaux horaires** (`time.LoadLocation` échoue).
> Solutions : copier les CA depuis l'étape de build
> (`COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/`), ou — plus simple —
> partir de **`gcr.io/distroless/static`**, qui inclut déjà CA, fuseaux et un utilisateur
> `nonroot`. Pour les fuseaux côté Go, l'import `_ "time/tzdata"` **embarque** la base dans
> le binaire.

> 💡 **`ko`** (`ko build ./cmd/app`) construit une image OCI **sans Dockerfile ni Docker**,
> directement depuis le code Go, de façon **reproductible** — pratique en CI pour des
> services Go purs.

---

## 🆕 Go 1.2x (rappels utiles ici)

- **1.16** — `//go:embed` et le package `embed` ; `io/fs`.
- **1.17** — syntaxe `//go:build` (remplace `// +build`).
- **1.18+** — `vcs.*` dans `BuildInfo.Settings` (commit, date, état sale) injectés par
  `go build`.
- **1.24** — directive **`tool`** dans `go.mod` (🔁 [Ch. 12](12-packages-modules.md)) pour
  épingler les outils de build/CI ; directive **`ignore`** (1.25) pour exclure des assets des
  motifs `./...`.

---

## ⚠️ Pièges

- **Ligne vide manquante** après `//go:build` → la contrainte est ignorée silencieusement.
- **`//go:embed` sur un fichier inexistant** → erreur de **compilation** (c'est voulu : on
  est prévenu tôt, pas en production).
- **Cas string/[]byte sans `_ "embed"`** → ne compile pas.
- **`-X` sur une `const`** ou un mauvais chemin → ignoré sans effet ; vérifier avec un
  `--version`.
- **Oublier `CGO_ENABLED=0`** puis déployer sur `scratch` → le binaire réclame `libc` et
  refuse de démarrer.
- **`scratch` sans CA** → toutes les requêtes HTTPS échouent par « certificat inconnu ».

## ⚡ Performance & taille

- `-ldflags="-s -w"` + `-trimpath` : binaire plus **petit** et **reproductible**.
- `embed` ne coûte rien à l'exécution (les octets sont déjà en mémoire/mappés) et **supprime**
  les I/O disque au démarrage pour charger templates/config.
- Image `distroless`/`scratch` : démarrage et _pull_ plus rapides, empreinte mémoire moindre.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch46-embed-build                 # version issue du fichier embarqué
go run -tags prod ./ch46-embed-build      # bascule sur la variante prod
go test ./ch46-embed-build/...

# injection de version + binaire minimal
go build -ldflags="-s -w -X main.version=2.0.0" -trimpath -o /tmp/app ./ch46-embed-build
/tmp/app                                  # affiche « version : 2.0.0 »
go version -m /tmp/app | head             # relit les BuildInfo du binaire
```

À essayer :

1. Supprimez la ligne vide après `//go:build prod` et observez que `featureName()` ne
   bascule plus jamais.
2. Renommez `version.txt` : la **compilation** échoue immédiatement (`pattern ... no matching
files found`).
3. Comparez la taille du binaire avec et sans `-ldflags="-s -w"`.

---

## 📌 À retenir

- **`//go:embed`** copie fichiers et dossiers **dans le binaire** à la compilation ;
  `embed.FS` implémente `fs.FS` (templates, assets HTTP, migrations).
- **Build constraints** (`//go:build`, suffixes `_os`/`_arch`, `-tags`) compilent
  conditionnellement — **ligne vide obligatoire** après la directive.
- **Binaire de prod** : `CGO_ENABLED=0` (statique) + `-trimpath` (reproductible) +
  `-ldflags="-s -w -X main.version=…"` (petit, versionné).
- **`debug.ReadBuildInfo()`** donne version de module et infos VCS (`vcs.revision/time/
modified`) **gratuitement**, sans `-X`.
- **Déploiement** : Dockerfile multi-stage → image **`scratch`/`distroless`** ; ⚠️ penser aux
  **CA** et aux **fuseaux** (`distroless/static` ou `_ "time/tzdata"`).

## 🔁 Pour aller plus loin

- [Ch. 1 — Toolchain & cross-compilation](01-installation-toolchain.md) : `GOOS`/`GOARCH`,
  `go build/env`.
- [Ch. 12 — Packages & modules](12-packages-modules.md) : `go.mod`, directives `tool`/`ignore`.
- [Ch. 45 — `net/http`](45-net-http.md) : servir un `embed.FS` via `http.FileServerFS` /
  `fs.Sub`.
- Projet 1 — Outil CLI : build, cross-compilation et distribution d'un binaire.
- Doc : `pkg.go.dev/embed`, `pkg.go.dev/runtime/debug#BuildInfo`, `go help buildconstraint`.
