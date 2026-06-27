# Ch. 28 — L'ordonnanceur (G-M-P)

> **Objectif** — Comprendre **comment** vos goroutines tournent : le modèle **G-M-P**, les **files**
> locales et globale, le **work stealing**, le **handoff** sur syscall bloquant, le **netpoller** (I/O
> non bloquante), la **préemption asynchrone**, et le rôle de **`GOMAXPROCS`** (conscient des cgroups
> depuis 1.25).
>
> **Prérequis** — [Ch. 19](19-goroutines.md), [Ch. 20](20-channels-select.md), [Ch. 24](24-runtime-bootstrap.md)

---

## Introduction

Une goroutine n'est **pas** un thread OS ([Ch. 19](19-goroutines.md)) : elle est mille fois plus
légère. Le secret est un **ordonnanceur en espace utilisateur** qui multiplexe des milliers (millions)
de goroutines sur une **poignée** de threads. C'est lui qui permet d'écrire « une goroutine par
connexion » sans écrouler la machine. Ce chapitre ouvre cette mécanique : le modèle **G-M-P**. Code
dans [`code/ch28-ordonnanceur-gmp/`](../code/ch28-ordonnanceur-gmp/).

---

## Le modèle G-M-P

Trois entités, à ne pas confondre :

- **G** — une **goroutine** : le travail à faire (pile, PC, état).
- **M** — un **thread OS** (_machine_) : la seule chose que le noyau sait exécuter.
- **P** — un **processeur logique** : une **ressource d'ordonnancement** (file de goroutines, cache
  d'allocation `mcache`). Leur nombre = **`GOMAXPROCS`**.

**Règle d'or** : pour exécuter du code Go, un **M doit détenir un P**. Le nombre de P **borne** donc le
parallélisme réel.

```
     P0             P1             P2          P = processeur logique (= GOMAXPROCS)
   +------+       +------+       +------+
   |  M   |       |  M   |       |  M   |      M = thread OS (detient un P)
   +------+       +------+       +------+
   |  G   |       |  G   |       |  G   |      G = goroutine en cours d'execution
   +------+       +------+       +------+
   LRQ:           LRQ:           LRQ:          LRQ = file locale (par P)
   [G][G]         [G]            [G][G][G]
                                    |  vol de travail (work stealing)
                                    +--------> P1 pioche ici quand sa file est vide

   GRQ (file globale) : [G][G][G] ...    netpoller : [G en attente d'I/O reseau]
```

## Files locales, file globale, work stealing

Chaque P a une **file locale** (LRQ, ~256 goroutines). Un M y prend la prochaine G à exécuter — **sans
verrou**, donc très vite. Quand sa LRQ est **vide**, il ne reste pas oisif ; il cherche du travail dans
cet ordre :

1. la **file globale** (GRQ) ;
2. le **netpoller** (goroutines dont l'I/O est prête) ;
3. il **vole** (work stealing) la **moitié** de la LRQ d'un **autre** P.

Le work stealing équilibre la charge **automatiquement** : aucun P ne dort pendant qu'un autre croule.
C'est ce qui rend `parallelSum` efficace sans qu'on gère quoi que ce soit :

```go
// code/ch28-ordonnanceur-gmp/scheduler.go
// Réparti sur `workers` goroutines ; le résultat ne dépend PAS du parallélisme.
func parallelSum(nums []int, workers int) int { /* fan-out + combine */ }
```

```
$ go run ./ch28-ordonnanceur-gmp
8 tâches CPU : GOMAXPROCS=1 -> 205ms ; GOMAXPROCS=8 -> 32ms   (exemple, dépend de la machine)
```

## Syscalls bloquants : le handoff

Quand une goroutine fait un **appel système bloquant** (lecture disque, par exemple), le **M** se
bloque avec elle dans le noyau. Si rien n'était fait, son P serait gelé. L'ordonnanceur effectue donc
un **handoff** : il **détache le P** du M bloqué et le confie à **un autre M** (réveillé ou créé), qui
continue d'exécuter les autres goroutines.

```
  M1 [P0] --- syscall bloquant ---> noyau          (M1 reste bloqué)
        \
         '--> P0 remis a M2 ------> M2 [P0] continue d'executer les G de la LRQ

  Au retour du syscall, M1 tente de reprendre un P ; sinon sa G part en GRQ et M1 se gare.
```

C'est pourquoi le **nombre de threads** (M) peut **dépasser** `GOMAXPROCS` : il y a un M par syscall
bloquant en cours (visible dans `schedtrace`, `threads=9` plus bas).

## Le netpoller : l'I/O sans bloquer un thread

Pour l'**I/O réseau** (et les timers), Go ne bloque **pas** un M. L'opération est enregistrée auprès du
**poller du noyau** (`kqueue` sur macOS, `epoll` sur Linux) ; la goroutine est **garée** ; quand l'I/O
est prête, le **netpoller** la rend de nouveau exécutable. Un seul thread peut ainsi gérer des
**milliers** de connexions en attente — le fondement des serveurs Go ([Ch. 20](20-channels-select.md),
projets 2 et 5).

## Préemption asynchrone

Depuis Go **1.14**, l'ordonnanceur peut **préempter** une goroutine même au milieu d'une **boucle
serrée** sans appel de fonction (via un signal). Avant, une goroutine qui ne « cédait » jamais pouvait
**monopoliser** un P. Aujourd'hui, **`sysmon`** ([Ch. 24](24-runtime-bootstrap.md)) repère une G qui
tourne depuis trop longtemps (>10 ms) et la fait préempter. `runtime.Gosched()` permet **en plus** de
céder la main **volontairement** :

```go
// code/ch28-ordonnanceur-gmp/scheduler.go
if i%1_000_000 == 0 {
	runtime.Gosched() // point de coopération explicite (rarement nécessaire)
}
```

## Observer : `GODEBUG=schedtrace`

```
$ GODEBUG=schedtrace=250 ./prog
SCHED 0ms:   gomaxprocs=8 idleprocs=6 threads=3 ... runqueue=0 [ 0 0 0 0 0 0 0 0 ]
SCHED 253ms: gomaxprocs=8 idleprocs=0 threads=9 ... runqueue=6 [ 0 1 0 0 0 0 1 0 ]
                        |          |          |             |     |
                        |          |          |             |     +- LRQ par P (files locales)
                        |          |          |             +- GRQ (file globale)
                        |          |          +- M (threads) : > GOMAXPROCS si syscalls
                        |          +- P inactifs
                        +- nombre de P
```

À `253ms`, les 8 P sont **occupés** (`idleprocs=0`), il y a **9 threads** (un de plus que les P : un
syscall en cours), et **6** goroutines patientent en file globale. Cette photographie révèle
contention et déséquilibre. 🔁 [Ch. 29](29-observabilite-runtime.md) pour les métriques `/sched`.

## `GOMAXPROCS` et les conteneurs

`GOMAXPROCS` vaut par défaut le nombre de **cœurs logiques**. On le lit/modifie par programme :

```go
// code/ch28-ordonnanceur-gmp/scheduler.go
func WithGOMAXPROCS(n int, f func()) {
	old := runtime.GOMAXPROCS(n)
	defer runtime.GOMAXPROCS(old)
	f()
}
```

---

## 🆕 Go 1.2x

- **1.25** — `GOMAXPROCS` devient **conscient des cgroups** : dans un conteneur limité (ex. 2 cœurs sur
  un hôte 64 cœurs), le runtime fixe `GOMAXPROCS=2` au lieu de 64 — fini la sur-souscription qui faisait
  s'effondrer les services conteneurisés. La valeur est **réajustée dynamiquement** si la limite change.
- **1.25** — **`runtime.SetDefaultGOMAXPROCS()`** réapplique la valeur calculée par défaut (utile après
  l'avoir forcée). Vérifié présent sur 1.26.4.
- 🔁 La préemption asynchrone (**1.14**) et le netpoller sont stables ; ce chapitre décrit l'état 1.26.

## ⚠️ Pièges

- **Croire que `GOMAXPROCS` = nombre de goroutines** — non : c'est le **plafond de parallélisme**. On
  peut avoir un million de goroutines pour 8 P.
- **Forcer `GOMAXPROCS=1` « pour éviter les races »** — ça ne supprime **pas** les data races (elles
  restent un bug, [Ch. 25](25-modele-memoire.md)), ça masque juste leur probabilité.
- **Bloquer un M avec du C/cgo** — un appel cgo long bloque un thread sans handoff propre ; à surveiller
  ([Ch. 35](35-unsafe-cgo.md)).
- **Sur-souscrire en conteneur** (avant 1.25, ou en forçant `GOMAXPROCS`) — trop de P pour le quota CPU
  réel = changements de contexte coûteux. Laissez le défaut 1.25 faire son travail.

## ⚡ Performance

- Pour du **CPU pur**, le bon nombre de workers ≈ **`GOMAXPROCS`** ; au-delà, on n'accélère plus, on
  ajoute du surcoût ([Ch. 23](23-patterns-concurrence.md)).
- Pour de l'**I/O**, on peut lancer **beaucoup plus** de goroutines : en attente réseau, elles ne
  consomment **aucun** P (netpoller).
- L'ordonnanceur favorise la **localité** (une G réveillée par une autre tend à rester sur le même P) :
  bon pour le cache. Le work stealing ne se déclenche qu'en cas de déséquilibre.
- 🔁 Visualisez l'ordonnanceur dans le temps avec les **traces** ([Ch. 38](38-traces-flight-recorder.md)).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch28-ordonnanceur-gmp
GODEBUG=schedtrace=200 go run ./ch28-ordonnanceur-gmp   # photo de l'ordonnanceur
go test ./ch28-ordonnanceur-gmp/...
```

À essayer :

1. Faites varier le nombre de workers de `parallelSum` (1, 4, 8, 64) et chronométrez : où plafonne le gain ?
2. Lancez un programme à I/O réseau sous `schedtrace` et observez `threads` rester bas malgré 1000 goroutines.
3. Comparez `GOMAXPROCS=1` et le défaut sur la démo CPU ; le ratio approche-t-il le nombre de cœurs ?

---

## 📌 À retenir

- **G** (goroutine) / **M** (thread OS) / **P** (processeur logique, = `GOMAXPROCS`). Un **M doit tenir
  un P** pour exécuter du Go : les **P bornent le parallélisme**.
- Chaque P a une **file locale** (sans verrou) ; un M oisif pioche en **file globale**, dans le
  **netpoller**, puis **vole** la moitié d'une autre file (**work stealing**).
- Un **syscall bloquant** déclenche un **handoff** du P → le nombre de **threads peut dépasser**
  `GOMAXPROCS`. L'**I/O réseau** passe par le **netpoller** : pas de thread bloqué.
- La **préemption asynchrone** (1.14) empêche une boucle serrée de monopoliser un P ; **`sysmon`** veille.
- Depuis **1.25**, `GOMAXPROCS` est **conscient des cgroups** ; `GODEBUG=schedtrace` photographie l'état.

## 🔁 Pour aller plus loin

- [Ch. 19 — Goroutines](19-goroutines.md) : le coût d'une goroutine, les fuites.
- [Ch. 24 — Bootstrap](24-runtime-bootstrap.md) : `g0`/`m0`, `sysmon`, `schedinit`.
- [Ch. 29 — Observabilité](29-observabilite-runtime.md) : métriques `/sched/*`, `NumGoroutine`.
- [Ch. 38 — Traces](38-traces-flight-recorder.md) : voir P, M et goroutines évoluer dans le temps.
- Doc : `go doc runtime.GOMAXPROCS` ; le document de design « Scalable Go Scheduler » (Dmitry Vyukov).
