# 28 — L'ordonnanceur (G-M-P)

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
de goroutines sur une **poignée** de threads — un modèle dit **M:N** (M goroutines réparties sur N
threads OS), par opposition au modèle **1:1** d'un thread par tâche concurrente. La raison est
arithmétique : créer un thread passe par un appel système et réserve plusieurs mégaoctets de pile, et
le **changer** en cours d'exécution coûte un aller-retour dans le noyau — de l'ordre de la
microseconde, tout compris. Multiplier ce coût par cent mille connexions simultanées écroulerait
n'importe quelle machine. Changer de goroutine, au contraire, ne quitte jamais l'espace utilisateur :
c'est l'échange de quelques registres et pointeurs de pile, sans appel système ([Ch. 19](19-goroutines.md)
chiffre le coût de création, du même ordre de grandeur). C'est ce qui permet d'écrire « une goroutine
par connexion » sans écrouler la machine. Ce chapitre ouvre cette mécanique : le modèle **G-M-P**. Code
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

### Pourquoi un P, et pas seulement G et M ?

Le modèle G-M-P n'est pas la première version de l'ordonnanceur Go : jusqu'à Go **1.1** (2013), il
n'y avait que **G** et **M**. Tous les threads piochaient dans une **unique file globale**, protégée
par un **seul verrou** — sous charge (beaucoup de cœurs, beaucoup de goroutines), ce verrou devenait
le goulot d'étranglement : chaque décision d'ordonnancement le contestait, sur tous les cœurs à la
fois. Le **P**, introduit par le « Scalable Go Scheduler » de Dmitry Vyukov, règle ce problème et en
résout un second au passage :

- **Une file par P, sans verrou** — un M qui pioche dans la LRQ de **son** P ne dispute aucune
  ressource partagée : la contention disparaît dans le cas commun (la file globale ne sert plus qu'en
  dernier recours, voir plus bas).
- **`GOMAXPROCS` P, ni plus ni moins** — c'est le **seul** levier qui borne le parallélisme réel.
  Qu'il y ait 1 ou 1 million de G, et quel que soit le nombre de M (variable, un par syscall bloquant
  en cours — voir plus bas), jamais plus de `GOMAXPROCS` G ne s'exécutent **au même instant**.
- **Le cache d'allocation (`mcache`) vit sur le P**, pas sur le M ([Ch. 26](26-allocation-escape.md)) :
  comme un P n'est jamais utilisé par deux M à la fois, ses allocations rapides n'ont pas besoin de
  verrou non plus.

Le P est donc moins « un processeur » qu'un **jeton de droit d'exécuter du Go** : un M qui n'en
détient pas ne peut faire tourner aucun code utilisateur, même s'il tourne réellement sur un cœur
disponible.

## Files locales, file globale, work stealing

Chaque P a une **file locale** (LRQ, ~256 goroutines). Un M y prend la prochaine G à exécuter — **sans
verrou**, donc très vite. Quand sa LRQ est **vide**, il ne reste pas oisif ; il cherche du travail dans
cet ordre :

1. la **file globale** (GRQ) ;
2. le **netpoller** (goroutines dont l'I/O est prête) ;
3. il **vole** (work stealing) la **moitié** de la LRQ d'un **autre** P.

Même quand sa LRQ **n'est pas vide**, un M consulte malgré tout la GRQ une fois toutes les **61**
prises de décision : sans ce garde-fou périodique, une G qui attend en file globale pourrait être
indéfiniment **affamée** par un flux ininterrompu de travail local. C'est un compromis assumé entre
performance (le cas courant reste sans verrou) et équité (rien n'attend éternellement).

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

Ce mécanisme couvre tout appel bloquant pris en charge par le runtime : lecture/écriture disque,
résolution DNS non asynchrone, ou appel **cgo** (voir le piège plus bas) — mais pas l'I/O réseau
standard, qui passe par le netpoller (section suivante) sans jamais bloquer de M.

C'est pourquoi le **nombre de threads** (M) peut **dépasser** `GOMAXPROCS` : il y a un M par syscall
bloquant en cours (visible dans `schedtrace`, `threads=9` plus bas).

Un M qui se retrouve sans P à la fin d'un syscall n'est pas **détruit** : il se **gare** (park), prêt
à être réveillé pour le prochain handoff, plutôt que de payer de nouveau le coût de création d'un
thread OS. Le runtime ne crée des M qu'à la demande, mais n'en détruit quasiment jamais — un pic
ponctuel de syscalls bloquants laisse donc une trace **durable** dans le nombre de threads du
processus, visible dans `schedtrace` ou `ps -T` longtemps après le pic.

## Le netpoller : l'I/O sans bloquer un thread

Pour l'**I/O réseau** (et les timers), Go ne bloque **pas** un M. L'opération est enregistrée auprès du
**poller du noyau** (`kqueue` sur macOS/BSD, `epoll` sur Linux, `IOCP` sur Windows) ; la goroutine est **garée** ; quand l'I/O
est prête, le **netpoller** la rend de nouveau exécutable. Un seul thread peut ainsi gérer des
**milliers** de connexions en attente — le fondement des serveurs Go ([Ch. 20](20-channels-select.md),
projets 2 et 5).

## Préemption asynchrone

Depuis Go **1.14**, l'ordonnanceur peut **préempter** une goroutine même au milieu d'une **boucle
serrée** sans appel de fonction. **Avant** 1.14, la préemption était uniquement **coopérative** : le
compilateur insérait un point de vérification à chaque **appel de fonction** (là où il vérifiait déjà
s'il fallait agrandir la pile). Une boucle sans aucun appel — un calcul pur sur un tableau, par exemple
— ne traversait jamais ce point et pouvait **monopoliser** son P indéfiniment ; un bug réel avant 1.14,
documenté dans le suivi des problèmes du projet sous le nom « tight loops should be preemptible ».

La préemption **asynchrone** lève cette limite via un **signal** : **`sysmon`**
([Ch. 24](24-runtime-bootstrap.md)) repère une G qui tourne depuis trop longtemps (**> 10 ms**) et
envoie `SIGURG` (sur Unix) au thread M qui l'exécute — un signal choisi car rarement utilisé par les
applications et ignoré par défaut. Le gestionnaire de signal du runtime n'interrompt **pas** n'importe
où : il vérifie que l'instruction courante est à un **point sûr** (hors d'une section critique du
runtime, comme un changement de pile ou une opération atomique) avant de basculer l'ordonnanceur ; la
goroutine reprendra exactement où elle en était. `runtime.Gosched()` permet **en plus** de céder la
main **volontairement** :

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
contention et déséquilibre. Pour un dump complet (état détaillé de **chaque** M, P et G), ajoutez
`scheddetail=1` (`GODEBUG=schedtrace=250,scheddetail=1`) : très verbeux, réservé au diagnostic
ponctuel. 🔁 [Ch. 29](29-observabilite-runtime.md) pour les métriques `/sched`.

## `GOMAXPROCS` et les conteneurs

`GOMAXPROCS` vaut par défaut le nombre de **cœurs logiques** — ça n'a pas toujours été le cas : avant
Go **1.5** (2015), le défaut était **1** (un seul P, donc un vrai mono-thread sauf parallélisme
explicite). Le passage à `NumCPU()` par défaut a fait du parallélisme la **norme implicite** d'un
programme Go ; 1.25 affine encore ce défaut pour les conteneurs (ci-dessous). On le lit/modifie par
programme :

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
- **Bloquer un M avec du C/cgo** — un appel cgo se traite comme un syscall bloquant : son P est rendu
  disponible pour un autre M, comme n'importe quel handoff. Mais le **thread** qui exécute le code C,
  lui, reste occupé jusqu'au retour de l'appel — le runtime ne peut ni l'interrompre ni le préempter,
  faute de visibilité sur du code étranger. Des appels C longs, multipliés sur plusieurs goroutines,
  font donc croître le nombre de threads tout comme des syscalls lents ([Ch. 35](35-unsafe-cgo.md)).
- **Le nombre de M n'est pas illimité** — au-delà de **10 000 threads** simultanés (limite par défaut,
  ajustable via `debug.SetMaxThreads`), le programme s'arrête sur un fatal `thread exhaustion`. Une
  goroutine par requête, chacune faisant un appel bloquant non couvert par le netpoller (fichier, cgo),
  peut s'en approcher sous forte charge : signe qu'il faut **borner** la concurrence (pool de workers,
  sémaphore) plutôt que laisser le runtime créer un M par appel bloquant.
- **Sur-souscrire en conteneur** (avant 1.25, ou en forçant `GOMAXPROCS`) — un cgroup limité à 2 cœurs
  sur un hôte 64 cœurs **voit quand même** 64 cœurs via `NumCPU()` ; avec `GOMAXPROCS=64`, le runtime
  programme plus de travail parallèle que le quota CPU n'en autorise. Le noyau **throttle** alors le
  cgroup (CFS bandwidth control) une fois le quota de la période épuisé : le service se fige par
  à-coups jusqu'à la période suivante, plus visible (latences en dents de scie) qu'un simple surcoût de
  changement de contexte. Laissez le défaut 1.25 faire son travail.

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
