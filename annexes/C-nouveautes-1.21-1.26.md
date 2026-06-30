# Annexe C — Carte des nouveautés Go 1.21 → 1.26

> **Objectif** — Offrir une **carte synthétique**, version par version, de ce qui a
> changé dans Go de la 1.21 à la 1.26 : langage, runtime, bibliothèque standard et
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
| Bibliothèque | `slices`, `maps`, `cmp`         | Opérations génériques sur tranches/maps ; `cmp.Compare`/`cmp.Or` pour les types ordonnés — évite de réécrire les mêmes utilitaires (`Contains`, `Index`, tri…) dans chaque projet. 🔁 Ch. 6, 11      |
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
| Bibliothèque | Refonte de `time.Timer`/`Ticker`   | Timers non démarrés plus simples à collecter ; canal non bufferisé.                                                                                                   |
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

| Domaine      | Nouveauté                                      | En bref                                                                                                                                        |
| ------------ | ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Langage      | **`new(expr)`**                                | `new` accepte une **expression** : alloue et **initialise** en une fois (`p := new(1 + 2)`). 🔁 Projet 2                                       |
| Langage      | Contraintes génériques **auto-référentielles** | Une contrainte peut se nommer elle-même (`type Adder[T] interface{ … T … }`). 🔁 Ch. 11                                                        |
| Bibliothèque | `go/ast.ParseDirective`                        | Décode une directive `//tool:name args` en `{Tool, Name, Args}`. 🔁 Projet 6                                                                   |
| Bibliothèque | `go/ast.BasicLit.ValueEnd`                     | Position **juste après** un littéral : diagnostics précis (portée exacte). 🔁 Projet 6                                                         |
| Bibliothèque | `errors.AsType`                                | Version **générique et type-sûre** de `errors.As` : plus rapide, sans réflexion, sans piège de pointeur-vers-pointeur. 🔁 Ch. 10               |
| Bibliothèque | `slog.NewMultiHandler`                         | Diffuser un même enregistrement vers **plusieurs handlers** (ex. texte + JSON) sans écrire de handler composite à la main. 🔁 Ch. 43, Projet 2 |
| Runtime      | GC **Green Tea** par défaut                    | Activé par défaut après son passage en expérimental en 1.25 : 10 à 40 % de coût GC en moins sur les programmes qui en abusent. 🔁 Ch. 27       |
| Runtime      | cgo plus rapide                                | Le surcoût d'un appel cgo baisse d'environ 30 %, allégeant la pénalité historique de l'interop C. 🔁 Ch. 35                                    |
| Outils       | `go fix` et les **modernizers**                | Réécriture automatisée du code vers les idiomes/API récents ; directives `//go:fix inline` pour vos propres migrations. 🔁 Ch. 13              |

> ⚠️ **Prudence sur 1.26** — Ce tableau ne liste que des éléments confirmés sur la
> toolchain `go1.26.4` (vérifiés via `go doc` / compilation). La version apporte
> par ailleurs les améliorations habituelles du compilateur, du runtime et de
> l'outillage ; reportez-vous aux **notes de version officielles** (🔁 Annexe G)
> pour la liste exhaustive.

---

## Vue d'ensemble (fil conducteur)

```
  1.21        1.22         1.23          1.24            1.25             1.26
  ----        ----         ----          ----            ----             ----
  min/max     loopvar*     iterators     b.Loop          synctest         new(expr)
  clear       range int    iter.Seq      type alias gén. WaitGroup.Go     contraintes
  slices/     ServeMux     unique        SplitSeq        FlightRecorder   auto-réf.
  maps/cmp    (PathValue)  slices×iter   weak / os.Root  CSRF protection  ParseDirective
  slog        rand/v2      time.Timer    Swiss Tables    GOMAXPROCS cgrp  ValueEnd
  PGO (prod)               télémétrie    tool directive  json/v2 (exp.)   Green Tea déf.

  * loopvar : préversion en 1.21, par défaut en 1.22 (lié à la ligne go de go.mod)
```

---

## 📌 À retenir

- La **ligne `go` de `go.mod`** décide quelles nouveautés de **langage**
  s'appliquent : mettre à jour la toolchain ne suffit pas toujours.
- Fil rouge **1.23–1.25** : les **itérateurs** (`iter.Seq`) irriguent `slices`,
  `maps`, `strings`, et deviennent un idiome central.
- Fil rouge **performance/observabilité** : **PGO** (1.21), **Swiss Tables**
  (1.24), **FlightRecorder** et `GOMAXPROCS` conscient des conteneurs (1.25),
  GC **Green Tea** — expérimental en 1.25, **par défaut en 1.26**.
- **1.26** : GC Green Tea par défaut, `go fix`/modernizers, `new(expr)`,
  contraintes auto-référentielles, et l'outillage AST (`ParseDirective`,
  `BasicLit.ValueEnd`) pour la génération de code.
- En cas de doute sur une API, la **source de vérité** est `go doc` sur votre
  toolchain et les **notes de version** officielles (🔁 Annexe G).
