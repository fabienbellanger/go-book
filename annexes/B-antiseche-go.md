# Annexe B — Antisèche des commandes `go`

> **Objectif** — Un aide-mémoire dense et pratique de l'outillage Go : pour chaque
> commande, son but, ses drapeaux les plus utiles et des exemples directement
> réutilisables. À garder ouvert à côté du terminal.

---

## `go build`

Compile les paquets et leurs dépendances (sans installer). Sur un paquet `main`,
produit un exécutable ; sur une bibliothèque, vérifie seulement que tout compile.

| Drapeau               | Effet                                                          |
| --------------------- | -------------------------------------------------------------- |
| `-o <chemin>`         | Nom/destination du binaire produit                             |
| `-ldflags "..."`      | Options passées à l'éditeur de liens (voir ci-dessous)         |
| `-gcflags "..."`      | Options du compilateur (ex. `-m` pour l'analyse d'échappement) |
| `-race`               | Active le détecteur de courses (binaire instrumenté)           |
| `-pgo=auto\|off\|<f>` | Optimisation guidée par profil (🆕 `auto` par défaut)          |
| `-tags "a,b"`         | Active des contraintes de build (`//go:build a`)               |
| `-trimpath`           | Retire les chemins absolus du binaire (builds reproductibles)  |
| `-v`                  | Affiche les paquets compilés                                   |

```bash
go build ./...                                   # compile tout le module
go build -o bin/app .                            # binaire nommé
go build -ldflags "-s -w -X main.version=v1.2.3" -o app .   # binaire allégé + version injectée
go build -gcflags="-m" ./... 2>&1 | grep escapes # voir ce qui échappe sur le tas
```

> 💡 `-ldflags "-s -w"` retire la table des symboles et les infos DWARF : binaire
> plus petit. `-X import/path.Var=valeur` injecte une chaîne dans une variable de
> paquet à la compilation (versioning sans recompiler le code).

---

## `go run`

Compile **et exécute** dans la foulée (binaire temporaire). Idéal pour itérer ;
à éviter en production.

| Drapeau    | Effet                |
| ---------- | -------------------- |
| `-race`    | Détecteur de courses |
| `-ldflags` | Comme `go build`     |
| `-tags`    | Contraintes de build |

```bash
go run .                       # exécute le paquet main du dossier courant
go run ./cmd/server -addr :8080
go run -race .                 # exécution instrumentée
```

> ⚠️ `go run fichier.go` ne compile QUE ce fichier : s'il dépend d'autres fichiers
> du même paquet, préférez `go run .` (tout le paquet).

---

## `go test`

Compile et lance les tests, benchmarks, exemples et fuzz du paquet. 🔁 Voir
Ch. 13 (tests) et Ch. 36 (benchmarks & fuzzing).

| Drapeau                | Effet                                                       |
| ---------------------- | ----------------------------------------------------------- |
| `-run <regex>`         | Ne lance que les tests dont le nom correspond               |
| `-bench <regex>`       | Lance les benchmarks correspondants (`.` = tous)            |
| `-benchmem`            | Ajoute octets/op et allocs/op au rapport de bench           |
| `-benchtime <d>\|<N>x` | Durée (`2s`) ou nombre d'itérations (`1000x`) par bench     |
| `-race`                | Détecteur de courses                                        |
| `-cover`               | Affiche le taux de couverture                               |
| `-coverprofile <f>`    | Écrit le profil de couverture (analysable, voir ci-dessous) |
| `-count <n>`           | Répète `n` fois (désactive le cache si `n>1`)               |
| `-fuzz <regex>`        | Lance le fuzzing d'une cible `FuzzXxx`                      |
| `-cpuprofile <f>`      | Profil CPU                                                  |
| `-memprofile <f>`      | Profil mémoire                                              |
| `-v`                   | Verbeux (chaque test nommé)                                 |
| `-short`               | Active `testing.Short()` (sauter les tests longs)           |
| `-timeout <d>`         | Tue la suite après ce délai (défaut 10m)                    |

```bash
go test ./...                                  # tout le module
go test -run TestParse -v ./internal/parser    # un test ciblé, verbeux
go test -race -count=1 ./...                    # courses, sans cache
go test -bench=. -benchmem -count=10 ./pkg | tee new.txt
go test -fuzz=FuzzDecode -fuzztime=30s ./pkg    # fuzzing borné
go test -coverprofile=cover.out ./... && go tool cover -html=cover.out
```

> 💡 Dans un benchmark moderne, bouclez avec `for b.Loop() { ... }` (🆕 Go 1.24) :
> la préparation hors boucle n'est pas chronométrée et le résultat n'est pas
> éliminé par le compilateur — plus besoin de `b.N` ni de variable « puits ».

---

## `go vet`

Analyse statique : signale du code qui compile mais est probablement faux
(formats `Printf` erronés, copies de mutex, balises de struct mal formées,
boucles douteuses…). À lancer en CI.

| Drapeau             | Effet                         |
| ------------------- | ----------------------------- |
| `-printf=false`     | Désactive un analyseur précis |
| (un analyseur seul) | `go vet -copylocks ./...`     |

```bash
go vet ./...
go tool vet help          # liste des analyseurs disponibles
```

---

## `go doc`

Affiche la documentation d'un paquet, type ou fonction, hors ligne, depuis le
terminal.

| Drapeau | Effet                                            |
| ------- | ------------------------------------------------ |
| `-all`  | Toute la doc du paquet (pas seulement le résumé) |
| `-src`  | Montre aussi le code source                      |
| `-u`    | Inclut les symboles non exportés                 |

```bash
go doc strings                 # synthèse du paquet
go doc strings.Builder         # un type
go doc -all sync.Once          # tout, y compris exemples
go doc -src io.Copy            # avec le source
```

---

## `go fix`

Réécrit du code pour suivre les évolutions des API (migrations automatiques entre
versions). À lancer après une montée de version, puis relire le diff.

```bash
go fix ./...
```

---

## `go tool`

Donne accès aux outils bas niveau de la distribution. Les plus utiles :

| Sous-outil         | Rôle                                                     |
| ------------------ | -------------------------------------------------------- |
| `pprof`            | Explorer un profil CPU/mémoire/blocage                   |
| `trace`            | Visualiser une trace d'exécution (`go tool trace t.out`) |
| `cover`            | Rendre un profil de couverture (`-func`, `-html`)        |
| `compile` / `link` | Étapes de compilation/édition de liens (debug avancé)    |
| `objdump`          | Désassembler un binaire                                  |
| `nm`               | Lister les symboles d'un objet/binaire                   |

```bash
go tool pprof cpu.pprof            # mode interactif : top, list, web
go tool pprof -top -nodecount=15 cpu.pprof
go tool pprof 'http://localhost:8080/debug/pprof/profile?seconds=5'
go tool trace trace.out            # ouvre l'explorateur de traces
go tool cover -func=cover.out      # couverture par fonction
```

> 💡 Dans `pprof` interactif : `top` (postes les plus coûteux), `list Fn` (coût
> ligne à ligne), `web` (graphe de flamme, nécessite Graphviz). 🔁 Voir Ch. 37–38.

---

## `go work` (espaces de travail multi-modules)

Travailler sur plusieurs modules locaux à la fois sans toucher leurs `go.mod`.

| Sous-commande | Effet                                              |
| ------------- | -------------------------------------------------- |
| `init`        | Crée `go.work`                                     |
| `use <dir>`   | Ajoute un module local à l'espace de travail       |
| `sync`        | Propage les versions de l'espace vers les `go.mod` |
| `edit`        | Modifie `go.work` par script                       |

```bash
go work init ./api ./shared      # un go.work référençant deux modules locaux
go work use ./outils             # en ajouter un troisième
go work sync
```

> ⚠️ Ne committez pas `go.work` dans une bibliothèque publiée : c'est un confort
> de développement local, pas une dépendance du module.

---

## `go mod` (modules & dépendances)

Gère le module courant et son graphe de dépendances.

| Sous-commande | Effet                                                     |
| ------------- | --------------------------------------------------------- |
| `init <path>` | Crée `go.mod` avec le chemin d'import du module           |
| `tidy`        | Ajoute les dépendances manquantes, retire les inutilisées |
| `download`    | Télécharge les modules dans le cache                      |
| `verify`      | Vérifie que le cache n'a pas été altéré                   |
| `why <pkg>`   | Explique pourquoi un module est requis                    |
| `graph`       | Affiche le graphe des dépendances                         |
| `edit`        | Édite `go.mod` par script (`-require`, `-replace`, `-go`) |

```bash
go mod init example.com/monapp
go mod tidy                       # LE réflexe après avoir ajouté/retiré un import
go mod why golang.org/x/sync/errgroup
go mod edit -go=1.26              # change la version de langage cible
```

---

## `go generate`

Exécute les commandes déclarées par des directives `//go:generate` dans le code.
Ne fait **rien** tout seul : ni `build` ni `test` ne le déclenchent ; c'est un
geste explicite, et le fichier produit est **versionné**. 🔁 Voir Projet 6.

```go
//go:generate stringer -type=Color
//go:generate enumgen
```

```bash
go generate ./...          # exécute toutes les directives du module
```

> ⚠️ La commande citée doit être dans le `PATH` (souvent `go install` au préalable).

---

## `go install`

Compile et installe un binaire dans `$GOBIN` (ou `$GOPATH/bin`). Sert aussi à
installer un outil par son chemin de module + version.

```bash
go install .                                         # installe le binaire courant
go install golang.org/x/perf/cmd/benchstat@latest    # un outil tiers, épinglé
go install honnef.co/go/tools/cmd/staticcheck@latest
```

---

## `go get`

Met à jour les dépendances **du module** (dans `go.mod`/`go.sum`). Depuis Go 1.17+,
**n'installe plus** de binaires (utilisez `go install` pour ça).

```bash
go get example.com/lib@v1.4.0     # version précise
go get example.com/lib@latest     # dernière version
go get -u ./...                   # met à jour les dépendances (mineures/patch)
go get example.com/lib@none       # retire une dépendance
```

---

## `go list`

Interroge les paquets et modules — précieux en scripts/CI.

```bash
go list ./...                                  # tous les paquets du module
go list -m all                                 # tous les modules du graphe
go list -m -u all                              # avec les mises à jour disponibles
go list -f '{{.ImportPath}} {{.Stale}}' ./...  # gabarit personnalisé
go list -json ./pkg                            # description complète en JSON
```

---

## `go env`

Affiche et configure la configuration de l'outillage (persistée dans
`go env -w`).

```bash
go env                          # tout
go env GOPATH GOMODCACHE        # quelques variables
go env -w GOFLAGS=-mod=readonly # persiste un réglage
go env -u GOFLAGS               # annule un réglage persistant
```

---

## `go version`

Version de la toolchain, et métadonnées de build d'un binaire.

```bash
go version                      # version de l'outil go
go version -m ./bin/app         # module, dépendances et réglages (dont -pgo) d'un binaire
```

---

## `go clean`

Nettoie les artefacts et caches.

| Drapeau      | Effet                                 |
| ------------ | ------------------------------------- |
| `-cache`     | Vide le cache de build                |
| `-testcache` | Vide le cache des résultats de test   |
| `-modcache`  | Vide le cache des modules téléchargés |

```bash
go clean -testcache       # forcer la réexécution de tous les tests
go clean -cache
```

---

## Variables d'environnement utiles

| Variable       | Rôle                                                                    |
| -------------- | ----------------------------------------------------------------------- |
| `GOFLAGS`      | Drapeaux ajoutés à toutes les commandes (ex. `-mod=readonly`)           |
| `GOOS`         | Système cible de compilation (`linux`, `darwin`, `windows`…)            |
| `GOARCH`       | Architecture cible (`amd64`, `arm64`…)                                  |
| `CGO_ENABLED`  | `0` désactive cgo → binaire statique, pur Go                            |
| `GOMAXPROCS`   | Nombre max d'OS threads exécutant du Go simultanément                   |
| `GOGC`         | Agressivité du GC (défaut `100` ; plus haut = moins de GC, plus de RAM) |
| `GOMEMLIMIT`   | Limite mémoire douce du runtime (ex. `512MiB`)                          |
| `GODEBUG`      | Drapeaux de debug du runtime (`gctrace=1`, `schedtrace=1000`…)          |
| `GOEXPERIMENT` | Active des fonctionnalités expérimentales de la toolchain               |
| `GOPATH`       | Racine héritée (cache modules, `bin/`) ; défaut `~/go`                  |
| `GOBIN`        | Destination des binaires de `go install`                                |
| `GOMODCACHE`   | Emplacement du cache des modules (défaut `$GOPATH/pkg/mod`)             |
| `GOPROXY`      | Proxy(s) de modules (ex. `https://proxy.golang.org,direct`, ou `off`)   |

```bash
GODEBUG=gctrace=1 ./app                 # une ligne par cycle de GC sur stderr
GODEBUG=schedtrace=1000 ./app           # état de l'ordonnanceur chaque seconde
GOFLAGS=-mod=readonly go build ./...     # interdit la modif implicite de go.mod
```

> 🔁 `GOGC`, `GOMEMLIMIT` et `GODEBUG=gctrace=1` sont détaillés au Ch. 27 (GC) ;
> `GOMAXPROCS` et `schedtrace` au Ch. 28 (ordonnanceur G-M-P).

---

## Cross-compilation

Go compile pour une autre plateforme sans chaîne d'outils externe : il suffit de
fixer `GOOS`/`GOARCH`. `CGO_ENABLED=0` garantit un binaire **statique**, sans
dépendance à la libc de la machine cible.

```bash
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o dist/app-linux-amd64 .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o dist/app-darwin-arm64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/app-windows-amd64.exe .

go tool dist list          # toutes les paires GOOS/GOARCH supportées
```

| GOOS      | GOARCH courants  | Cas d'usage                    |
| --------- | ---------------- | ------------------------------ |
| `linux`   | `amd64`, `arm64` | Serveurs, conteneurs           |
| `darwin`  | `amd64`, `arm64` | macOS Intel / Apple Silicon    |
| `windows` | `amd64`, `arm64` | Postes Windows (`.exe`)        |
| `js`      | `wasm`           | WebAssembly dans le navigateur |

> 💡 Boucler sur une liste `linux/amd64 linux/arm64 darwin/arm64 …` dans un
> `Makefile` produit en une cible tous les binaires de distribution. 🔁 Voir les
> `Makefile` des projets pratiques 1 et 5.

---

## 📌 À retenir

- `go build`/`run`/`test` sont le trio quotidien ; `-race` et `go vet ./...` sont
  des réflexes de qualité, `go mod tidy` celui d'après chaque changement d'import.
- Le profilage passe par `go test -cpuprofile/-memprofile` puis `go tool pprof` ;
  les traces par `go tool trace`.
- Injecter une version : `-ldflags "-X import/path.Var=valeur"` ; alléger :
  `-ldflags "-s -w"`.
- Cross-compiler = `GOOS`/`GOARCH` (+ `CGO_ENABLED=0` pour du statique) ; aucune
  chaîne d'outils externe nécessaire.
- `GODEBUG=gctrace=1` et `GOMAXPROCS`/`GOGC`/`GOMEMLIMIT` ouvrent une fenêtre sur
  le runtime sans recompiler.
