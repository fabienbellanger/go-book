# Plan détaillé — Livre « Comprendre et maîtriser Go 1.26 »

> Document de travail issu de `IDEA.md`. Il fige la structure, les conventions et le
> contenu chapitre par chapitre avant la rédaction. À valider/amender avant écriture.

---

## 1. Décisions structurantes (validées)

| Décision            | Choix retenu                                                                                                                                                                                              |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Progression**     | Top-down : _fondamentaux du langage → mécanismes avancés → concurrence → runtime/mémoire → internals des types → performance/outils → projets_. On apprend à **utiliser** Go, puis on **ouvre le capot**. |
| **Public**          | Développeur·se expérimenté·e (autre langage + algo), **nouveau en Go**. On enseigne la syntaxe Go depuis zéro, sans réexpliquer les bases de la programmation.                                            |
| **Objectif double** | _Apprendre_ (écrire du Go idiomatique) **et** _comprendre_ (runtime, GC, scheduler, layout mémoire).                                                                                                      |
| **Version**         | Go **1.26** (les nouveautés 1.21 → 1.26 sont signalées par des encarts `🆕`).                                                                                                                             |
| **Projets**         | 7 projets pratiques (CLI, API REST, pipeline concurrent, lib générique, service réseau, générateur de code, profiling).                                                                                   |

### Principe pédagogique : la « spirale » au niveau macro

Les types (slices, maps, strings, interfaces) sont d'abord vus **côté usage** en Partie I,
puis **côté implémentation** en Partie V. On annonce systématiquement le renvoi
(« 🔁 internals au chap. 33 ») pour que les deux lectures se répondent sans se répéter.

---

## 2. Conventions de rédaction

Reprises de `IDEA.md`, complétées :

- **Langue** : français. Markdown simple à écrire.
- **Code** : identifiants (variables, types, fonctions) en **anglais**, **commentaires en français**.
- **Exemples** : courts, compilables, exécutables (un module Go réel dans `code/`).
- **Schémas** : **ASCII pur** uniquement (caractères `+ - | / \ < > v ^`), jamais de box-drawing Unicode, pour garantir l'alignement partout.
- **Densité** : détaillé et précis, **sans verbosité** — privilégier schémas, exemples, listes, tableaux.
- **Émojis** : autorisés, avec parcimonie, comme repères visuels (`🆕` nouveauté, `⚠️` piège, `💡` astuce, `🔁` renvoi, `⚡` perf, `🧪` test).
- **Un fichier par chapitre**.
- **Encarts récurrents** par chapitre :
  - `🆕 Go 1.2x` — ce qui a changé récemment.
  - `⚠️ Pièges` — erreurs classiques.
  - `⚡ Performance` — coût, allocations, alternatives.
  - `🧪 À tester soi-même` — petit exercice ou benchmark.
  - `📌 À retenir` — synthèse en 3-5 puces.

### Exemples de schémas ASCII (charte de style)

En-tête de slice :

```
  s := arr[1:4]

  s  (slice header = 3 mots machine)
  +---------+---------+---------+
  |  ptr    |  len=3  |  cap=4  |
  +----+----+---------+---------+
       |
       v
  arr [0] [1] [2] [3] [4]
           ^---- len ----^
           ^------- cap -------^
```

Ordonnanceur G-M-P :

```
     P0             P1             P2          P = processeur logique (= GOMAXPROCS)
   +------+       +------+       +------+
   |  M   |       |  M   |       |  M   |      M = thread OS
   +------+       +------+       +------+
   |  G   |       |  G   |       |  G   |      G = goroutine en exécution
   +------+       +------+       +------+
   LRQ:           LRQ:           LRQ:          LRQ = file locale
   [G][G]         [G]            [G][G][G]
                                    |  vol de travail (work stealing)
                                    +--------> P1 pioche ici quand sa file est vide

   GRQ (file globale) : [G][G][G] ...   netpoller : [G en attente I/O]
```

---

## 3. Arborescence du dépôt proposée

```
go-book/
├─ README.md                 présentation + sommaire + parcours de lecture
├─ SOMMAIRE.md               table des matières cliquable (générée/maintenue)
├─ PLAN.md                   ce document
├─ IDEA.md                   brief d'origine
├─ chapitres/
│  ├─ 00-introduction.md
│  ├─ 01-installation-outils.md
│  ├─ 02-...
│  └─ ...
├─ code/                     exemples compilables (un module Go unique)
│  ├─ go.mod                 module example.com/gobook (go 1.26)
│  ├─ ch07-slices/
│  │  ├─ main.go
│  │  └─ slices_test.go
│  └─ ...
├─ projets/
│  ├─ 1-cli/
│  ├─ 2-api-rest/
│  ├─ 3-pipeline/
│  ├─ 4-lib-generique/
│  ├─ 5-service-reseau/
│  ├─ 6-codegen/
│  └─ 7-profiling/
└─ annexes/
   ├─ A-glossaire.md
   ├─ B-antiseche-go.md
   └─ ...
```

> Tout le code des chapitres vit dans **un seul module** `code/` afin que `go test ./...`
> et `go vet ./...` valident l'ensemble du livre en une commande (intégrable en CI).

---

## 4. Parcours de lecture

- **🟢 Débutant Go** : Parties 0 → I → II → III, puis projets 1 et 2. (Les parties IV-V-VI sont l'« approfondissement ».)
- **🟡 Lecture intégrale** : dans l'ordre — c'est le parcours conçu.
- **🔵 « Je connais Go, je veux les internals »** : Parties IV → V → VI, en piochant les renvois `🔁` vers la Partie I.
- **🟣 Focus concurrence** : Partie III → chap. 30 (scheduler) → chap. 27 (modèle mémoire) → projet 3.
- **🟠 Focus performance** : Partie VI → chap. 28-29 (alloc/GC) → projet 7.

---

## 5. Sommaire détaillé

Légende : chaque chapitre liste **Objectif**, **Contenu**, et selon les cas **Schémas**,
**Exemples**, **Perf/Tests**, **🆕 versions**.

---

### PARTIE 0 — Introduction & mise en route

#### Ch. 0 — Pourquoi Go ? Philosophie & panorama

- **Objectif** : situer Go, comprendre ses partis pris, savoir lire le livre.
- **Contenu** : histoire courte (Google, 2009) ; objectifs de design (simplicité, lisibilité, compilation rapide, concurrence native, outillage intégré) ; compilé/statiquement typé/GC ; comparaison express vs C/Java/Python ; cycle de release semestriel et compatibilité (« Go 1 promise ») ; panorama des nouveautés 1.21 → 1.26 ; comment lire ce livre (parcours, conventions).
- **📌 À retenir** : la culture « less is more », l'outillage comme fonctionnalité du langage.

#### Ch. 1 — Installation, toolchain & premier programme

- **Objectif** : avoir un environnement fonctionnel et exécuter du Go.
- **Contenu** : installation (`go`), variables d'environnement (`GOROOT`, `GOPATH` _legacy_, `GOBIN`, `GOTOOLCHAIN`) ; commandes cœur `go run / build / test / vet / fix / fmt / doc / env` ; anatomie d'un binaire statique (cross-compilation `GOOS`/`GOARCH`) ; `gofmt`/`goimports` ; éditeurs & `gopls` ; premier programme « Hello ».
- **Exemples** : `hello/main.go` ; cross-compile linux/arm64 depuis macOS.
- **🆕 1.25** : `go.mod` `ignore`, `go doc -http`. **🆕 1.26** : `go fix` (modernizers).

---

### PARTIE I — Fondamentaux du langage

#### Ch. 2 — Structure d'un programme : packages, `import`, `main`

- **Objectif** : comprendre l'unité de compilation et le point d'entrée.
- **Contenu** : package `main` vs bibliothèque ; `import` (chemins, alias, blank `_`, dot import) ; identifiants exportés (majuscule) ; `func main` / `func init` (ordre) ; commentaires de doc ; `go doc`.
- **Schéma** : graphe de dépendances de packages + ordre d'initialisation.

#### Ch. 3 — Variables, constantes & types de base

- **Objectif** : déclarer et typer des valeurs.
- **Contenu** : `var`, `:=`, déclarations groupées ; **zero values** ; entiers/flottants/complexes, `bool`, `byte`/`rune` ; conversions explicites (pas de coercition implicite) ; constantes typées/**non typées**, `iota` ; portée & _shadowing_ ; built-ins `len/cap/new/make/append/min/max/clear`.
- **Schéma** : table des types numériques (taille, signé, intervalle).
- **🆕 1.21** : `min`/`max`/`clear`. **🆕 1.26** : `new(expr)` accepte une expression d'initialisation.
- **⚠️ Pièges** : débordement silencieux, `int` vs `int64`, conversions de précision.

#### Ch. 4 — Flux de contrôle

- **Objectif** : maîtriser branchements et boucles idiomatiques.
- **Contenu** : `if`/`else` (+ statement d'init) ; `for` (3 formes) ; `range` sur slice/map/string/chan ; `range` sur **entier** ; `switch` (usage de base, renvoi ch. 15) ; `break`/`continue`/labels/`goto`.
- **🆕 1.22** : `for range N` (entier) ; **portée par itération** de la variable de boucle (le piège historique de capture disparaît — détaillé ch. 16).
- **Schéma** : table de décision des formes de `for`.

#### Ch. 5 — Fonctions

- **Objectif** : écrire et composer des fonctions.
- **Contenu** : signatures, **retours multiples**, retours nommés, variadiques, fonctions comme valeurs/paramètres, récursivité, passage par valeur (toujours), pointeurs en paramètre ; convention erreur en dernier retour.
- **Exemples** : `divmod`, callback, option pattern (teaser).
- **⚡ Perf** : coût d'appel, inlining (renvoi ch. 41).

#### Ch. 6 — Arrays & slices (usage)

- **Objectif** : manipuler les séquences, le modèle mental du slice.
- **Contenu** : arrays (taille fixe, valeur) ; slices (`make`, littéraux) ; **header ptr/len/cap** ; `append` (réallocation), `copy`, slicing `a[i:j:k]` ; aliasing & sous-slices ; slices de slices.
- **Schéma** : header de slice + réallocation d'`append`.
- **⚠️ Pièges** : aliasing après `append`, fuite mémoire via grand backing array.
- **🔁 Internals** : croissance, allocation pile/tas → ch. 31.

#### Ch. 7 — Maps & strings (usage)

- **Objectif** : tables associatives et texte.
- **Contenu** : `map` (`make`, littéraux, `comma-ok`, `delete`, itération **non ordonnée**) ; strings immuables, `byte` vs `rune`, UTF-8, `range` sur string, conversions `[]byte`/`[]rune` ; packages `strings`, `strconv`, `unicode/utf8`, `strings.Builder`.
- **Schéma** : encodage UTF-8 d'une chaîne multi-octets (indices byte vs rune).
- **🔁 Internals** : maps → ch. 32, strings → ch. 31.

#### Ch. 8 — Structs, méthodes & composition

- **Objectif** : modéliser des données et leur comportement.
- **Contenu** : `struct`, champs, tags ; littéraux & champs nommés ; méthodes ; **récepteur valeur vs pointeur** (règles de choix) ; **embedding** (composition, promotion de champs/méthodes) ; structs vides, alignement/padding (teaser ch. 35).
- **Schéma** : layout mémoire d'un struct avec padding.
- **⚠️ Pièges** : copie de gros structs, récepteurs mixtes.

#### Ch. 9 — Interfaces (fondamentaux)

- **Objectif** : abstraction par le comportement.
- **Contenu** : déclaration, **satisfaction implicite** ; `any` (`interface{}`) ; type assertions & `comma-ok` ; **type switch** ; interfaces idiomatiques (`Stringer`, `io.Reader`/`Writer`, `error`) ; petites interfaces & acceptation large/retour concret.
- **Schéma** : valeur d'interface = (type, valeur) — teaser de `iface`/`eface`.
- **🔁 Internals** : itab, dispatch → ch. 33.
- **⚠️ Pièges** : interface nil vs pointeur nil contenu.

#### Ch. 10 — Gestion des erreurs

- **Objectif** : le modèle d'erreur Go, idiomatique et robuste.
- **Contenu** : type `error` ; `errors.New`, `fmt.Errorf` + `%w` (wrapping) ; chaînes d'erreurs ; `errors.Is` / `errors.As` ; erreurs sentinelles vs types d'erreur ; quand `panic` n'est pas une erreur ; `defer` (intro, détail ch. 16).
- **🆕 1.26** : `errors.AsType[E]` (variante générique, typée et plus rapide que `As`).
- **🆕 1.26** : `fmt.Errorf("x")` alloue autant que `errors.New`.
- **📌 À retenir** : erreurs = valeurs ; on les enrichit, on ne les masque pas.

#### Ch. 11 — Généricité : types paramétrés

- **Objectif** : polymorphisme à la compilation, sans surcoût d'interface.
- **Contenu** : paramètres de type, contraintes, `comparable`, `~` (underlying), inférence ; fonctions et types génériques ; packages `slices`, `maps`, `cmp` ; **quand NE PAS** utiliser les génériques ; instanciation (renvoi ch. 39 pour la stratégie GC-shape/monomorphisation).
- **🆕 1.21** : `slices`/`maps`/`cmp`. **🆕 1.24** : alias de type génériques. **🆕 1.26** : contraintes auto-référentielles (`type Adder[A Adder[A]] interface{ Add(A) A }`).
- **⚡ Perf** : génériques vs interfaces vs duplication.

#### Ch. 12 — Packages, modules & organisation du code

- **Objectif** : structurer, versionner, distribuer du code.
- **Contenu** : `go.mod`/`go.sum`, SemVer, `go get`/`go mod tidy` ; packages `internal/` ; visibilité ; workspaces `go work` ; dépendances outils (`go tool`) ; documentation (doc comments, `Example` testables) ; mise en page d'un projet (layout pragmatique).
- **🆕 1.24** : dépendances outils dans `go.mod`. **🆕 1.25** : directive `ignore`, `go doc -http`.

#### Ch. 13 — Tests & outillage de base

- **Objectif** : la culture du test, intégrée au langage.
- **Contenu** : `testing`, `go test` ; tests **table-driven** ; `t.Run`/sous-tests ; helpers `t.Helper()` ; `Example` comme doc exécutable ; `t.TempDir`, `t.Cleanup` ; couverture (`-cover`, `-coverprofile`) ; `go vet` & analyzers ; teaser benchmarks/fuzzing (détail ch. 36).
- **🆕 1.25** : `T.Attr`/`T.Output`, analyzers `waitgroup`/`hostport`. **🆕 1.26** : `T.ArtifactDir` + `-artifacts`.
- **🧪** : premier test table-driven complet.

---

### PARTIE II — Mécanismes avancés du langage

#### Ch. 14 — `switch` & sélection de cas (en profondeur)

- **Objectif** : exploiter toute la puissance de `switch`.
- **Contenu** : switch d'expression, sans condition (= `if/else if`), `fallthrough`, cas multiples, **type switch** avancé, switch sur `init; cond` ; ce que le compilateur génère (jump table vs comparaisons en cascade) ; `select` annoncé (ch. 20).
- **⚡ Perf** : binary search / jump table selon densité des cas.

#### Ch. 15 — Fonctions anonymes & closures

- **Objectif** : comprendre la capture et l'état des closures.
- **Contenu** : fonctions littérales ; **capture par référence** ; durée de vie & échappement (renvoi ch. 28) ; closures à état ; patterns (décorateur, middleware, option pattern, mémoïsation).
- **🆕 1.22** : portée par itération de la variable de boucle — **l'ancien piège `for { go func(){ use(i) } }` n'existe plus** ; on montre l'avant/après.
- **Schéma** : closure capturant une variable promue sur le tas.

#### Ch. 16 — `defer` : garanties d'exécution

- **Objectif** : maîtriser le nettoyage déterministe.
- **Contenu** : sémantique LIFO ; évaluation des arguments **à l'enregistrement** ; interaction avec retours **nommés** ; `defer` en boucle (⚠️) ; patterns (unlock, close, trace d'entrée/sortie, recover).
- **⚡ Perf** : _open-coded defers_ (coût quasi nul) vs defer en boucle.
- **Schéma** : pile de defers d'une fonction.

#### Ch. 17 — `panic` & `recover`

- **Objectif** : gérer les conditions exceptionnelles sans en abuser.
- **Contenu** : quand paniquer (bugs, invariants) vs renvoyer une erreur ; `recover` dans un `defer` ; re-panic ; panique et goroutines (non rattrapable d'une autre goroutine ⚠️) ; déroulement de pile ; pattern « frontière de recover » (serveur HTTP).
- **🆕 1.25** : format de sortie de panique re-déclenchée (`[recovered, repanicked]`).

#### Ch. 18 — Itérateurs par fonction (range-over-func)

- **Objectif** : écrire et composer des séquences paresseuses.
- **Contenu** : `iter.Seq[V]` / `iter.Seq2[K,V]` ; `for x := range myIter` ; écrire un itérateur (push) ; itérateurs _pull_ (`iter.Pull`) ; `slices.Values`/`All`/`Sorted`, `maps.Keys`/`Values` ; composition (map/filter/take) ; gestion de l'arrêt anticipé.
- **🆕 1.23** : `iter`, range-over-func.
- **Schéma** : flux push vs pull (qui appelle qui).
- **⚡ Perf** : inlining des `yield`, coût vs slice matérialisée.

---

### PARTIE III — Concurrence

#### Ch. 19 — Goroutines : le modèle

- **Objectif** : lancer et raisonner sur des tâches concurrentes.
- **Contenu** : `go` ; goroutine vs thread OS ; coût (~ko de pile, croissance) ; cycle de vie ; concurrence ≠ parallélisme ; **fuites de goroutines** (introduction) ; arrêt propre.
- **🔁 Internals** : G-M-P, pile croissante → ch. 28, 30.
- **🆕 1.26** : profil expérimental `goroutineleak` (détail ch. 31/39).

#### Ch. 20 — Channels & `select`

- **Objectif** : communiquer par les canaux (« share memory by communicating »).
- **Contenu** : canaux bufferisés/non bufferisés ; envoi/réception/`close` ; `range` sur canal ; `select` (multiplexage, `default`, timeout via `time.After`) ; directions (`chan<-`, `<-chan`) ; nil channel ; patterns signaling (done, fan-in).
- **Schéma** : rendez-vous non bufferisé vs file bufferisée.
- **⚠️ Pièges** : envoi sur canal fermé, deadlock, fuite par canal non drainé.

#### Ch. 21 — Primitives de synchronisation

- **Objectif** : protéger l'état partagé quand les canaux ne suffisent pas.
- **Contenu** : `sync.Mutex`/`RWMutex` ; `sync.WaitGroup` ; `sync.Once` ; `sync.Cond` ; `sync.Pool` ; `sync.Map` (cas d'usage) ; `sync/atomic` (types `atomic.Int64`/`Pointer[T]`…).
- **🆕 1.25** : `WaitGroup.Go()` (lance + compte) ; analyzer `vet waitgroup`.
- **⚡ Perf** : contention, `RWMutex` vs `Mutex`, atomics vs mutex, faux partage (false sharing).
- **🔁** : garanties mémoire → ch. 27.

#### Ch. 22 — `context` : annulation, délais, valeurs

- **Objectif** : propager l'annulation et les deadlines.
- **Contenu** : `context.Context`, `WithCancel`/`WithTimeout`/`WithDeadline`/`WithCancelCause` ; propagation dans les appels ; `ctx.Done()`/`Err()` ; valeurs de contexte (et leur usage mesuré) ; conventions d'API.
- **⚠️ Pièges** : context oublié, `context.Value` comme sac fourre-tout.

#### Ch. 23 — Patterns de concurrence, data races & tests concurrents

- **Objectif** : composer des systèmes concurrents corrects et **testables**.
- **Contenu** : pipelines (fan-out/fan-in), worker pools, _bounded parallelism_, _rate limiting_, `errgroup` ; **data race** (définition) ; **race detector** (`go test -race`, `go run -race`) ; tests déterministes du temps et de la concurrence.
- **🆕 1.25** : `testing/synctest` (GA) — bulle isolée + horloge virtuelle (`synctest.Test`/`Wait`).
- **🆕 1.26** : profil `goroutineleak` pour détecter les goroutines bloquées.
- **🧪** : tester un timeout sans `time.Sleep` réel grâce à `synctest`.

---

### PARTIE IV — Runtime & modèle mémoire

#### Ch. 24 — Architecture du runtime & bootstrap

- **Objectif** : que se passe-t-il avant et autour de `main` ?
- **Contenu** : rôle du runtime (scheduler, GC, allocateur, netpoller) lié statiquement ; du `_rt0` OS au `runtime.main` ; création de la goroutine principale et de `g0`/`m0` ; ordre d'initialisation des packages et des `init()` ; `GODEBUG`/`GOMAXPROCS` au démarrage.
- **Schéma** : séquence de bootstrap (OS → runtime → main).

#### Ch. 25 — Le modèle mémoire de Go

- **Objectif** : savoir quand une écriture est visible par une autre goroutine.
- **Contenu** : relation **happens-before** ; garanties de `go`, canaux, `sync`, `sync/atomic` ; absence de garantie sans synchronisation ; pourquoi `-race` est indispensable ; pièges de double-checked locking.
- **Schéma** : arêtes happens-before établies par un canal.

#### Ch. 26 — Allocation mémoire & escape analysis

- **Objectif** : comprendre où vivent les données (pile vs tas).
- **Contenu** : pile par goroutine (croissance/copie) ; **escape analysis** (lire `-gcflags=-m`) ; allocateur (`mcache`/`mcentral`/`mheap`, size classes, tiny allocator) ; `make`/`new` ; réduction d'allocations.
- **🆕 1.25/1.26** : backing store de slices alloué **sur la pile** dans davantage de cas.
- **Schéma** : carte mémoire (pile, tas, spans/size classes).
- **⚡ Perf** : `-benchmem`, `allocs/op`, `sync.Pool`.

#### Ch. 27 — Le garbage collector

- **Objectif** : comprendre et régler le GC.
- **Contenu** : mark-sweep **concurrent tri-couleur**, write barriers, _GC pacing_ ; `GOGC`, `GOMEMLIMIT` ; phases & temps de pause ; finalizers, `runtime.AddCleanup`, pointeurs faibles (`weak`).
- **🆕 1.24** : `weak`, `AddCleanup`. **🆕 1.25** : `AddCleanup` concurrent/parallèle, `GODEBUG=checkfinalizers=1`, GC **Green Tea** (expérimental). **🆕 1.26** : **Green Tea GC par défaut** (−10 à −40 % d'overhead), randomisation de l'adresse de base du tas.
- **Schéma** : marquage tri-couleur (blanc/gris/noir) + write barrier.

#### Ch. 28 — L'ordonnanceur (G-M-P)

- **Objectif** : comprendre comment les goroutines tournent.
- **Contenu** : modèle **G-M-P**, files locales/globale, **work stealing** ; _handoff_ sur syscall bloquant ; **netpoller** (I/O non bloquante) ; préemption asynchrone ; `GOMAXPROCS`.
- **🆕 1.25** : `GOMAXPROCS` **conscient des cgroups** (limites conteneur), réajustement dynamique, `runtime.SetDefaultGOMAXPROCS`.
- **Schéma** : G-M-P + work stealing (cf. charte §2) ; cycle d'un syscall bloquant.

#### Ch. 29 — Observabilité du runtime & monitoring

- **Objectif** : voir ce que fait le runtime en production.
- **Contenu** : package `runtime` (`NumGoroutine`, `ReadMemStats`) ; `runtime/metrics` ; `runtime/debug` (`BuildInfo`, `SetGCPercent`, `FreeOSMemory`) ; `GODEBUG` (`gctrace=1`, `schedtrace`, `inittrace`) ; exposition (expvar, Prometheus) ; mappings nommés VMA sous Linux.
- **🆕 1.25** : `decoratemappings` (noms VMA `[anon: Go: heap]`). **🆕 1.26** : métriques `/sched/goroutines`, `/sched/threads`, `/sched/goroutines-created` ; profil `goroutineleak`.

---

### PARTIE V — Internals des structures de données & du système de types

#### Ch. 30 — Slices & arrays en profondeur

- **Objectif** : maîtriser le coût réel des slices.
- **Contenu** : layout du header ; **stratégie de croissance** d'`append` (amortissement, facteur) ; expression à 3 indices et contrôle de `cap` ; `copy` ; pièges d'aliasing et de rétention mémoire ; patterns _zero-allocation_ ; package `slices`.
- **⚡ Perf** : préallouer `cap`, réutiliser un buffer, `slices.Clip`.
- **Schéma** : évolution de cap au fil des `append`.

#### Ch. 31 — Strings en profondeur

- **Objectif** : comprendre l'immutabilité et les conversions.
- **Contenu** : backing immuable, header (ptr/len) ; conversions `string`↔`[]byte` (copie vs optimisations sans copie) ; `strings.Builder` (amortissement) ; UTF-8 en détail ; **interning** via `unique` ; comparaison/hash.
- **🆕 1.23/1.24** : package `unique`.
- **⚡ Perf** : éviter les conversions superflues, `unsafe` (renvoi ch. 37).

#### Ch. 32 — Maps : tables de hachage

- **Objectif** : ouvrir la table de hachage.
- **Contenu** : buckets, hashing, facteur de charge, croissance & évacuation incrémentale, randomisation d'itération (et pourquoi) ; non sûreté concurrente ; `sync.Map` vs map+mutex.
- **🆕 1.24** : implémentation **Swiss Tables** (gains mémoire/CPU).
- **Schéma** : bucket, tophash, overflow.
- **⚡ Perf** : préallouer (`make(map, n)`), clés coûteuses.

#### Ch. 33 — Interfaces & système de types en profondeur

- **Objectif** : comprendre le dispatch dynamique et son coût.
- **Contenu** : `eface` (type vide) vs `iface` (`itab` + data) ; construction d'`itab`, mise en cache ; dispatch dynamique ; coût des assertions/conversions ; **boxing**/allocations à la conversion ; piège interface-nil-non-nil.
- **🆕 1.25** : `reflect.TypeAssert` (évite l'allocation de `Value.Interface`).
- **Schéma** : `iface = (*itab, *data)`, `itab = (type, []méthodes)`.

#### Ch. 34 — Réflexion (`reflect`)

- **Objectif** : introspecter et manipuler dynamiquement.
- **Contenu** : `Type`/`Value`/`Kind` ; lire/écrire (`CanSet`, `Addr`) ; tags de struct (décodeurs, ORM) ; appel dynamique ; coût & alternatives (génériques, code-gen).
- **🆕 1.25** : `TypeAssert`. **🆕 1.26** : itérateurs `Type.Fields/Methods/Ins/Outs`, `Value.Fields/Methods`.
- **⚠️** : la réflexion contourne le typage — la confiner aux frontières (sérialisation).

#### Ch. 35 — `unsafe` & interopérabilité bas niveau

- **Objectif** : contourner le typage en connaissance de cause.
- **Contenu** : `unsafe.Pointer`, règles de conversion ; `Sizeof`/`Alignof`/`Offsetof` ; `unsafe.Slice`/`unsafe.String` ; alignement & padding ; `//go:linkname` (mention) ; **cgo** (principe, coût, `C.`), aperçu SIMD.
- **🆕 1.26** : appels **cgo ~30 % plus rapides** ; `simd/archsimd` (expérimental, `GOEXPERIMENT=simd`) ; `runtime/secret` (expérimental).
- **⚠️** : règles `unsafe.Pointer` (les 4 patterns valides), portabilité.

---

### PARTIE VI — Performance, profiling & outils

#### Ch. 36 — Tests avancés, benchmarks & fuzzing

- **Objectif** : mesurer et sécuriser correctement.
- **Contenu** : benchmarks (`testing.B`, `b.Loop`, `b.ReportAllocs`, `-benchmem`) ; bonnes pratiques (éviter l'élimination par le compilateur) ; `benchstat` (comparer A/B) ; **fuzzing** (`testing.F`, corpus) ; `Example` ; couverture.
- **🆕 1.24** : `b.Loop`. **🆕 1.26** : `b.Loop` n'empêche plus l'inlining.
- **🧪** : benchmark + `benchstat` avant/après une optimisation.

#### Ch. 37 — Profiling avec pprof

- **Objectif** : trouver les coûts réels (CPU, mémoire, blocage).
- **Contenu** : profils CPU/heap/allocs/block/mutex/goroutine ; `runtime/pprof` (programmes) ; `net/http/pprof` (services) ; `go tool pprof` (top, list, web, flame graph) ; interprétation (flat vs cum).
- **🆕 1.26** : UI web pprof en **flame graph** par défaut ; profil `goroutineleak` (`/debug/pprof/goroutineleak`).
- **Schéma** : lecture d'un flame graph (largeur = temps).

#### Ch. 38 — Traces d'exécution & Flight Recorder

- **Objectif** : comprendre le comportement temporel (scheduler, GC, latence).
- **Contenu** : `runtime/trace` ; `go test -trace` ; `go tool trace` (vues goroutines/procs/GC) ; régions & tâches utilisateur ; diagnostic de latence/contention.
- **🆕 1.25** : `runtime/trace.FlightRecorder` (anneau en mémoire, capture des dernières secondes lors d'un évènement rare).
- **🧪** : déclencher un `FlightRecorder.WriteTo` sur dépassement de latence.

#### Ch. 39 — Compilation, inlining, PGO & optimisations

- **Objectif** : comprendre ce que fait le compilateur et l'aider.
- **Contenu** : pipeline de compilation (parse → types → SSA → asm) ; inlining (budget, `-gcflags=-m`) ; escape analysis (rappel) ; **bounds check elimination** ; **PGO** (profil → recompilation guidée) ; `GOAMD64`/FMA ; DWARF5 ; build modes & flags.
- **🆕 1.21** : PGO (GA). **🆕 1.25** : DWARF5, FMA en `GOAMD64=v3+`. **🆕 1.26** : davantage d'allocations de slices sur la pile.
- **⚡** : étude de cas PGO chiffrée.

#### Ch. 40 — Méthodologie de performance

- **Objectif** : un processus rigoureux, pas des micro-optimisations au hasard.
- **Contenu** : « mesurer d'abord » ; définir un budget/SLO ; reproduire (benchmarks représentatifs) ; localiser (pprof/trace) ; corriger ; revérifier (benchstat) ; éviter les pièges (micro-bench trompeurs, GC vs latence p99, `GOMEMLIMIT`) ; checklist d'optimisation.
- **📌** : boucle mesure → hypothèse → changement → re-mesure.

---

### PARTIE VII — Projets pratiques

> Chaque projet : cahier des charges, architecture (schéma ASCII), code commenté par
> étapes, tests/benchmarks, points de vigilance, « pour aller plus loin ».

#### Projet 1 — Outil CLI

- **Couvre** : `flag`/`os.Args`, sous-commandes, lecture stdin/fichiers, configuration, **concurrence** (worker borné), sortie formatée, codes de retour, tests, build & **cross-compilation**, distribution.
- **Idée** : un utilitaire de traitement de fichiers (ex. compteur/transformateur parallèle).

#### Projet 2 — API REST

- **Couvre** : `net/http`, **routage `ServeMux` (méthodes + wildcards + `PathValue`)**, middlewares (logging, recover, CORS), **logging structuré `slog`**, base de données (`database/sql`, migrations, contexte/timeout), JSON, validation, _graceful shutdown_, tests `httptest`.
- **🆕 1.22** : routage enrichi. **🆕 1.25** : `http.CrossOriginProtection` (anti-CSRF), `slog.NewMultiHandler` ; mention `encoding/json/v2` (expérimental).

#### Projet 3 — Pipeline concurrent / worker pool

- **Couvre** : fan-out/fan-in, `select`, `context` (annulation), **backpressure** (canaux bornés), `errgroup`, _rate limiting_, arrêt propre, métriques.
- **Tests** : `-race` + **`testing/synctest`** (temps virtuel), détection de fuite via `goroutineleak`.

#### Projet 4 — Bibliothèque générique réutilisable

- **Couvre** : conception d'API, **génériques** + contraintes, doc comments + `Example` testables, tests + **benchmarks**, **fuzzing**, SemVer, compatibilité, CI (`go test/vet ./...`).
- **Idée** : structures de données génériques (ensemble, file de priorité, cache LRU).

#### Projet 5 — Service réseau TCP/RPC

- **Couvre** : package `net`, gestion d'une connexion par goroutine, **deadlines/timeouts**, protocole binaire (`encoding/binary` ou `gob`), framing, arrêt propre, robustesse, tests d'intégration.
- **Idée** : mini serveur clé-valeur en réseau avec protocole maison.

#### Projet 6 — Générateur de code (`go:generate`)

- **Couvre** : `go/ast`, `go/parser`, `go/token`, parcours d'AST, `text/template`, directive `//go:generate`, intégration `go generate ./...`.
- **🆕 1.26** : `go/ast.ParseDirective`, `BasicLit.ValueEnd`.
- **Idée** : générer des méthodes (ex. `String()` d'enum, accesseurs) à partir d'annotations.

#### Projet 7 — Profiling & debug d'un service réel (capstone)

- **Couvre** : reprendre le Projet 2 ou 3 et le **profiler de bout en bout** : pprof (CPU/heap/block/mutex/goroutine + **goroutineleak**), traces + **FlightRecorder**, benchmarks + benchstat, `-race`, **PGO**, itération d'optimisation chiffrée.
- **Livrable** : rapport « avant/après » avec graphes de flamme et gains mesurés.

---

### ANNEXES

- **A — Glossaire** : termes runtime/langage (goroutine, GC, itab, span, P/M/G, escape, etc.).
- **B — Antisèche des commandes `go`** : `build/run/test/vet/doc/fix/tool/work/mod/generate`, flags utiles.
- **C — Carte des nouveautés Go 1.21 → 1.26** : tableau version × fonctionnalité (langage, runtime, stdlib, outils).
- **D — Algorithmes & structures de données en Go** : implémentations idiomatiques commentées (tri, graphes, structures génériques).
- **E — Démonstrations techniques & benchmarks** : micro-études chiffrées (alloc pile/tas, mutex vs atomic, interface vs générique, GC tuning).
- **F — Idiomes & style** : condensé « Effective Go » + erreurs fréquentes des nouveaux venus.
- **G — Ressources** : doc officielle, `pkg.go.dev/std`, Tour of Go, Go by Example, blogs runtime, proposals.

---

## 6. Charte qualité (definition of done par chapitre)

Un chapitre est « terminé » quand :

- [ ] Tous les exemples de code **compilent** et leurs tests passent (`go test ./...` dans `code/`).
- [ ] `go vet ./...` est propre.
- [ ] Les schémas sont en **ASCII pur** et alignés en police à chasse fixe.
- [ ] Les nouveautés mentionnées sont **vérifiées** sur la doc 1.26 officielle.
- [ ] Présence des encarts pertinents (`🆕`, `⚠️`, `⚡`, `🧪`, `📌`).
- [ ] Renvois `🔁` cohérents avec les autres chapitres.
- [ ] Relecture orthographique FR (accents/diacritiques corrects).

---

## 7. Volume & phasage de production

- **Échelle** : ~42 chapitres + 7 projets + 7 annexes. Estimation indicative 300-450 pages.
- **Ordre de production conseillé** (pour livrer de la valeur tôt) :
  1. **Vague 1 — Apprentissage** : Parties 0, I, II (ch. 0-18) + Projets 1 et 2.
  2. **Vague 2 — Concurrence** : Partie III (ch. 19-23) + Projet 3.
  3. **Vague 3 — Internals** : Parties IV, V (ch. 24-35).
  4. **Vague 4 — Performance** : Partie VI (ch. 36-40) + Projets 4-7.
  5. **Vague 5 — Annexes & passes de cohérence/relecture**.
- Mise en place dès le départ du **module `code/` + CI** pour garantir des exemples toujours valides.

---

## 8. Prochaines étapes

1. [x] **Valider/ajuster** ce plan (ordre des chapitres, granularité, projets).
2. [x] Mettre en place le **squelette du dépôt** (`chapitres/`, `code/go.mod`, `projets/`, `annexes/`, `SOMMAIRE.md`, `README.md`, `.gitignore`).
3. [x] Établir un **gabarit de chapitre** réutilisable (`chapitres/_gabarit.md`).
4. [~] **Rédaction des chapitres** — **ch. 0 à 40 rédigés** : **Parties I, II, III, IV, V et VI terminées** (+ exemples `code/ch01-hello/`, `ch02-structure/`, `ch03-basics/`, `ch04-controlflow/`, `ch05-functions/`, `ch06-slices/`, `ch07-maps-strings/`, `ch08-structs/`, `ch09-interfaces/`, `ch10-errors/`, `ch11-generics/`, `ch12-packages/`, `ch13-tests/`, `ch14-switch/`, `ch15-closures/`, `ch16-defer/`, `ch17-panic-recover/`, `ch18-iterators/`, `ch19-goroutines/`, `ch20-channels-select/`, `ch21-synchronisation/`, `ch22-context/`, `ch23-patterns-concurrence/`, `ch24-runtime-bootstrap/`, `ch25-modele-memoire/`, `ch26-allocation-escape/`, `ch27-garbage-collector/`, `ch28-ordonnanceur-gmp/`, `ch29-observabilite-runtime/`, `ch30-slices-profondeur/`, `ch31-strings-profondeur/`, `ch32-maps-hachage/`, `ch33-interfaces-profondeur/`, `ch34-reflexion/`, `ch35-unsafe-cgo/`, `ch36-benchmarks-fuzzing/`, `ch37-profiling-pprof/`, `ch38-traces-flightrecorder/`, `ch39-compilation-pgo/`, `ch40-methodologie/`). Les Parties III (Vague 2), IV-V (Vague 3) et VI (Vague 4) ont été rédigées en avance ; restent les **projets** et les **annexes**.
5. [~] Rédiger les projets — **Projets 1 (CLI `txtkit`), 2 (API REST `tasksd`), 3 (pipeline concurrent `pipe`) et 4 (bibliothèque générique `gends`) rédigés** (`projets/1-cli/` module `example.com/txtkit`, `projets/2-api-rest/` module `example.com/tasksapi`, `projets/3-pipeline/` module `example.com/pipeline`, `projets/4-lib-generique/` module `example.com/gends`, `go test -race`/`vet`/`gofmt` propres) ; restent les **Projets 5-7** (Vague 4) ; puis la **Vague 5 — Annexes** (A → G) et les passes de cohérence/relecture.

---

## 9. État d'avancement

> Légende : ✅ rédigé & validé (`go test`/`vet` propres) · 🚧 en cours · ⬜ à faire.

### Chapitres

| Partie                  | Chapitres          | État      |
| ----------------------- | ------------------ | --------- |
| 0 — Introduction        | Ch. 0 ✅, Ch. 1 ✅ | **2/2**   |
| I — Fondamentaux        | Ch. 2-13 ✅        | **12/12** |
| II — Mécanismes avancés | Ch. 14-18 ✅       | **5/5**   |
| III — Concurrence       | Ch. 19-23 ✅       | **5/5**   |
| IV — Runtime & mémoire  | Ch. 24-29 ✅       | **6/6**   |
| V — Internals           | Ch. 30-35 ✅       | **6/6**   |
| VI — Performance        | Ch. 36-40 ✅       | **5/5**   |
| VII — Projets           | Projets 1-4 ✅, 5 → 7 | **4/7**   |
| Annexes                 | A → G              | ⬜ 0/7    |

### Infrastructure

- ✅ Squelette du dépôt, `README.md`, `SOMMAIRE.md`, `.gitignore`.
- ✅ Module `code/` (`example.com/gobook`, `go 1.26`) — `go test ./...` & `go vet ./...` propres.
- ✅ Gabarit de chapitre (`chapitres/_gabarit.md`).
- ✅ Exemples compilables et testés : `code/ch01-hello/` (`greet`), `code/ch02-structure/`
  (multi-packages `main` + `greeting`), `code/ch03-basics/` (zero values, conversions,
  `iota`/`ByteSize`, conversion sûre, `new(expr)`), `code/ch04-controlflow/` (FizzBuzz,
  `classify`, `break` étiqueté), `code/ch05-functions/` (retours multiples, variadiques,
  fonctions-valeurs, valeur vs pointeur, functional options), `code/ch06-slices/`
  (`reverseInts`, `filter`, `chunk` + 3-index anti-aliasing), `code/ch07-maps-strings/`
  (`wordCount`, `uniqueSorted` via set + `slices.Sorted(maps.Keys)`, `reverseString` &
  `truncate` rune-aware), `code/ch08-structs/` (`Point`/`Rectangle` récepteur valeur,
  `Account`/`AuditedAccount` récepteur pointeur + embedding/override, `Padded`/`Packed` padding),
  `code/ch09-interfaces/` (`Shape`/`Circle`/`Rect` satisfaction implicite + Stringer, `classify`
  type switch, `ValidationError`/`error`, piège interface-nil), `code/ch10-errors/`
  (`ErrEmptyKey` sentinelle, `ParseError`+Unwrap, `parseConfig` via `errors.Join`, `errors.AsType`),
  `code/ch11-generics/` (`Number`/`~` + `Sum`/`Max`/`Map`/`Filter`/`Index`, `Stack[T]`, alias
  générique `Set[T]` (1.24), contrainte auto-référentielle `Adder[A Adder[A]]` (1.26),
  benchmark générique vs interface), `code/ch12-packages/` (package `internal/money`,
  `fmt.Stringer`, test « boîte noire » + `Example`), `code/ch13-tests/` (`Slugify` table-driven +
  sous-tests + `t.Helper`/`Example`, `SaveLines` via `t.TempDir`, `t.Cleanup` LIFO, `T.Attr`/
  `T.Output`/`T.ArtifactDir`, `BenchmarkSlugify` + `FuzzSlugify`), `code/ch14-switch/`
  (`grade` tagless, `dayKind` cas multiples, `capabilities` via `fallthrough`, `describe` type
  switch multi-types, `levelFromString`/`levelFromMap`/`levelFromInt` + benchmark switch vs map),
  `code/ch15-closures/` (`counter` à état, `makeAdders` portée par itération 1.22, décorateur
  `logged`, `memoize`, middleware `chain`/`tagged`/`upper`, functional options `Server`/`Option`),
  `code/ch16-defer/` (`lifoOrder`, `evalContrast` argument vs closure, `doubleViaDefer` retour nommé,
  `processScoped` vs `processDeferInLoop`, `trace`, `withLock` + benchmarks open-coded vs en boucle),
  `code/ch17-panic-recover/` (`safeCall` recover→erreur, `divide` panique runtime, `mustPositive`
  pattern Must, `validate` recover sélectif + re-panic, `recoverMiddleware` frontière HTTP),
  `code/ch18-iterators/` (`Count`/`Naturals`, combinateurs `Map`/`Filter`/`Take`/`Enumerate`, `Zip`
  via `iter.Pull`, `slices.Values`/`Sorted` + benchmark itérateur vs slice matérialisée),
  `code/ch19-goroutines/` (`parallelMap` générique via `WaitGroup` + écriture par index, `tickUntilStop`
  arrêt propre par canal fermé, `runtime.NumGoroutine`/`GOMAXPROCS`), `code/ch20-channels-select/`
  (`gen` générateur + `range`/`close`, `fanIn` fusion, `trySend` via `select`/`default`,
  `recvWithTimeout` via `time.After`, directions `<-chan`/`chan<-`), `code/ch21-synchronisation/`
  (`SafeCounter` Mutex vs `AtomicCounter` atomic, `runConcurrently` via `WaitGroup.Go`, `OnceValue`,
  `Registry` RWMutex, `Config` via `atomic.Pointer[T]`, `joinInts` via `sync.Pool` + benchmarks
  atomic/mutex/RWMutex), `code/ch22-context/` (`sumUntilCancel` surveillant `ctx.Done()`,
  `WithCancelCause`/`Cause`, `WithTimeout`, valeur de contexte à clé de type non exporté),
  `code/ch23-patterns-concurrence/` (`source`/`stage` pipeline en flux, `workerPool` parallélisme borné,
  `Group` errgroup maison en stdlib, `rateLimited` via `time.Ticker`, tests `testing/synctest` à horloge
  virtuelle), `code/ch24-runtime-bootstrap/` (ordre d'init dépendances→`init()`, `CurrentRuntime` via
  `runtime.Version`/`NumCPU`/`GOMAXPROCS`, démonstration `GODEBUG=inittrace`), `code/ch25-modele-memoire/`
  (publication sûre `PublishViaChannel`, `sync.Once` contre le double-checked locking, `atomic.Pointer`,
  tests `-race` verts), `code/ch26-allocation-escape/` (`sumLocalArray`/`sumSmallSlice` 0 alloc sur pile vs
  `NewPoint`/`LeakSlice` échappés, `concatPrealloc` vs `concatNoPrealloc`, assertions `testing.AllocsPerRun`
  + benchmarks `-benchmem`), `code/ch27-garbage-collector/` (cache à références faibles `weak.Pointer`,
  `runtime.AddCleanup`, `WithGCPercent`/`CurrentMemoryLimit` via `debug`), `code/ch28-ordonnanceur-gmp/`
  (`parallelSum` fan-out, `WithGOMAXPROCS`, `busyWork`+`runtime.Gosched`, démo 205 ms→32 ms),
  `code/ch29-observabilite-runtime/` (`ReadSnapshot` via `runtime/metrics` dont `/sched/*` 1.26,
  `ReadBuildInfo`, compteur+jauge `expvar`, comparaison `ReadMemStats`), `code/ch30-slices-profondeur/`
  (`CapGrowth` suite des cap, `SubSliceCap` 3 indices, aliasing vs `SafeAppend`, `FilterInPlace`
  zéro-alloc, `TrimRetention` via `slices.Clone`, bench in-place 0 alloc vs new-slice 10 allocs),
  `code/ch31-strings-profondeur/` (`ByteVsRune`/`RuneWidths` UTF-8, `JoinCSV` via `strings.Builder`,
  `ToUpperASCII` 2 copies, `Intern`/`CountDistinct` via `unique`, bench concat `+` 117 µs/499 allocs vs
  Builder 3,9 µs/12 allocs), `code/ch32-maps-hachage/` (`WordCount` préalloué, `IterationOrders`
  randomisation, `SafeCounter` map+Mutex testé `-race`, bench prealloc 20→5 allocs), `code/ch33-interfaces-profondeur/`
  (`Shape`/`Circle`/`Rectangle` dispatch, `FailBuggy`/`FailCorrect` piège interface-nil, `BoxValue`
  boxing 0/1 alloc, `AsCircle` via `reflect.TypeAssert`, bench dispatch interface 4562 ns vs concret 1716 ns),
  `code/ch34-reflexion/` (`InspectFields` via `Type.Fields()` 1.26, `FillDefaults` écriture `CanSet`+tags,
  `CallMethod` appel dynamique, `Ins()`/`Outs()` 1.26, bench reflect 355 ns/5 allocs vs direct 3,2 ns),
  `code/ch35-unsafe-cgo/` (padding `Padded`=24/`Packed`=16, `Sizeof`/`Alignof`/`Offsetof`, `BytesToString`/
  `StringToBytes` zéro-copie via `unsafe.String`/`Slice`/`SliceData`, `SecondElem` via `unsafe.Add`, bench
  `string([]byte)` 21 ns/1 alloc vs `unsafe.String` 3,2 ns/0 alloc), `code/ch36-benchmarks-fuzzing/`
  (`FormatThousands` via `strings.Builder` vs `formatNaive`, `BenchmarkBuilder`/`Naive` + sous-benchmarks
  `b.Run` par taille, `sink` anti-DCE, `FuzzFormatThousands` invariant round-trip, bench Builder 69 ns/2 allocs
  vs Naïve 107 ns/3 allocs, benchstat A/B réel −34,9 % p=0,000), `code/ch37-profiling-pprof/` (`HotCompute`/
  `collatzSteps` cible CPU propre, `wordFrequencies` cible tas, `CaptureCPUProfile`/`CaptureHeapProfile` via
  `runtime/pprof`, en-tête gzip vérifié, 6 profils par défaut, `go tool pprof -top`/`-list` flat vs cum,
  `net/http/pprof` en prose), `code/ch38-traces-flightrecorder/` (`CaptureTrace` via `runtime/trace`,
  `processBatch` tâches/régions/`Log`, `MonitorLatency` via **FlightRecorder** 1.25 `WriteTo` sur dépassement
  de latence, en-tête `"go 1."` vérifié), `code/ch39-compilation-pgo/` (`add`/`AddTwice` inlining `-m`,
  `SumRange`/`SumGather`/`SumHinted` BCE `Found IsInBounds`, `TotalArea` `//go:noinline` cible PGO,
  `default.pgo` + `-pgo=auto` + `-d=pgodebug=2` dévirtualisation, bench BCE 504,7 ns vs 595,6 ns),
  `code/ch40-methodologie/` (`DedupNaive` O(n²) `slices.Contains` vs `Dedup` map O(n), boucle mesure→profil→
  correction→re-mesure, bench 2472 µs→77 µs **32×** mais +1410 % mémoire, compromis latence/mémoire selon SLO).
- ✅ Nouveautés **vérifiées sur la toolchain 1.26.4** : `new(expr)` (type inféré),
  `min`/`max`/`clear`, débordement silencieux vs erreur de compilation sur constante ;
  `for range N` et **portée par itération** de la variable de boucle (1.22) ; itération de map
  **randomisée** + `slices.Sorted(maps.Keys)`, `clear` laisse la map non-nil, octets vs runes
  UTF-8 (`"café"` = 5 octets / 4 runes, `🚀` = 4 octets) ; récepteur valeur vs pointeur
  (auto-adressage), embedding **sans dispatch dynamique**, padding `Padded`=24/`Packed`=16
  octets (64 bits), `==` interdit sur struct à champ slice (erreur compile), `structs.HostLayout` ;
  satisfaction implicite, `type switch`, `any` ≡ `interface{}`, **piège interface-nil** (pointeur
  nil typé → interface non-nil), method set récepteur pointeur (`T` valeur ne satisfait pas) ;
  `errors.AsType[E]` (1.26, `func AsType[E error](err error) (E, bool)`), `errors.Join`/`Is`/`As`
  à travers la chaîne `%w`, `fmt.Errorf` sans `%w` = 1 alloc (comme `errors.New`) vs 2 avec `%w` ;
  génériques — contrainte `~` (capte les types définis), `comparable`, type générique,
  **alias de type générique** `type Set[T comparable] = ...` (1.24), **contrainte
  auto-référentielle** `type Adder[A Adder[A]] interface{ Add(A) A }` (1.26), `cmp.Or`/`cmp.Compare`/
  `slices.SortFunc`, générique ~4-6× plus rapide qu'une interface (dispatch évité, 0 alloc) ;
  modules — barrière **`internal/`** (« use of internal package not allowed »), directives `tool`
  (1.24) et `ignore` (1.25), `go doc -http` (1.25) ; tests — `// Output:` d'`Example` vérifié,
  `t.Cleanup` **LIFO** observé, `T.Attr`/`T.Output` (1.25) et **`T.ArtifactDir`** (1.26, non vide
  sans `-artifacts`) + flag `-artifacts`, `FuzzSlugify` (340 k exécutions, invariants tenus) ;
  `switch` — `fallthrough` en cascade (sans réévaluer la condition), variable d'un `case`
  multi-types reste de type interface, erreurs compile (`duplicate case`, `fallthrough` hors
  place / dernier cas / type switch), **jump table** pour un switch entier dense ≥ 8 cas
  (asm vérifié : `CMP $7, R0` + `JMP (R27)`), `switch` chaînes ~4-5× plus rapide qu'une `map`
  (0 alloc) ; closures — **capture par référence** (`counter` → 1 2 3), **portée par itération**
  1.22 vérifiée (`[0 1 2]` en `range` ET 3-clauses sous go 1.26, `[3 3 3]` sous go 1.21), variable
  capturée **`moved to heap`** (escape) ; `defer` — **LIFO** (`2 1 0`), **arguments évalués à
  l'enregistrement** vs closure lue à l'exécution, retour **nommé** modifié par un defer (`42`),
  piège `defer` en boucle (Close repoussés en fin de fonction), **open-coded defer** ~3,24 ns ≈ appel
  direct vs `defer` en boucle ~16,6 ns/defer (0 alloc) ; `panic`/`recover` — `recover` rattrape dans
  un `defer` (y compris paniques runtime), **re-panique même valeur** → `panic: … [recovered,
repanicked]` (1.25) vs valeur différente → `[recovered]` + chaîne, **panique de goroutine fatale**
  (non rattrapable depuis `main`) ; itérateurs (1.23) — `iter.Seq`/`Seq2`, **range-over-func**, arrêt
  anticipé (`break` propage `yield`=false), composition **paresseuse** `Map`/`Filter`/`Take`
  (`[0 4 16]`) sur source **infinie**, `iter.Pull` (`Zip`, goroutine + `stop()` obligatoire),
  `slices.Values`/`Sorted`/`Collect` + `maps.Keys`, itérateur **0 alloc / ~2,0 µs** vs slice échappée
  **8192 B / 1 alloc / ~2,9 µs** ; concurrence (Partie III) — goroutine **~2049 octets de pile**
  (`StackInuse`, 100 k goroutines = +205 Mo), `runtime.NumGoroutine` 1 → 100001, **profil
  `goroutineleak`** gated `GOEXPERIMENT=goroutineleakprofile` (nil sinon, pointe la ligne `<-ch`
  exacte), métriques **1.26** `/sched/goroutines-created` + ventilation `/sched/goroutines/{running,
runnable,waiting}` + `/sched/threads/total` ; canaux — paniques `send on closed channel` /
  `close of closed channel` / `close of nil channel`, réception post-`close` draine puis zéro/`false`,
  bench canal non bufferisé **185 ns** vs bufferisé **43 ns** ; `sync` — **`WaitGroup.Go`** (1.25) +
  analyzer **`go vet waitgroup`** (« Add called from inside new goroutine »), `OnceValue` exécuté
  **1 fois** sous 100 appels, bench **atomic 52 ns / mutex 138 ns / RWMutex lecture 64 ns / Mutex
  lecture 120 ns** (0 alloc) ; `context` — `WithCancelCause`/`Cause` (`Err()`=Canceled mais
  `Cause()`=erreur métier), `WithTimeout` → `DeadlineExceeded`, clé de contexte à type non exporté ;
  **`testing/synctest`** (GA 1.25) — `synctest.Test`/`Wait`, horloge **virtuelle exacte** (5 × 100 ms =
  500 ms en 0 s réel), tout `go test`/`-race`/`vet` propres sur ch19-23 ; runtime & mémoire (Partie IV) —
  **bootstrap** `_rt0_<arch>_<os>` → `rt0_go` (`g0`/`m0`) → `schedinit` → `runtime.main` → `main.main`,
  ordre d'init **dépendances→`init()`** observé (`base` avant `derived`, vars avant `init()`),
  **`GODEBUG=inittrace=1`** format `init <pkg> @t ms, clock ms clock, N bytes, M allocs` (avec `init main`) ;
  **modèle mémoire** — publication par canal/`sync.Once`/`atomic.Pointer` **`-race` verte**, double-checked
  locking buggé, atomics séquentiellement cohérents depuis 1.19 ; **escape analysis** `-gcflags=-m`
  (`moved to heap`, `make([]int, 8) does not escape` = backing **sur pile** 1.25/1.26, `escapes to heap` sur
  interface), `testing.AllocsPerRun` = 0/0/1/1 alloc (`sumLocalArray`/`sumSmallSlice`/`NewPoint`/`LeakSlice`),
  préallocation **9→1 alloc, 25152→8192 B, 2690→1116 ns** ; **GC** — `weak.Make`/`weak.Pointer[T].Value()`
  (nil après GC) + `runtime.AddCleanup` (cleanup exécuté, distinct des finalizers), `GODEBUG=gctrace=1`
  format complet (`avant->après->vivant MB`, `goal`, 2 STW + marquage concurrent), `checkfinalizers=1` (1.25,
  `queue: N finalizers + M cleanups`), **Green Tea GC par défaut 1.26** (`GOEXPERIMENT=nogreenteagc` accepté),
  `debug.SetGCPercent`/`SetMemoryLimit(-1)`=MaxInt64 ; **ordonnanceur** — `GODEBUG=schedtrace=N` format
  (`gomaxprocs`/`idleprocs`/`threads`/`runqueue` globale + `[ LRQ par P ]`), `threads`>`GOMAXPROCS` sur
  syscall, `runtime.SetDefaultGOMAXPROCS()` (1.25) présent, démo CPU 205 ms (P=1) → 32 ms (P=8) ;
  **observabilité** — `runtime/metrics` **112 descripteurs**, `NumGoroutine`=1 (user) vs
  `/sched/goroutines`=6-7 (toutes, système comprises), `/sched/goroutines-created` cumulatif + ventilation
  `running`/`runnable`/`waiting` (1.26), `ReadBuildInfo`=`go1.26.4`, `expvar` (`/debug/vars`) publié/lu ;
  internals des types (Partie V) — **slices** header **24 o** (3 mots), croissance `append` observée
  `4 8 16 32 64 128 256 512 848 1280 1792 2560` (double < 256 puis ≈1,25× arrondi size class ; `growslice`
  réel 0→1 en `//go:noinline`, cap 4 d'entrée = `append` inliné), `s[i:j:k]` borne `cap=k-i`, aliasing
  (`append` écrase le parent) vs `[:n:n]`, `FilterInPlace` **0 alloc** vs new-slice **10 allocs/8184 B** ;
  **strings** header **16 o** (2 mots), immuable, `"héllo, 日本"`=14 o/9 runes, `range` décode par rune
  (index d'octet sauté), conversion `string([]byte)`/`[]byte(string)` qui échappe **1 alloc** vs lookup
  map/comparaison/`range` **0 alloc** (no-copy), `strings.Builder` concat **+** 117 µs/499 allocs → **3,9 µs/12
  allocs**, **`unique.Make`** (1.23) handles `==` pour contenus égaux, `Handle[string]`=**8 o** ; **maps**
  **Swiss Tables** (1.24, groupes de 8 slots + mot de contrôle, recherche `h2` parallèle SWAR, ~87,5 % de
  charge, croissance incrémentale par annuaire), itération **randomisée** (parcours différents intra/inter
  exécution), `make(map, n)` **20→5 allocs / 45→12 µs**, `fatal error: concurrent map writes` (non
  rattrapable) → `map`+`Mutex` ; **interfaces** `eface`/`iface` = **2 mots / 16 o**, dispatch monomorphe
  ≈ gratuit mais **inlining perdu** (interface 4562 ns vs concret 1716 ns), **boxing** `int` 0..255 caché
  **0 alloc** sinon **1 alloc**, **piège interface-nil** (`FailBuggy(true)==nil` → false), **`reflect.TypeAssert[T]`**
  (1.25) sans re-boxing ; **reflect** itérateurs **1.26** `Type.Fields/Methods`, `Value.Fields/Methods`,
  `Method.Type.Ins()`/`Outs()` (récepteur = `in[0]`), écriture via pointeur+`CanSet`, appel dynamique
  `MethodByName`/`Call`, coût réflexion **355 ns/5 allocs vs direct 3,2 ns/0** (~110×) ; **unsafe** padding
  `Padded`=**24** vs `Packed`=**16** (réordonner économise 8 o), `Offsetof`(0,8,16), `Alignof`
  int64=8/int32=4/byte=1, **`unsafe.String`/`Slice`/`SliceData`/`Add`** zéro-copie (backing partagé,
  `string([]byte)` 21 ns/1 alloc → **3,2 ns/0**), **cgo ~30 % plus rapide** (1.26, notes de version),
  **`simd/archsimd`** (`GOEXPERIMENT=simd`, AMD64) et **`runtime/secret`** (`GOEXPERIMENT=runtimesecret`)
  expérimentaux off par défaut (constantes `goexperiment` `SIMD`/`RuntimeSecret` = false) ; performance &
  outils (Partie VI) — **benchmarks** `b.Loop() bool` (1.24), `-benchmem` (`B/op`/`allocs/op`), **anti-DCE**
  par `sink` de package (sans lui, appel éliminé ~0,3 ns), sous-benchmarks `b.Run`, **`benchstat`** installé
  (`golang.org/x/perf`) comparaison A/B réelle `106,9n→69,6n −34,89 % p=0,000 n=10` (exige `Benchmark` en
  préfixe + `-count`≥10 + `p<0,05`), **fuzzing** `testing.F`/`f.Add`/`f.Fuzz` 1,4 M exéc/6 s sur 8 workers,
  corpus cache (`$GOCACHE/fuzz`) vs crashers versionnés (`testdata/fuzz/`) ; **pprof** — **6 profils par
  défaut** (`allocs`/`block`/`goroutine`/`heap`/`mutex`/`threadcreate`), `block`/`mutex` à armer
  (`SetBlockProfileRate`/`SetMutexProfileFraction`), `goroutineleak` **absent** par défaut (`Lookup`=nil,
  gated `GOEXPERIMENT=goroutineleakprofile`), `StartCPUProfile`/`WriteHeapProfile`/`Lookup`, profil =
  protobuf **gzip** (`1f 8b`), `go tool pprof -top`/`-list` **flat** (dans la fonction) vs **cum** (avec
  appelés), tas `alloc_space` (pression GC) vs `inuse_space` (rétention), artefact macOS `runtime.kevent`,
  **flame graph par défaut 1.26** ; **trace** — en-tête `"go 1.26 trace…"`, `trace.Start`/`Stop`,
  tâches `NewTask`/régions `WithRegion`/`StartRegion`/`Log`, `go test -trace`, lecture via `go tool trace`
  (vues proc/goroutine/scheduler/blocking/MMU), **`FlightRecorder`** (1.25) `NewFlightRecorder(cfg{MinAge,
  MaxBytes})`+`Start()`/`Stop()`/`WriteTo()`/`Enabled()` (anneau mémoire, `WriteTo` quasi instantané sur
  évènement rare, `Enabled` false→true) ; **compilateur** — inlining `-gcflags=-m` (`can inline`/`inlining
  call to`, budget ~80, `//go:noinline`), escape (`does not escape`/`escapes to heap`), **BCE**
  `-d=ssa/check_bce` (`Found IsInBounds` sur index externe, rien en `range`/témoin `_=xs[3]`, bench 504,7 ns
  vs 595,6 ns), **PGO** (GA 1.21) `default.pgo`+`-pgo=auto` (défaut) à la racine du `main`, `-d=pgodebug=1/2`
  (`hot-callsite-thres-from-CDF`, `PGO devirtualize considering call s.Area()`), dévirtualisation statique
  mono-type sinon profil, gains typiques 2-14 %, **`GOAMD64=v3+`** FMA (x86 ; machine arm64 → n/a),
  **DWARF5** défaut 1.25 ; **méthodologie** — boucle mesure→hypothèse→changement→re-mesure (une hypothèse à
  la fois), SLO d'abord, algorithme≫allocations≫micro-opt, `slices.Contains` O(n²)→map O(n) **32×** mais
  benchstat révèle **+1410 % mémoire** (compromis selon SLO), `GOMEMLIMIT` levier p99.
- ⬜ CI (GitHub Actions) lançant `go test ./...` + `go vet ./...` + `gofmt -l`.

**Prochaine action concrète** : **Parties I à VI terminées (Ch. 0 à 40)** et **Projets 1 (Outil CLI),
2 (API REST) et 3 (pipeline concurrent) rédigés**. Le **Projet 1** vit dans `projets/1-cli/` — module autonome `example.com/txtkit`,
outil `txtkit` à sous-commandes (`count` façon `wc` rune par rune + `freq` top-N), lecture fichiers/stdin via
`sourcesFrom`, **worker pool générique borné** `mapBounded[T,R]` (sémaphore à canal + `WaitGroup.Go` 1.25,
résultats ordonnés, `-race` propre), configuration en couches (défaut `GOMAXPROCS` < `TXTKIT_WORKERS` <
flag `-j`), patron testable `Run(args, in, out, err) int`, codes de retour 0/1/2, sortie `text/tabwriter`,
tests table-driven, `Makefile` de cross-compilation. Le **Projet 2** vit dans `projets/2-api-rest/` —
module `example.com/tasksapi`, API REST `tasksd` **stdlib pure** : routage `ServeMux` 1.22 (méthode + `{id}`
via `PathValue`, `405`/`404` natifs), CRUD JSON `/api/tasks` avec validation (`400/404/405/422/500`),
chaîne de middlewares (`recoverPanic` → `requestID` → `logging` → CSRF), **logging structuré `slog`**
(`NewMultiHandler` 1.25 : texte stderr + JSON d'audit), **protection CSRF** `http.CrossOriginProtection`
(1.25), persistance derrière l'**interface `store.Store`** (`MemStore` `RWMutex` par défaut + `SQLStore`
`database/sql` à migrations `//go:embed`), **arrêt propre** `signal.NotifyContext` + `Server.Shutdown`,
décodage borné/strict (`MaxBytesReader`, `DisallowUnknownFields`), tests `httptest` de bout en bout, et
usage de `new(expr)` / `strings.SplitSeq` (1.26/1.24). Le **Projet 3** vit dans `projets/3-pipeline/` —
module `example.com/pipeline`, **pipeline concurrent générique** `Process[I,O](ctx, iter.Seq[I], Stage, Config)`
appliqué à l'outil `pipe` (SHA-256 de fichiers en parallèle) : **fan-out/fan-in** (feeder interne + N workers),
**pression arrière** (canaux bornés), **annulation `context`** (lecture par blocs annulable), **`errgroup`**
(première erreur ⇒ annulation globale), **limitation de débit** (`RateLimiter` sur `time.Ticker`), **métriques**
atomiques (pic de concurrence par CAS), source paresseuse `iter.Seq`, et tests **`testing/synctest`** (temps
virtuel + détection de fuite de goroutines) sous `-race`. Seule dépendance externe : `golang.org/x/sync/errgroup`.
Le **Projet 4** vit dans `projets/4-lib-generique/` — module `example.com/gends`, **bibliothèque générique
réutilisable** en trois packages (`set` `Set[T comparable]` ensembliste, `pqueue` `Queue[T any]` tas binaire à
comparateur + `NewOrdered[cmp.Ordered]`, `lru` `Cache[K comparable,V]` borné sur `container/list`) : conception
d'API (type zéro explicite, **contraintes minimales** — `set.Sorted` ajoute `cmp.Ordered` en fonction libre),
**`Example` testables** (godoc + vérifiés), tests table-driven, **benchmarks** `for b.Loop()` (1.24) `-benchmem`,
**fuzzing** du LRU contre un modèle de référence, source `iter.Seq`, et discipline **SemVer/compat + CI**
documentée. Stdlib pure. Restent : **Projets 5-7** (Vague 4, qui réinvestissent la Partie VI : pprof,
traces/FlightRecorder, PGO), puis la **Vague 5 — Annexes** (A → G). Tout le module `code/` (ch01 → ch40)
reste vert (`go build`/`vet`/`test ./...` sur **40 packages**, `gofmt -l` vide), et `projets/1-cli/`,
`projets/2-api-rest/`, `projets/3-pipeline/` comme `projets/4-lib-generique/` sont `go test -race`/`vet`/`gofmt` propres.
