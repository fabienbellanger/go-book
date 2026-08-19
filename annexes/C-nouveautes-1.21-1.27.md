# Annexe C — Carte des nouveautés Go 1.21 → 1.27

> **Objectif** — Offrir une **carte synthétique**, version par version, de ce qui a
> changé dans Go de la 1.21 à la 1.27 : langage, runtime, bibliothèque standard et
> outils. À utiliser comme aide-mémoire pour savoir « depuis quand » une
> fonctionnalité existe et vers quel chapitre se tourner.

---

Convention de lecture : chaque version est résumée par un tableau **Domaine |
Nouveauté | En bref**, regroupé par domaine. Les renvois 🔁 pointent vers le
chapitre ou le projet qui développe le sujet.

> 💡 **Politique de compatibilité** — La directive `go 1.NN` dans `go.mod` fixe la
> sémantique du langage utilisée pour compiler le module. Grâce aux garanties de
> **compatibilité descendante et ascendante** (généralisées en 1.21), mettre à
> jour la toolchain ne change pas le comportement d'un module tant que sa ligne
> `go` ne bouge pas. ⚠️ Certaines nouveautés de langage (range sur entier,
> variable de boucle par itération…) ne s'activent qu'à partir d'une certaine
> ligne `go` dans `go.mod`.

---

## Go 1.21

| Domaine      | Nouveauté                       | En bref                                                                                                                                                                                              |
| ------------ | ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Langage      | Builtins `min`, `max`, `clear`  | `min`/`max` sur types ordonnés ; `clear(m)` vide une map, `clear(s)` met une tranche à zéro.                                                                                                         |
| Langage      | Inférence de type améliorée     | Inférence plus complète pour les fonctions génériques.                                                                                                                                               |
| Bibliothèque | `slices`, `maps`, `cmp`         | Opérations génériques sur tranches/maps ; `cmp.Compare`/`cmp.Less` pour les types ordonnés — évite de réécrire les mêmes utilitaires (`Contains`, `Index`, tri…) dans chaque projet. 🔁 Ch. 6, 11    |
| Bibliothèque | `log/slog`                      | **Journalisation structurée** officielle (handlers texte/JSON, niveaux, attributs) : plus besoin d'une dépendance tierce (logrus, zap) pour des logs exploitables en production. 🔁 Ch. 43, Projet 2 |
| Bibliothèque | `sync.OnceFunc`/`OnceValue`     | Mémoïsation d'une initialisation paresseuse sans écrire un `sync.Once` à la main.                                                                                                                    |
| Runtime      | Ordre d'exécution des `init`    | Ordre des paquets d'initialisation rendu déterministe.                                                                                                                                               |
| Outils       | **PGO** prêt pour la production | L'optimisation guidée par profil (`default.pgo`) sort de l'expérimental. 🔁 Ch. 39, Projet 7                                                                                                         |
| Outils       | `loopvar` en préversion         | Comportement « une variable par itération » testable via `GOEXPERIMENT=loopvar` (officialisé en 1.22).                                                                                               |

---

## Go 1.22

| Domaine      | Nouveauté                              | En bref                                                                                                                                                                              |
| ------------ | -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Langage      | **Variable de boucle par itération**   | Dans un `for`, la variable est désormais **redéfinie à chaque tour** : fin du piège de capture. 🔁 Ch. 15, 23                                                                        |
| Langage      | `range` sur un entier                  | `for i := range n { … }` itère de 0 à n-1.                                                                                                                                           |
| Bibliothèque | Routage enrichi de `net/http.ServeMux` | Motifs avec **méthode** (`GET /…`), **jokers** (`/{id}`) et `r.PathValue("id")` : un routeur correct sans dépendance externe (chi, gorilla/mux) pour la plupart des API. 🔁 Projet 2 |
| Bibliothèque | `math/rand/v2`                         | Première version `v2` de la stdlib : API nettoyée, meilleurs générateurs.                                                                                                            |
| Bibliothèque | `cmp.Or`                               | Renvoie le premier argument **non-zéro** : valeurs par défaut concises, comparaisons multi-critères.                                                                                 |
| Runtime      | Métadonnées plus compactes             | Optimisations mémoire/CPU de l'allocateur et du ramasse-miettes.                                                                                                                     |
| Outils       | `go vet` : avertissements de boucle    | Ajustements liés au nouveau comportement des variables de boucle.                                                                                                                    |

> ⚠️ Le nouveau comportement des variables de boucle ne s'active que si `go.mod`
> déclare `go 1.22` (ou plus). C'est l'exemple type d'une nouveauté **liée à la
> ligne `go`**.

---

## Go 1.23

| Domaine      | Nouveauté                          | En bref                                                                                                                                                               |
| ------------ | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Langage      | **Itérateurs « range-over-func »** | `for x := range seq` où `seq` est une fonction ; types `iter.Seq[T]`, `iter.Seq2[K,V]`. 🔁 Ch. 18                                                                     |
| Bibliothèque | `iter`                             | Le contrat des itérateurs (push) que consomment `slices`/`maps`.                                                                                                      |
| Bibliothèque | `slices`/`maps` × itérateurs       | `slices.Collect`, `slices.Sorted`, `slices.Values`, `maps.Keys`, `maps.Values`…                                                                                       |
| Bibliothèque | `unique`                           | **Internement** de valeurs comparables : déduplique et accélère les comparaisons — utile pour des ensembles de chaînes très répétées (tags, identifiants) en mémoire. |
| Bibliothèque | Refonte de `time.Timer`/`Ticker`   | Timers **non référencés** collectés par le GC même sans `Stop()` ; canal non bufferisé.                                                                               |
| Outils       | Télémétrie opt-in de la toolchain  | Collecte **facultative** de statistiques d'usage de l'outillage Go.                                                                                                   |

---

## Go 1.24

| Domaine      | Nouveauté                                | En bref                                                                                                         |
| ------------ | ---------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| Langage      | **Alias de types génériques**            | Un alias `type` peut désormais porter des paramètres de type. 🔁 Ch. 11                                         |
| Bibliothèque | **API de benchmark `for b.Loop()`**      | Boucle de mesure plus sûre : préparation hors boucle, pas d'élimination par le compilateur. 🔁 Ch. 36, Projet 7 |
| Bibliothèque | `strings.SplitSeq`/`Lines`/`FieldsSeq`   | Variantes **itérateur** (sans allouer la tranche) de `Split`/`Fields`. 🔁 Ch. 31                                |
| Bibliothèque | `weak`                                   | **Pointeurs faibles** : référencer sans empêcher la collecte. 🔁 Ch. 27                                         |
| Bibliothèque | `runtime.AddCleanup`                     | Remplaçant moderne et plus sûr de `SetFinalizer` (plusieurs cleanups, pas de résurrection).                     |
| Bibliothèque | `os.Root`                                | Accès fichiers **confiné** à un répertoire (anti « path traversal »).                                           |
| Bibliothèque | `encoding.TextAppender`/`BinaryAppender` | Sérialiser en **ajoutant** à un buffer existant, sans allocation intermédiaire.                                 |
| Runtime      | Maps « **Swiss Tables** »                | Nouvelle implémentation interne des maps : plus rapide, moins de mémoire. 🔁 Ch. 32                             |
| Outils       | Directive `tool` dans `go.mod`           | Déclarer les **outils** d'un module (remplace l'astuce du fichier `tools.go`).                                  |

---

## Go 1.25

| Domaine      | Nouveauté                             | En bref                                                                                                        |
| ------------ | ------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| Bibliothèque | **`testing/synctest` stable**         | `synctest.Test(t, f)` exécute du code concurrent en **temps virtuel** ; `synctest.Wait()`. 🔁 Ch. 23, Projet 3 |
| Bibliothèque | `sync.WaitGroup.Go`                   | `wg.Go(f)` lance la goroutine **et** gère `Add`/`Done`. 🔁 Ch. 21, Projets 5, 7                                |
| Bibliothèque | `runtime/trace.FlightRecorder`        | **Enregistreur de vol** : fenêtre glissante de trace, figée à la demande. 🔁 Ch. 38, Projet 7                  |
| Bibliothèque | `net/http.CrossOriginProtection`      | Protection **anti-CSRF** intégrée fondée sur l'origine de la requête. 🔁 Projet 2                              |
| Bibliothèque | `encoding/json/v2` (expérimental)     | Refonte de l'API JSON, derrière `GOEXPERIMENT=jsonv2`.                                                         |
| Runtime      | `GOMAXPROCS` conscient des conteneurs | Par défaut, respecte les **limites CPU cgroup** (utile en Kubernetes/Docker). 🔁 Ch. 28                        |
| Runtime      | GC « **Green Tea** » (expérimental)   | Nouvelle conception du ramasse-miettes, derrière `GOEXPERIMENT`. 🔁 Ch. 27                                     |

---

## Go 1.26

| Domaine      | Nouveauté                                      | En bref                                                                                                                                                            |
| ------------ | ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Langage      | **`new(expr)`**                                | `new` accepte une **expression** : alloue et **initialise** en une fois (`p := new(1 + 2)`). 🔁 Projet 2                                                           |
| Langage      | Contraintes génériques **auto-référentielles** | Un type générique peut se référer à lui-même dans sa liste de paramètres de type (`type Adder[A Adder[A]] interface{ Add(A) A }`) — auparavant interdit. 🔁 Ch. 11 |
| Bibliothèque | `go/ast.ParseDirective`                        | Décode une directive `//tool:name args` en `{Tool, Name, Args}`. 🔁 Projet 6                                                                                       |
| Bibliothèque | `go/ast.BasicLit.ValueEnd`                     | Position **juste après** un littéral : diagnostics précis (portée exacte). 🔁 Projet 6                                                                             |
| Bibliothèque | `errors.AsType`                                | Version **générique et type-sûre** de `errors.As` : plus rapide, sans réflexion, sans piège de pointeur-vers-pointeur. 🔁 Ch. 10                                   |
| Bibliothèque | `slog.NewMultiHandler`                         | Diffuser un même enregistrement vers **plusieurs handlers** (ex. texte + JSON) sans écrire de handler composite à la main. 🔁 Ch. 43, Projet 2                     |
| Runtime      | GC **Green Tea** par défaut                    | Activé par défaut après son passage en expérimental en 1.25 : 10 à 40 % de coût GC en moins sur les programmes qui en abusent. 🔁 Ch. 27                           |
| Runtime      | cgo plus rapide                                | Le surcoût d'un appel cgo baisse d'environ 30 %, allégeant la pénalité historique de l'interop C. 🔁 Ch. 35                                                        |
| Outils       | `go fix` et les **modernizers**                | Réécriture automatisée du code vers les idiomes/API récents ; directives `//go:fix inline` pour vos propres migrations. 🔁 Ch. 1                                   |


---

## Go 1.27

| Domaine      | Nouveauté                                       | En bref                                                                                                                                                                     |
| ------------ | ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Langage      | **Méthodes génériques**                         | Une méthode peut déclarer ses **propres** paramètres de type : `func (l List[E]) Apply[F any](f func(E) F) List[F]`. Fin de la limitation « pas de `Map` en méthode ». 🔁 Ch. 11 |
| Langage      | Clés de littéral = **sélecteurs de champ**      | Dans un littéral de struct, la clé peut désigner un champ **promu** : `Point3D{name: "o", x: 1}` au lieu de `Point3D{Object: Object{name: "o"}, x: 1}`. 🔁 Ch. 8            |
| Langage      | Inférence de type **généralisée**               | L'inférence s'applique dans **tous** les contextes d'affectation d'une fonction générique : `var g func([]int, func(int) string) []string = Map`. 🔁 Ch. 11                  |
| Bibliothèque | **`encoding/json/v2`** stable                   | Refonte de l'API JSON (options, `MarshalWrite`/`UnmarshalRead`, refus des doublons et de l'UTF-8 invalide). `encoding/json` v1 repose désormais dessus. 🔁 Ch. 42            |
| Bibliothèque | **`encoding/json/jsontext`**                    | Couche **syntaxique** bas niveau : `Encoder`/`Decoder` de *tokens*, `Value`, `Pointer` — pour streamer ou transformer du JSON sans le typer. 🔁 Ch. 42                       |
| Bibliothèque | `strings.CutLast` / `bytes.CutLast`             | Découpe autour de la **dernière** occurrence d'un séparateur : remplace le couple `LastIndex` + tranchage. 🔁 Ch. 7, 31                                                     |
| Bibliothèque | **`uuid`**                                      | UUID (RFC 9562) dans la stdlib : `uuid.New()`, `NewV4`, `NewV7`, `Parse`, `Nil`, `Max`. Plus de dépendance tierce pour un identifiant. 🔁 Ch. 51, Projet 2                   |
| Bibliothèque | `hash/maphash.Hasher[T]`                        | Contrat `Hash`/`Equal` pour les structures de données à hachage maison ; `ComparableHasher[T]` fournit l'implémentation par défaut. 🔁 Ch. 32                                |
| Bibliothèque | `math/rand/v2` : méthode générique `N`          | Première méthode générique de la stdlib : `r.N(int32(10))` en plus de `rand.N`. 🔁 Ch. 11                                                                                   |
| Bibliothèque | `testing/synctest.Sleep`                        | `synctest.Sleep(d)` = `time.Sleep(d)` + `synctest.Wait()` : le test attend que la bulle se stabilise. 🔁 Ch. 23, 44                                                          |
| Bibliothèque | `httptest.NewTestServer`                        | Serveur de test sur un réseau **en mémoire**, lié au `testing.TB` — compatible avec `testing/synctest`. 🔁 Ch. 45, Projet 2                                                  |
| Bibliothèque | `net/url` : `URL.Clone` / `Values.Clone`        | Copies **profondes**, au lieu de recopier champ par champ à la main. 🔁 Ch. 45                                                                                              |
| Bibliothèque | `net/http` : priorités HTTP/2, en-têtes bornés  | Le serveur honore les signaux de priorité RFC 9218 (`Server.DisableClientPriority` pour revenir au tourniquet) ; `Server.MaxHeaderValueCount` borne le nombre de valeurs. 🔁 Ch. 45 |
| Bibliothèque | **`crypto/mldsa`**                              | Signatures **post-quantiques** ML-DSA (FIPS 204), câblées dans `crypto/x509` et TLS 1.3. 🔁 Ch. 53                                                                           |
| Bibliothèque | `math/big.Int.Divide`                           | Quotient **et** reste selon un mode d'arrondi (`Trunc`, `Floor`, `Ceil`, `Round`) — la division euclidienne enfin explicite.                                                |
| Bibliothèque | `unicode` : Unicode 17                          | Passage d'Unicode 15 à **17** (nouveaux scripts, nouvelles `RangeTable`). 🔁 Ch. 31                                                                                         |
| Runtime      | **Profil `goroutineleak` stable**               | Détection des goroutines bloquées à jamais, promue hors de `GOEXPERIMENT` : `/debug/pprof/goroutineleak`. 🔁 Ch. 23, 29                                                      |
| Runtime      | Allocation **spécialisée par taille**           | Le compilateur appelle des routines d'allocation dédiées : jusqu'à **30 %** plus rapide sous 80 octets, ~1 % sur un programme gourmand en allocations. 🔁 Ch. 26            |
| Runtime      | Étiquettes de goroutine dans les *tracebacks*   | Les labels `runtime/pprof` apparaissent dans l'en-tête des piles (`GODEBUG=tracebacklabels=0` pour désactiver). 🔁 Ch. 17, 29                                                |
| Runtime      | `asynctimerchan` **supprimé**                   | Les canaux de `time` sont désormais **toujours** synchrones, quel que soit le `GODEBUG`. 🔁 Ch. 44                                                                           |
| Outils       | `go fix` : nouveaux modernizers                 | Ajout de `atomictypes`, `embedlit`, `slicesbackward`, `unsafefuncs` ; `fmtappendf` retiré, `waitgroup` renommé `waitgroupgo`. 🔁 Ch. 13                                     |
| Outils       | `go vet` : `stdversion` par défaut              | `go test` signale l'usage d'un symbole de la stdlib **trop récent** pour la ligne `go` du module. 🔁 Ch. 13                                                                 |
| Outils       | `go doc` : `@version` et `-ex`                  | `go doc example.com/pkg@v1.2.3` ; `-ex` liste les exemples exécutables, et `go doc bytes.ExampleBuffer` en imprime la source. 🔁 Ch. 13                                     |
| Outils       | `go mod tidy` : blocs `require` normalisés      | Pour les modules en `go 1.27`+, fusion des `require` en **deux blocs** (directes / indirectes) avec conservation des commentaires. 🔁 Ch. 12                                |
| Outils       | `go tool trace -http=:6060` → **localhost**     | N'écoute plus sur toutes les interfaces quand seul le port est donné (aligné sur `pprof`). 🔁 Ch. 38                                                                        |
| Ports        | macOS 13+, ppc64 en ELFv2                       | macOS 12 et antérieurs ne sont plus gérés ; `linux/ppc64` (gros-boutiste) passe à l'ABI ELFv2 et gagne cgo, PIE et l'édition de liens externe.                              |

> ⚠️ **Deux « expérimentaux » devenus des défauts** — `encoding/json/v2` (opt-out
> `GOEXPERIMENT=nojsonv2`) et l'allocation spécialisée par taille (opt-out
> `GOEXPERIMENT=nosizespecializedmalloc`, retrait prévu en 1.28). Le comportement
> de `encoding/json` v1 est préservé, mais **les messages d'erreur changent** :
> ne les comparez jamais littéralement dans un test.

---

## Vue d'ensemble (fil conducteur)

```
  1.21        1.22         1.23           1.24             1.25             1.26                   1.27
  ----        ----         ----           ----             ----             ----                   ----
  min/max     loopvar*     iterators      b.Loop           synctest         new(expr)              méthodes génériques
  clear       range int    iter.Seq       type alias gén.  WaitGroup.Go     contraintes auto-réf.  clés = champ promu
  slices/     ServeMux     unique         SplitSeq         FlightRecorder   ParseDirective         json/v2 + jsontext
  maps/cmp    (PathValue)  slices × iter  weak / os.Root   CSRF protection  ValueEnd               uuid / crypto/mldsa
  slog        rand/v2      time.Timer     Swiss Tables     GOMAXPROCS cgrp  Green Tea déf.         CutLast
  PGO (prod)               télémétrie     tool directive   json/v2 (exp.)   goroutineleak (exp.)   goroutineleak stable

  * loopvar : préversion en 1.21, par défaut en 1.22 (lié à la ligne `go` de go.mod)
```

---

## 📌 À retenir

- La **ligne `go` de `go.mod`** décide quelles nouveautés de **langage**
  s'appliquent : mettre à jour la toolchain ne suffit pas toujours.
- Fil rouge **1.23–1.25** : les **itérateurs** (`iter.Seq`) irriguent `slices`,
  `maps`, `strings`, et deviennent un idiome central.
- Fil rouge **performance/observabilité** : **PGO** (1.21), **Swiss Tables**
  (1.24), **FlightRecorder** et `GOMAXPROCS` conscient des conteneurs (1.25),
  GC **Green Tea** — expérimental en 1.25, **par défaut en 1.26** —, allocation
  spécialisée par taille et profil `goroutineleak` stable (1.27).
- **1.26** : GC Green Tea par défaut, `go fix`/modernizers, `new(expr)`,
  contraintes auto-référentielles, et l'outillage AST (`ParseDirective`,
  `BasicLit.ValueEnd`) pour la génération de code.
- **1.27** : la **généricité se complète** (méthodes génériques, inférence
  généralisée), **`encoding/json/v2`** devient le moteur par défaut de
  `encoding/json`, et la stdlib absorbe deux dépendances universelles —
  **`uuid`** et les signatures post-quantiques **`crypto/mldsa`**.
- Motif récurrent : une fonctionnalité naît derrière un `GOEXPERIMENT`
  (json/v2 en 1.25, `goroutineleak` en 1.26), devient le défaut une ou deux
  versions plus tard, puis l'opt-out disparaît. Anticipez le retrait de l'opt-out
  plutôt que de vous y adosser.
- En cas de doute sur une API, la **source de vérité** est `go doc` sur votre
  toolchain et les **notes de version** officielles (🔁 Annexe G).
