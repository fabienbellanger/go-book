# 0 — Pourquoi Go ? Philosophie & panorama

> **Objectif** — Situer Go parmi les langages, comprendre ses partis pris, et savoir
> comment lire ce livre.

---

## Introduction

Go (souvent écrit **Golang** pour les recherches web) est un langage **compilé**,
**statiquement typé**, doté d'un **ramasse-miettes** (garbage collector, GC) et d'une
**concurrence native**. Il a été conçu pour une seule obsession : **rester simple à
grande échelle** — grandes équipes, grandes bases de code, gros déploiements.

En une phrase : _Go privilégie la lisibilité et l'outillage à l'expressivité maximale._
Là où d'autres langages ajoutent des fonctionnalités, Go en retire. C'est un choix
inhabituel, et c'est ce qui fait sa force comme ses limites.

## Une histoire courte

Go naît chez **Google en 2007** (annonce publique fin **2009**, version **1.0 en 2012**).
Ses auteurs — **Robert Griesemer, Rob Pike et Ken Thompson** — partent d'un constat :
la compilation des énormes services C++ de Google était lente, et le langage devenu
complexe. Leur réponse : un langage **qui compile vite**, **se lit facilement**, et
**gère la concurrence** sans douleur.

```
  1972  C            (Thompson, Ritchie)
   |
  1980s C++          (objets, templates, complexité croissante)
   |
  2009  Go           "la simplicité de C, l'outillage moderne, la concurrence native"
```

## Les partis pris de design

Go fait des choix **assumés**, parfois à contre-courant :

- **Simplicité avant tout** — peu de mots-clés (25), une seule façon de faire les
  choses. La devise officieuse : _« less is more »_.
- **Lisibilité > concision** — le code est lu bien plus souvent qu'écrit. Un formatage
  unique imposé (`gofmt`) met fin aux débats de style.
- **Compilation rapide** — un gros projet compile en secondes, pas en minutes.
- **Concurrence native** — les **goroutines** et les **channels** sont dans le langage,
  pas dans une bibliothèque (voir Partie III).
- **Outillage intégré** — formatage, tests, profiling, doc, analyse statique :
  tout vient avec `go`, sans dépendances externes.
- **Compatibilité** — le code écrit pour Go 1.0 compile encore aujourd'hui
  (la _« Go 1 promise »_, voir plus bas).
- **Pragmatisme** — pas d'exceptions (les erreurs sont des valeurs), pas d'héritage
  (composition à la place), pas de surcharge d'opérateurs.

> 💡 Ce que Go **n'a pas** est aussi important que ce qu'il a : pas d'héritage de classes,
> pas d'exceptions, pas de génériques avant 1.18, pas de surcharge. Chaque absence est un
> choix pour réduire la complexité.

## Compilé, typé statiquement, ramassé

Trois caractéristiques structurantes :

- **Compilé en binaire natif** — Go produit un **exécutable autonome** (un seul fichier),
  sans machine virtuelle ni runtime à installer. Le déploiement se résume à copier
  un fichier.
- **Statiquement typé** — les types sont vérifiés **à la compilation**. Beaucoup
  d'erreurs sont attrapées avant l'exécution. Mais l'**inférence de type** (`:=`) évite
  la lourdeur de la déclaration explicite partout.
- **Ramasse-miettes** — pas de `malloc`/`free` manuel. Le GC libère la mémoire
  automatiquement, avec des **temps de pause très courts** (voir Ch. 27).

## Comparaison express

| Critère                | Go            | C / C++          | Java           | Python        |
| ---------------------- | ------------- | ---------------- | -------------- | ------------- |
| Exécution              | binaire natif | binaire natif    | JVM (bytecode) | interprété    |
| Typage                 | statique      | statique         | statique       | dynamique     |
| Gestion mémoire        | GC            | manuelle         | GC             | GC            |
| Concurrence            | goroutines    | threads / libs   | threads        | GIL / asyncio |
| Compilation            | très rapide   | lente            | moyenne        | (aucune)      |
| Déploiement            | 1 fichier     | dépendances libs | JRE requis     | interpréteur  |
| Courbe d'apprentissage | douce         | raide            | moyenne        | douce         |

Go vise le **point d'équilibre** : presque aussi rapide que C, presque aussi simple
que Python, avec une concurrence que peu de langages égalent.

## Cycle de release & la « Go 1 promise »

Go suit un **rythme semestriel** : une version mineure tous les ~6 mois
(février et août), nommées `1.21`, `1.22`, … jusqu'à **`1.26`** que cible ce livre.

La **Go 1 promise** (« promesse de compatibilité ») garantit que du code valide écrit
pour Go 1.x continuera de compiler et de fonctionner avec les versions ultérieures de
Go 1.x. En pratique : **on met à jour la toolchain sans réécrire son code**. Les
évolutions du langage sont donc **incrémentales et rétrocompatibles**.

> 🆕 Tout au long du livre, un encart **🆕** signale ce qui a été introduit récemment
> (1.21 → 1.26), pour distinguer le Go « classique » des ajouts récents.

## Panorama des nouveautés 1.21 → 1.26

Un aperçu de ce qui a marqué les versions récentes (détail en **annexe C**) :

| Version  | Points marquants                                                                                                  |
| -------- | ----------------------------------------------------------------------------------------------------------------- |
| **1.21** | `min`/`max`/`clear` natifs ; packages `slices`/`maps`/`cmp` ; **PGO** en GA                                       |
| **1.22** | `for range N` (entier) ; **portée par itération** de la variable de boucle ; routage `net/http` enrichi           |
| **1.23** | **itérateurs** (`iter`, range-over-func) ; package `unique` (interning)                                           |
| **1.24** | **Swiss Tables** pour les maps ; pointeurs faibles (`weak`) ; `runtime.AddCleanup` ; alias de type génériques     |
| **1.25** | `testing/synctest` (GA) ; `GOMAXPROCS` conscient des cgroups ; `FlightRecorder` ; GC **Green Tea** (expérimental) |
| **1.26** | **Green Tea GC par défaut** ; `go fix` (modernizers) ; cgo plus rapide ; `errors.AsType`                          |

Ne cherchez pas à tout retenir maintenant : chaque point est expliqué le moment venu.

## Comment lire ce livre

Le livre suit une progression **top-down** : on apprend d'abord à **utiliser** Go
(Parties 0 à III), puis on **ouvre le capot** pour comprendre comment il fonctionne
(Parties IV à VI), avant de **mettre en pratique** (Partie VII — projets).

Choisissez votre **parcours** selon votre objectif (détail dans le `README`) :

- 🟢 **Débutant Go** : lisez dans l'ordre les Parties 0 → I → II → III.
- 🔵 **Vous connaissez Go** : sautez aux internals (Parties IV → V → VI).
- 🟣 **Focus concurrence** ou 🟠 **focus performance** : voir les parcours dédiés.

Repères visuels utilisés partout :

| Émoji | Sens                            |
| ----- | ------------------------------- |
| 🆕    | nouveauté d'une version récente |
| ⚠️    | piège classique                 |
| 💡    | astuce                          |
| 🔁    | renvoi vers un autre chapitre   |
| ⚡    | note de performance             |
| 🧪    | à tester / mesurer soi-même     |
| 📌    | synthèse à retenir              |

Les **exemples de code** sont **compilables** et vivent dans le dossier `code/`
(un seul module Go). Vous pourrez les exécuter et les modifier — c'est la meilleure
façon d'apprendre.

---

## 📌 À retenir

- Go est **compilé**, **statiquement typé**, **ramassé**, avec une **concurrence native**.
- Sa philosophie : **la simplicité à grande échelle** — _« less is more »_.
- Un **outillage intégré** (`gofmt`, `go test`, `pprof`…) fait partie du langage.
- La **Go 1 promise** garantit la rétrocompatibilité : on met à jour sans réécrire.
- Ce livre cible **Go 1.26** et signale les nouveautés par des encarts 🆕.

## 🔁 Pour aller plus loin

- Chapitre suivant : [Ch. 1 — Installation, toolchain & premier programme](01-installation-toolchain.md).
- Annexe C — Carte complète des nouveautés Go 1.21 → 1.26.
- [Effective Go](https://go.dev/doc/effective_go) et le [blog officiel](https://go.dev/blog/).
