# 27 — Le garbage collector

> **Objectif** — Comprendre et **régler** le GC de Go : un **mark-sweep concurrent tri-couleur**, ses
> **write barriers**, son **pacing** (`GOGC`, `GOMEMLIMIT`), comment lire un cycle avec
> `GODEBUG=gctrace=1`, et les outils de fin de vie d'objet (**finalizers**, `runtime.AddCleanup`,
> **pointeurs faibles**).
>
> **Prérequis** — [Ch. 25](25-modele-memoire.md), [Ch. 26](26-allocation-escape.md)

---

## Introduction

Le [Ch. 26](26-allocation-escape.md) a montré ce qui part sur le **tas**. Reste à le **récupérer** :
c'est le rôle du **garbage collector**. Celui de Go est conçu pour une **faible latence** — il tourne
**en même temps** que votre programme, avec des pauses « stop-the-world » de l'ordre de la
**sous-milliseconde**, au prix d'un peu de débit CPU. Vous n'allouez ni ne libérez à la main, mais vous
pouvez **comprendre** son comportement et le **régler** via deux molettes : `GOGC` et `GOMEMLIMIT`.
Code dans [`code/ch27-garbage-collector/`](../code/ch27-garbage-collector/).

---

## Mark-sweep concurrent tri-couleur

Le GC procède en deux temps : **marquer** ce qui est atteignable depuis les **racines** (piles des
goroutines, variables globales), puis **balayer** (sweep) le reste. Pour marquer **sans arrêter** le
programme, il utilise l'abstraction **tri-couleur** :

```
  BLANC : pas encore visité (candidat à la collecte)
  GRIS  : atteint, mais ses pointeurs pas encore parcourus
  NOIR  : atteint ET tous ses pointeurs parcourus (vivant)

  racines (piles, globals)        tas
  +--------+   marque   +-------+    +-------+    +-------+
  | NOIR   |----------->| GRIS  |--->| BLANC |    | BLANC |  <- inatteignable
  +--------+            +-------+    +-------+    +-------+
                         (file)         ^             |
                                        |          sweep : récupéré
                       on grise, on noircit, jusqu'à file grise vide

  Invariant : aucun objet NOIR ne pointe DIRECTEMENT vers un objet BLANC.
```

Quand la file grise est vide, tout ce qui est encore **blanc** est inatteignable : le sweep le
récupère (paresseusement, au fil des allocations suivantes). Le GC de Go est **non compactant** : il ne
déplace pas les objets vivants (les adresses restent stables pendant la vie de l'objet).

## Les write barriers

Problème : pendant le marquage **concurrent**, le programme **modifie** des pointeurs. Il pourrait
cacher un objet vivant derrière un objet déjà noirci, et le GC le collecterait à tort. La **write
barrier** l'empêche : à chaque écriture de pointeur **pendant le marquage**, un petit bout de code
maintient l'invariant tri-couleur (Go utilise une barrière **hybride** depuis la 1.8). C'est le même
genre de **barrière mémoire** que celle du [Ch. 25](25-modele-memoire.md), mais au service du GC. Elle
n'est active **que** durant la phase de marquage.

## GC pacing : `GOGC` et `GOMEMLIMIT`

Le **pacer** décide **quand** lancer un cycle pour qu'il se termine **avant** que le tas ne dépasse un
objectif. L'objectif dépend de **`GOGC`** (défaut **100**) :

```
  objectif = tas_vivant * (1 + GOGC/100)
  GOGC=100  ->  on laisse le tas DOUBLER avant le prochain GC (compromis par défaut)
  GOGC=200  ->  GC moins fréquent, PLUS de mémoire, MOINS de CPU
  GOGC=50   ->  GC plus fréquent, MOINS de mémoire, PLUS de CPU
  GOGC=off  ->  GC désactivé (à vos risques)
```

**`GOMEMLIMIT`** (depuis 1.19) ajoute une **limite mémoire douce** : à mesure que le tas s'en approche,
le GC devient **plus agressif**, quitte à ignorer `GOGC`. Idéal en **conteneur** : on fixe
`GOMEMLIMIT` proche de la limite du cgroup pour éviter l'OOM-kill. « Douce » = Go préfère dépasser la
limite plutôt que s'étrangler, mais fait tout pour la tenir.

```go
// code/ch27-garbage-collector/gc.go
func WithGCPercent(pct int, f func()) {
	old := debug.SetGCPercent(pct) // équivaut à GOGC, par programme
	defer debug.SetGCPercent(old)
	f()
}

func CurrentMemoryLimit() int64 { return debug.SetMemoryLimit(-1) } // -1 = lecture
```

> 💡 Réglez de préférence par **variables d'environnement** (`GOGC`, `GOMEMLIMIT=512MiB`) : aucun
> recompilation, et c'est l'usage attendu en production.

## Lire un cycle : `GODEBUG=gctrace=1`

```
$ GODEBUG=gctrace=1 go run ./ch27-garbage-collector
gc 1 @0.005s 2%: 0.026+0.37+0.042 ms clock, ... , 3->4->0 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 8 P
    |    |    |   |      |     |                  |       |        |
    |    |    |   |      |     +- STW fin marquage|       |        +- nb de P
    |    |    |   |      +- marquage concurrent   |       +- objectif de tas
    |    |    |   +- STW début                    +- tas: avant->après->VIVANT
    |    |    +- % CPU passé dans le GC depuis le début
    |    +- temps depuis le démarrage
    +- numéro du cycle
```

Les **deux** pauses STW (ici `0.026` et `0.042` ms) encadrent un **marquage concurrent** (`0.37` ms)
qui tourne **pendant** votre programme. Le triplet `3->4->0 MB` se lit : tas **avant** le GC → tas en
fin de marquage → **vivant** après sweep. 🔁 [Ch. 29](29-observabilite-runtime.md) pour exploiter ces
chiffres en production.

## Fin de vie d'objet : cleanup & pointeurs faibles

Au-delà de la mémoire pure, on veut parfois **réagir** à la collecte (libérer un descripteur, vider un
cache). Trois outils, du meilleur au plus ancien :

```go
// code/ch27-garbage-collector/gc.go
// runtime.AddCleanup (1.24) : exécute un nettoyage quand r devient inatteignable.
func WithCleanup(r *Resource, done chan<- int) {
	runtime.AddCleanup(r, func(id int) { done <- id }, r.ID)
}
```

Et un **cache à références faibles** (`weak.Pointer`, 1.24) : il mémorise sans **empêcher** la collecte.

```go
// Get renvoie la ressource si elle est ENCORE vivante, sinon nil (collectée).
func (c *Cache) Get(id int) *Resource {
	if wp, ok := c.m[id]; ok {
		return wp.Value() // promotion faible -> forte ; nil si déjà collecté
	}
	return nil
}
```

```
$ go run ./ch27-garbage-collector
avant GC, Get(1) != nil : true
après GC, Get(1) == nil : true     <- la référence faible n'a pas retenu l'objet
```

> ⚠️ **`runtime.SetFinalizer`** (l'ancêtre) est **déconseillé** : un finalizer peut **ressusciter**
> l'objet, retarde sa collecte d'un cycle, et se trompe facilement. Préférez **`AddCleanup`** (plusieurs
> par objet, ne ressuscite pas) et **`weak`** pour les caches.

---

## 🆕 Go 1.2x

- **1.24** — **pointeurs faibles** (`weak.Pointer[T]`, `weak.Make`), **`runtime.AddCleanup`** (remplace
  `SetFinalizer`). Le package **`unique`** ([Ch. 31](31-strings-profondeur.md)) s'appuie dessus.
- **1.25** — `runtime.AddCleanup` exécuté de façon **concurrente/parallèle** (cleanups plus rapides) ;
  **`GODEBUG=checkfinalizers=1`** diagnostique les erreurs et **distingue** finalizers et cleanups
  (vérifié : `checkfinalizers: queue: 0 finalizers + 0 cleanups`). GC **Green Tea** en expérimental.
- **1.26** — **Green Tea GC par défaut** : le marquage scanne la mémoire par **spans contigus** (bien
  meilleure localité de cache) plutôt qu'en sautant de pointeur en pointeur — **−10 à −40 %** d'overhead
  GC selon la charge. On peut le désactiver via `GOEXPERIMENT=nogreenteagc` (vérifié accepté sur 1.26.4).
  Le tas est aussi **randomisé** en adresse de base ([Ch. 24](24-runtime-bootstrap.md)).

## ⚠️ Pièges

- **Croire que le GC « compacte »** — il ne déplace pas les objets ; la fragmentation se gère par les
  **size classes** ([Ch. 26](26-allocation-escape.md)), pas par compaction.
- **`SetFinalizer`** pour libérer une ressource critique — non déterministe, peut ne **jamais**
  s'exécuter. Pour fermer un fichier/connexion, utilisez **`defer`** ([Ch. 16](16-defer.md)), pas le GC.
- **Désactiver le GC** (`GOGC=off`) « pour la perf » — l'empreinte mémoire explose. Préférez régler
  `GOGC`/`GOMEMLIMIT`.
- **Oublier `GOMEMLIMIT` en conteneur** — sans lui, le tas peut viser au-delà de la RAM allouée et
  provoquer un **OOM-kill** brutal.

## ⚡ Performance

- Le coût du GC est **proportionnel** à la quantité de **mémoire vivante scannée** (pointeurs), pas à
  la mémoire morte. Moins de pointeurs vivants = GC moins cher → revoir le [Ch. 26](26-allocation-escape.md).
- **`GOMEMLIMIT`** + un `GOGC` modéré est souvent le meilleur réglage serveur : on tient une enveloppe
  mémoire sans trop de cycles.
- **Green Tea** (1.26) améliore la localité du marquage sans rien changer à votre code.
- 🔁 Mesurez l'impact GC avec `gctrace`, `runtime/metrics` ([Ch. 29](29-observabilite-runtime.md)) et
  les traces ([Ch. 38](38-traces-flight-recorder.md)).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch27-garbage-collector
go test ./ch27-garbage-collector/...
GODEBUG=gctrace=1 go run ./ch27-garbage-collector            # chaque cycle de GC
GOGC=20 GODEBUG=gctrace=1 go run ./ch27-garbage-collector    # GC bien plus fréquent
```

À essayer :

1. Comparez le **nombre de cycles** entre `GOGC=20` et `GOGC=400` sur le même programme.
2. Fixez `GOMEMLIMIT=8MiB` et observez le GC devenir agressif sous la contrainte.
3. Remplacez le cache faible par une `map[int]*Resource` (forte) : les objets ne sont **plus** collectés.

---

## 📌 À retenir

- GC **mark-sweep concurrent tri-couleur**, **non compactant**, à **faible latence** : deux courtes
  pauses STW encadrent un marquage qui tourne **avec** votre programme.
- La **write barrier** (active pendant le marquage) maintient l'invariant tri-couleur malgré les
  écritures concurrentes.
- **Pacing** : le tas vise `vivant * (1 + GOGC/100)`. **`GOGC`** arbitre CPU↔mémoire ; **`GOMEMLIMIT`**
  pose une limite douce (indispensable en conteneur).
- **`GODEBUG=gctrace=1`** donne, par cycle : pauses, % CPU, `avant->après->vivant MB`, objectif.
- Fin de vie : **`runtime.AddCleanup`** et **`weak.Pointer`** (1.24) ; **`SetFinalizer` est déconseillé**.
  **Green Tea GC** par défaut en **1.26**.

## 🔁 Pour aller plus loin

- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : allouer moins = moins de GC.
- [Ch. 29 — Observabilité](29-observabilite-runtime.md) : `runtime/metrics`, `MemStats`, monitoring du GC.
- [Ch. 31 — Strings en profondeur](31-strings-profondeur.md) : le package `unique` et les pointeurs faibles.
- [Ch. 38 — Traces](38-traces-flight-recorder.md) : visualiser les cycles de GC dans le temps.
- Doc : `go doc runtime/debug.SetGCPercent`, `go doc weak`, le guide « A Guide to the Go GC » (go.dev).
