# Ch. 24 — Architecture du runtime & bootstrap

> **Objectif** — Comprendre ce qui s'exécute **avant** et **autour** de votre `main` : le rôle du
> **runtime** (ordonnanceur, GC, allocateur, netpoller) lié à chaque binaire, la séquence de
> **bootstrap** (du `_rt0` de l'OS à `runtime.main`), les goroutines/threads d'amorçage **`g0`/`m0`**,
> et l'**ordre d'initialisation** des packages et des `init()`.
>
> **Prérequis** — [Ch. 2](02-structure-programme.md) (packages, `init`), [Ch. 19](19-goroutines.md) (goroutines)

---

## Introduction

Cette partie **ouvre le capot**. Jusqu'ici nous avons appris à **utiliser** Go ; nous allons
maintenant comprendre **comment il tourne**. Tout commence ici : que se passe-t-il entre le moment où
le système d'exploitation charge votre binaire et le moment où votre `main.main` reçoit la main ?

La réponse tient en un mot : le **runtime**. Contrairement à C (où `main` est quasi le premier code
exécuté) ou à Java (qui démarre une JVM externe), un programme Go embarque **son propre runtime**, lié
**statiquement** dans le binaire ([Ch. 1](01-installation-toolchain.md)). C'est lui qui crée les
goroutines, ordonnance, ramasse les miettes (GC) et gère les entrées/sorties. L'exemple est dans
[`code/ch24-runtime-bootstrap/`](../code/ch24-runtime-bootstrap/).

---

## Le runtime : un programme sous votre programme

Le runtime n'est pas une bibliothèque externe ni un interpréteur : c'est du code Go (et un peu
d'assembleur) **compilé avec le vôtre** et embarqué dans le binaire. Ses grands services :

| Service          | Rôle                                                      | Chapitre                          |
| ---------------- | --------------------------------------------------------- | --------------------------------- |
| **Ordonnanceur** | répartit les goroutines (G) sur les threads (M) via les P | [Ch. 28](28-ordonnanceur-gmp.md)  |
| **Allocateur**   | distribue la mémoire (pile/tas, size classes, `mcache`)   | [Ch. 26](26-allocation-escape.md) |
| **GC**           | récupère la mémoire morte (mark-sweep concurrent)         | [Ch. 27](27-garbage-collector.md) |
| **Netpoller**    | I/O non bloquante (réveille les goroutines en attente)    | [Ch. 28](28-ordonnanceur-gmp.md)  |

C'est pourquoi un « hello world » Go pèse ~1-2 Mo : il **contient** tout cela. En échange, il ne
dépend d'aucune VM ni runtime installé sur la machine cible.

## Du processus OS à `runtime.main`

Quand l'OS charge le binaire, il saute à un point d'entrée en assembleur (`_rt0_arm64_darwin` sur
ce Mac), **pas** à votre `main`. Voici la chaîne d'amorçage :

```
  noyau OS
    |  charge le binaire, saute au point d'entree
    v
  _rt0_<arch>_<os>            asm, specifique OS/architecture
    |
    v
  rt0_go                      installe g0 + m0 (pile systeme, 1er thread OS)
    |
    +--> osinit               NumCPU, taille de page
    +--> schedinit            ordonnanceur, GC, allocateur, GOMAXPROCS
    |
    v
  newproc(runtime.main)       cree la 1re goroutine ORDINAIRE (G1)
    |
    v
  mstart -> schedule          m0 demarre sa boucle et prend G1
    |
    v
  runtime.main   (sur G1)
    +--> demarre sysmon       thread de surveillance, SANS P
    +--> init des packages    ordre de dependance (stdlib puis le votre)
    +--> tous les init()
    +--> main.main            <=== VOTRE code commence ICI
    |
    v
  exit(0)                     quand main.main renvoie
```

Point clé : votre `main.main` s'exécute **très tard**, sur une **goroutine ordinaire** (G1), après que
tout le runtime a été mis en place et que toutes les initialisations sont terminées.

## `g0` et `m0` : les amorces

Deux objets sont **statiquement alloués** au démarrage, avant toute allocation dynamique :

- **`m0`** — le **premier thread OS**. Tous les threads suivants sont créés à la demande par
  l'ordonnanceur ([Ch. 28](28-ordonnanceur-gmp.md)).
- **`g0`** — une goroutine **spéciale**, à **pile fixe et large**, qui exécute le **code du runtime
  lui-même** (ordonnancement, GC, création de goroutines). Chaque M possède son `g0`. Vos goroutines,
  elles, ont une petite pile **croissante** (~2 Kio, [Ch. 19](19-goroutines.md), [Ch. 26](26-allocation-escape.md)).

```
   m0 (thread OS) ---- exécute --> g0     (pile système, code du runtime)
                                    |  bascule pour exécuter du code utilisateur
                                    v
                                   G1 ---> runtime.main ---> main.main
```

Un thread alterne ainsi entre **son `g0`** (pour ordonnancer, déclencher le GC…) et les **goroutines
utilisateur**. Un troisième larron, **`sysmon`**, tourne sur un thread dédié **sans P** : il surveille
la préemption, relance le netpoller et force un GC si besoin.

## Ordre d'initialisation

Avant `main.main`, Go initialise dans cet ordre **strict** :

1. les **packages importés** d'abord (récursivement, en profondeur) ;
2. dans chaque package, les **variables de niveau package**, dans l'ordre de leurs **dépendances** ;
3. puis les **`init()`** du package, dans l'ordre du source ;
4. enfin, `main.main`.

Le point subtil est l'étape 2 : l'ordre suit le **graphe de dépendances**, pas le texte.

```go
// code/ch24-runtime-bootstrap/bootstrap.go
// derived est déclaré AVANT base, mais l'utilise : Go initialise base d'abord.
var derived = track("derived(after " + base + ")")
var base = track("base")

func init() { track("init #1") }
func init() { track("init #2") }
```

```
$ go run ./ch24-runtime-bootstrap
Ordre d'initialisation observé :
  base -> derived(after base) -> init #1 -> init #2
```

`base` s'initialise avant `derived` (dépendance), et les deux **avant** les `init()`. ⚠️ Ne dépendez
**jamais** de l'ordre du _source_ entre variables : seul le graphe de dépendances est garanti
([Ch. 2](02-structure-programme.md)).

## Observer le bootstrap : `GODEBUG=inittrace=1`

Le runtime sait tracer l'init de **chaque** package (stdlib comprise), avec son coût :

```
$ GODEBUG=inittrace=1 go run ./ch24-runtime-bootstrap
init runtime @0.016 ms, 0.030 ms clock, 0 bytes, 0 allocs
init errors  @0.24 ms,  0.002 ms clock, 0 bytes, 0 allocs
init time    @0.78 ms,  0.003 ms clock, 256 bytes, 2 allocs
init os      @0.82 ms,  0.046 ms clock, 7776 bytes, 22 allocs
init main    @4.2 ms,   0.001 ms clock, 320 bytes, 1 allocs
```

Chaque ligne : `@t` = instant depuis le début, `clock` = durée de cet `init`, puis la mémoire allouée.
Un `init()` **lent ou gourmand** ralentit **tout** démarrage (pénalisant pour une CLI ou un _cold
start_ serverless) : `inittrace` le révèle. 🔁 [Ch. 29](29-observabilite-runtime.md) pour les autres
sondes `GODEBUG`.

---

## 🆕 Go 1.2x

- **1.25** — `GOMAXPROCS` devient **conscient des cgroups** : dans un conteneur limité à 2 cœurs,
  `schedinit` fixe `GOMAXPROCS=2` (et non le nombre de cœurs de l'hôte). Réajusté dynamiquement ;
  `runtime.SetDefaultGOMAXPROCS()` réapplique la valeur par défaut. Détail [Ch. 28](28-ordonnanceur-gmp.md).
- **1.26** — le runtime **randomise l'adresse de base du tas** (durcissement contre l'exploitation
  mémoire) ; le **Green Tea GC** est activé par défaut ([Ch. 27](27-garbage-collector.md)). Ces choix
  sont faits dans `schedinit`, avant votre code.

## ⚠️ Pièges

- **`init()` qui fait trop** — connexions réseau, lecture de fichiers, gros calculs : tout cela
  **bloque** le démarrage et n'est pas testable proprement. Préférez une initialisation **paresseuse**
  (`sync.OnceValue`, [Ch. 21](21-synchronisation.md)) ou explicite.
- **Dépendre de l'ordre des `init()` entre fichiers** — il suit l'ordre des fichiers passés au
  compilateur (alphabétique via `go build`). Fragile : ne vous y fiez pas.
- **Effets de bord à l'import** (`import _ "pkg"` pour son `init()`) — pratique pour enregistrer un
  driver, mais rend les dépendances **invisibles**. À utiliser avec parcimonie.

## ⚡ Performance

- Le temps de bootstrap est **dominé par les `init()`** de vos dépendances, pas par le runtime
  lui-même (quelques dizaines de µs). `inittrace=1` chiffre chaque package.
- Un binaire Go est **autonome** : pas de coût de démarrage de VM, mais une **empreinte disque** plus
  grande (runtime embarqué). Compromis assumé.
- 🔁 [Ch. 39](39-compilation-inlining-pgo.md) : ce que le compilateur fait du code du runtime (inlining).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch24-runtime-bootstrap
go test ./ch24-runtime-bootstrap/...
GODEBUG=inittrace=1 go run ./ch24-runtime-bootstrap   # init de TOUS les packages
```

À essayer :

1. Ajoutez une 3ᵉ variable qui dépend de `derived` et vérifiez sa position dans la trace.
2. Comparez l'`inittrace` d'un programme qui importe `net/http` : combien de packages, quel coût ?
3. Mesurez le temps total de bootstrap avec `GODEBUG=inittrace=1` sur un de vos vrais programmes.

---

## 📌 À retenir

- Un binaire Go **embarque son runtime** (ordonnanceur, GC, allocateur, netpoller), lié **statiquement**.
- Le démarrage va de `_rt0_<arch>_<os>` → `rt0_go` (installe **`g0`/`m0`**) → `schedinit` →
  `runtime.main` → **`main.main`**. Votre code s'exécute **en dernier**, sur une goroutine ordinaire.
- **`g0`** (pile large) exécute le **runtime** ; vos goroutines ont une petite pile croissante.
  **`sysmon`** surveille en arrière-plan, sans P.
- L'init suit : **packages importés** → **variables** (ordre des **dépendances**) → **`init()`**
  (ordre du source) → `main`.
- **`GODEBUG=inittrace=1`** chiffre l'init de chaque package — outil clé contre les démarrages lents.

## 🔁 Pour aller plus loin

- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : la pile croissante d'une goroutine.
- [Ch. 27 — Garbage collector](27-garbage-collector.md) : ce que `schedinit` règle pour le GC.
- [Ch. 28 — L'ordonnanceur](28-ordonnanceur-gmp.md) : G-M-P, `sysmon`, `GOMAXPROCS` et cgroups.
- [Ch. 29 — Observabilité](29-observabilite-runtime.md) : toutes les sondes `GODEBUG` et `runtime`.
- Doc : `go doc runtime`, et le fichier `runtime/proc.go` (fonctions `schedinit`, `main`).
