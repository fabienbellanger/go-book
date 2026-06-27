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
- **🔁 Internals** : maps → ch. 34, strings → ch. 33.

#### Ch. 8 — Structs, méthodes & composition

- **Objectif** : modéliser des données et leur comportement.
- **Contenu** : `struct`, champs, tags ; littéraux & champs nommés ; méthodes ; **récepteur valeur vs pointeur** (règles de choix) ; **embedding** (composition, promotion de champs/méthodes) ; structs vides, alignement/padding (teaser ch. 37).
- **Schéma** : layout mémoire d'un struct avec padding.
- **⚠️ Pièges** : copie de gros structs, récepteurs mixtes.

#### Ch. 9 — Interfaces (fondamentaux)

- **Objectif** : abstraction par le comportement.
- **Contenu** : déclaration, **satisfaction implicite** ; `any` (`interface{}`) ; type assertions & `comma-ok` ; **type switch** ; interfaces idiomatiques (`Stringer`, `io.Reader`/`Writer`, `error`) ; petites interfaces & acceptation large/retour concret.
- **Schéma** : valeur d'interface = (type, valeur) — teaser de `iface`/`eface`.
- **🔁 Internals** : itab, dispatch → ch. 35.
- **⚠️ Pièges** : interface nil vs pointeur nil contenu.

#### Ch. 10 — Gestion des erreurs

- **Objectif** : le modèle d'erreur Go, idiomatique et robuste.
- **Contenu** : type `error` ; `errors.New`, `fmt.Errorf` + `%w` (wrapping) ; chaînes d'erreurs ; `errors.Is` / `errors.As` ; erreurs sentinelles vs types d'erreur ; quand `panic` n'est pas une erreur ; `defer` (intro, détail ch. 17).
- **🆕 1.26** : `errors.AsType[E]` (variante générique, typée et plus rapide que `As`).
- **🆕 1.26** : `fmt.Errorf("x")` alloue autant que `errors.New`.
- **📌 À retenir** : erreurs = valeurs ; on les enrichit, on ne les masque pas.

#### Ch. 11 — Généricité : types paramétrés

- **Objectif** : polymorphisme à la compilation, sans surcoût d'interface.
- **Contenu** : paramètres de type, contraintes, `comparable`, `~` (underlying), inférence ; fonctions et types génériques ; packages `slices`, `maps`, `cmp` ; **quand NE PAS** utiliser les génériques ; instanciation (renvoi ch. 41 pour la stratégie GC-shape/monomorphisation).
- **🆕 1.21** : `slices`/`maps`/`cmp`. **🆕 1.24** : alias de type génériques. **🆕 1.26** : contraintes auto-référentielles (`type Adder[A Adder[A]] interface{ Add(A) A }`).
- **⚡ Perf** : génériques vs interfaces vs duplication.

#### Ch. 12 — Packages, modules & organisation du code

- **Objectif** : structurer, versionner, distribuer du code.
- **Contenu** : `go.mod`/`go.sum`, SemVer, `go get`/`go mod tidy` ; packages `internal/` ; visibilité ; workspaces `go work` ; dépendances outils (`go tool`) ; documentation (doc comments, `Example` testables) ; mise en page d'un projet (layout pragmatique).
- **🆕 1.24** : dépendances outils dans `go.mod`. **🆕 1.25** : directive `ignore`, `go doc -http`.

#### Ch. 13 — Tests & outillage de base

- **Objectif** : la culture du test, intégrée au langage.
- **Contenu** : `testing`, `go test` ; tests **table-driven** ; `t.Run`/sous-tests ; helpers `t.Helper()` ; `Example` comme doc exécutable ; `t.TempDir`, `t.Cleanup` ; couverture (`-cover`, `-coverprofile`) ; `go vet` & analyzers ; teaser benchmarks/fuzzing (détail ch. 38).
- **🆕 1.25** : `T.Attr`/`T.Output`, analyzers `waitgroup`/`hostport`. **🆕 1.26** : `T.ArtifactDir` + `-artifacts`.
- **🧪** : premier test table-driven complet.

---

### PARTIE II — Mécanismes avancés du langage

#### Ch. 14 — `switch` & sélection de cas (en profondeur)

- **Objectif** : exploiter toute la puissance de `switch`.
- **Contenu** : switch d'expression, sans condition (= `if/else if`), `fallthrough`, cas multiples, **type switch** avancé, switch sur `init; cond` ; ce que le compilateur génère (jump table vs comparaisons en cascade) ; `select` annoncé (ch. 21).
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
4. [~] Lancer la **rédaction de la Vague 1** — **ch. 0 à 3 rédigés** (+ exemples `code/ch01-hello/`, `ch02-structure/`, `ch03-basics/`). Suite : ch. 4 → 13.
5. [ ] Continuer la Vague 1 (Parties I et II), puis Projets 1 et 2.

---

## 9. État d'avancement

> Légende : ✅ rédigé & validé (`go test`/`vet` propres) · 🚧 en cours · ⬜ à faire.

### Chapitres

| Partie                  | Chapitres            | État    |
| ----------------------- | -------------------- | ------- |
| 0 — Introduction        | Ch. 0 ✅, Ch. 1 ✅   | **2/2** |
| I — Fondamentaux        | Ch. 2-3 ✅, Ch. 4 → 13 | 🚧 2/12 |
| II — Mécanismes avancés | Ch. 14 → 18          | ⬜ 0/5  |
| III — Concurrence       | Ch. 19 → 23          | ⬜ 0/5  |
| IV — Runtime & mémoire  | Ch. 24 → 29          | ⬜ 0/6  |
| V — Internals           | Ch. 30 → 35          | ⬜ 0/6  |
| VI — Performance        | Ch. 36 → 40          | ⬜ 0/5  |
| VII — Projets           | Projets 1 → 7        | ⬜ 0/7  |
| Annexes                 | A → G                | ⬜ 0/7  |

### Infrastructure

- ✅ Squelette du dépôt, `README.md`, `SOMMAIRE.md`, `.gitignore`.
- ✅ Module `code/` (`example.com/gobook`, `go 1.26`) — `go test ./...` & `go vet ./...` propres.
- ✅ Gabarit de chapitre (`chapitres/_gabarit.md`).
- ✅ Exemples compilables et testés : `code/ch01-hello/` (`greet`), `code/ch02-structure/`
  (multi-packages `main` + `greeting`), `code/ch03-basics/` (zero values, conversions,
  `iota`/`ByteSize`, conversion sûre, `new(expr)`).
- ✅ Nouveautés 1.26 **vérifiées sur la toolchain 1.26.4** : `new(expr)` (type inféré),
  `min`/`max`/`clear`, débordement silencieux vs erreur de compilation sur constante.
- ⬜ CI (GitHub Actions) lançant `go test ./...` + `go vet ./...` + `gofmt -l`.

**Prochaine action concrète** : rédiger le **Ch. 4 — Flux de contrôle** (+ exemple
`code/ch04-...`), puis enchaîner la Partie I.
