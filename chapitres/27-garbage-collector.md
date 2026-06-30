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

Pourquoi ne pas simplement **tout arrêter** le temps de collecter ? C'est l'option la plus simple — un
GC **stop-the-world** intégral : programme figé, donc aucune incohérence possible pendant le parcours
du tas. Mais la pause **croît avec la taille du tas vivant** ; au-delà de quelques mégaoctets retenus,
une pause de plusieurs dizaines voire centaines de millisecondes devient inacceptable pour un service
qui doit répondre vite. Un GC **concurrent** garde le programme actif pendant l'essentiel du travail,
au prix d'un problème nouveau : le programme peut **modifier le graphe d'objets pendant que le GC le
parcourt**. Les deux mécanismes ci-dessous — marquage **tri-couleur** et **write barriers** — existent
uniquement pour résoudre ce problème sans réintroduire de longues pauses.

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

Concrètement, suivons un objet `A` qui pointe vers `B`, sur deux instants du marquage :

```
  Étape 1 — le marqueur atteint A depuis la racine ; B n'a pas encore été vu :

    racine --> +-------+        +--------+
               | A:GRIS|------->| B:BLANC|
               +-------+        +--------+
               (en file grise, pointeurs pas encore parcourus)

  Étape 2 — le marqueur dépile A : il grise B (le pointeur de A vers B est suivi),
  puis noircit A (A n'a plus aucun pointeur à explorer) :

    racine --> +-------+        +-------+
               | A:NOIR|------->| B:GRIS|
               +-------+        +-------+
                                 (en file grise à son tour)

  B suivra exactement le même chemin BLANC -> GRIS -> NOIR. Un objet ne bascule
  JAMAIS directement de BLANC à NOIR : il transite TOUJOURS par GRIS, le temps
  que ses propres pointeurs soient parcourus.
```

Quand la file grise est vide, tout ce qui est encore **blanc** est inatteignable : le sweep le
récupère (paresseusement, au fil des allocations suivantes). Le GC de Go est **non compactant** : il ne
déplace pas les objets vivants (les adresses restent stables pendant la vie de l'objet).

## Les write barriers

Problème concret : pendant le marquage **concurrent**, le programme continue d'écrire des pointeurs.
Imaginons un objet **noir** `P` (déjà scanné, le marqueur le considère terminé) et un objet **gris**
`G` qui détient encore l'unique chemin vers un objet **blanc** `C`. Si le programme exécute
`P.field = C` puis efface aussitôt la référence de `G` (`G.field = nil`), plus aucun chemin gris ou
blanc ne mène à `C` : le marqueur ne revisite **jamais** un objet déjà noirci, donc il ne (re)découvrira
jamais `C`. Pourtant `C` reste atteignable, via `P` — le sweep le récupérerait **à tort**, un objet
vivant disparaîtrait sous les pieds du programme. C'est l'invariant tri-couleur violé : _un objet noir
ne doit jamais devenir le seul chemin vers un objet blanc_.

La **write barrier** empêche exactement ce scénario : à chaque écriture de pointeur **pendant le
marquage**, un petit bout de code inséré par le compilateur grise la cible concernée pour qu'elle
reste dans le périmètre du marqueur. Go utilise depuis la 1.8 une barrière **hybride** : elle grise à
la fois la **nouvelle** valeur écrite et **l'ancienne** valeur remplacée, ce qui maintient l'invariant
dans les deux sens de modification sans avoir besoin de rebalayer les piles des goroutines à la fin du
marquage — un gain net sur la pause STW de terminaison. C'est le même genre de **barrière mémoire** que
celle du [Ch. 25](25-modele-memoire.md), mais au service du GC plutôt que de la synchronisation entre
goroutines : elle n'est active **que** durant la phase de marquage, désactivée le reste du temps.

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
limite plutôt que s'étrangler, mais fait tout pour la tenir. On peut aussi inverser la priorité des
deux molettes : `GOGC=off` **combiné** à un `GOMEMLIMIT` fixé délègue tout le pacing à la limite
mémoire — le GC ne tourne **que** quand le tas s'en approche, ce qui élimine les cycles inutiles quand
l'enveloppe mémoire disponible est connue précisément. ⚠️ Mais `GOGC=off` **seul**, sans
`GOMEMLIMIT`, n'a aucun garde-fou : le tas grossit jusqu'à l'OOM.

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

### Pourquoi un `GOGC` trop bas coûte du CPU : le mark assist

Le coût d'un cycle de marquage est à peu près **proportionnel** à la quantité de mémoire **vivante**
parcourue ([Ch. 26](26-allocation-escape.md)) — réduire `GOGC` ne réduit pas ce travail, il le **répète
simplement plus souvent** sur un tas tout aussi vivant. Pire : si les goroutines **allouent plus vite**
que les workers de marquage en tâche de fond ne progressent, le runtime force les goroutines
allouantes à participer **elles-mêmes** au marquage avant de continuer leur travail — c'est le **mark
assist**. Un `GOGC` agressif combiné à un fort débit d'allocation peut donc **ralentir directement le
code applicatif** (latence côté requête), pas seulement consommer du CPU « en arrière-plan ». La
section `gctrace` ci-dessous montre où lire ce détail (`assist`/`background`/`idle`).

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

Le `...` élidé ci-dessus cache le détail **CPU** du marquage, au format `#+#/#/#+# ms cpu` : temps STW
de sweep + (**assist** / **background** / **idle**) + temps STW de fin de marquage. `assist` est le
temps que les **goroutines allouantes** passent elles-mêmes à marquer (forcées d'y contribuer si
l'allocation va plus vite que le marquage de fond, voir la section pacing plus haut) ; `background`
celui des workers dédiés du GC ; `idle` celui glané sur des P autrement inoccupés. Un `assist` qui
grossit est le signal direct d'un `GOGC` trop bas pour le débit d'allocation réel du programme. Si la
ligne se termine par `(forced)`, le cycle a été déclenché par un appel explicite à `runtime.GC()`.

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
- **1.26** — **Green Tea GC par défaut** : le marquage classique suit les pointeurs un par un, ce qui
  saute d'une adresse mémoire à l'autre **sans rapport de proximité** (défaut de cache fréquent). Green
  Tea scanne plutôt la mémoire par **spans contigus** — les objets d'un même span (souvent alloués au
  même moment, souvent liés) sont traités ensemble, donc déjà chargés en cache. Résultat : **−10 à
  −40 %** d'overhead GC selon la charge, l'effet étant d'autant plus net que le tas contient beaucoup de
  petits objets dispersés. On peut le désactiver via `GOEXPERIMENT=nogreenteagc` (vérifié accepté sur
  1.26.4). Le tas est aussi **randomisé** en adresse de base ([Ch. 24](24-runtime-bootstrap.md)).

## ⚠️ Pièges

- **Croire que le GC « compacte »** — il ne déplace pas les objets ; la fragmentation se gère par les
  **size classes** ([Ch. 26](26-allocation-escape.md)), pas par compaction.
- **`SetFinalizer`** pour libérer une ressource critique — non déterministe, peut ne **jamais**
  s'exécuter. Pour fermer un fichier/connexion, utilisez **`defer`** ([Ch. 16](16-defer.md)), pas le GC.
- **Désactiver le GC** (`GOGC=off`) « pour la perf » — l'empreinte mémoire explose. Préférez régler
  `GOGC`/`GOMEMLIMIT`.
- **`GOGC` trop bas** (ex. `10`) sur un service à fort débit d'allocation — les cycles se multiplient
  et le **mark assist** déborde sur le code applicatif (voir « GC pacing » plus haut) : le CPU grimpe
  sans gain de stabilité mémoire proportionnel.
- **`GOGC` trop haut** (ex. `800`) — le tas est autorisé à grossir bien au-delà du vivant avant le
  prochain cycle : pic mémoire potentiellement important, même si la mémoire réellement **vivante**
  reste petite. Dangereux sur une machine partagée ou en conteneur sans `GOMEMLIMIT`.
- **Oublier `GOMEMLIMIT` en conteneur** — sans lui, le tas peut viser au-delà de la RAM allouée et
  provoquer un **OOM-kill** brutal.

## ⚡ Performance

- Le coût du GC est **proportionnel** à la quantité de **mémoire vivante scannée** (pointeurs), pas à
  la mémoire morte. Moins de pointeurs vivants = GC moins cher → revoir le [Ch. 26](26-allocation-escape.md).
- Le compilateur n'insère une **write barrier** que pour les écritures de pointeur vers de la mémoire
  potentiellement gérée par le tas. Une écriture vers une variable **prouvée** rester sur la pile par
  l'escape analysis ([Ch. 26](26-allocation-escape.md)) n'en a jamais besoin — encore un effet de bord
  positif d'allouer moins.
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
  pose une limite douce (indispensable en conteneur). Un **mark assist** force les goroutines
  allouantes à marquer elles-mêmes si l'allocation devance le marquage de fond — d'où le coût direct
  d'un `GOGC` trop bas sous forte allocation.
- **`GODEBUG=gctrace=1`** donne, par cycle : pauses, % CPU, `avant->après->vivant MB`, objectif.
- Fin de vie : **`runtime.AddCleanup`** et **`weak.Pointer`** (1.24) ; **`SetFinalizer` est déconseillé**.
  **Green Tea GC** par défaut en **1.26**.

## 🔁 Pour aller plus loin

- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : allouer moins = moins de GC.
- [Ch. 29 — Observabilité](29-observabilite-runtime.md) : `runtime/metrics`, `MemStats`, monitoring du GC.
- [Ch. 31 — Strings en profondeur](31-strings-profondeur.md) : le package `unique` et les pointeurs faibles.
- [Ch. 38 — Traces](38-traces-flight-recorder.md) : visualiser les cycles de GC dans le temps.
- Doc : `go doc runtime/debug.SetGCPercent`, `go doc weak`, le guide « A Guide to the Go GC » (go.dev).
