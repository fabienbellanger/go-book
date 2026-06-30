# 24 — Architecture du runtime & bootstrap

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

La réponse tient en un mot : le **runtime**. En C, un `_start` minimal fourni par la libc (le
`crt0`) initialise la pile et les arguments puis saute quasi directement dans `main` — il n'y a rien
à ordonnancer. En Java, c'est l'inverse : une JVM **externe**, installée séparément sur la machine
cible, charge et exécute le bytecode. Go choisit une troisième voie : un programme Go embarque **son
propre runtime**, lié **statiquement** dans le binaire ([Ch. 1](01-installation-toolchain.md)) — ni
aussi nu que le C, ni externe comme la JVM. C'est lui qui crée les goroutines, ordonnance, ramasse
les miettes (GC) et gère les entrées/sorties. L'exemple est dans
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

Deux étapes méritent d'être déroulées. D'abord, `osinit` s'exécute **avant** `schedinit` parce que ce
dernier en a besoin : impossible de dimensionner l'ordonnanceur (nombre de `P`, `GOMAXPROCS`) ou
l'allocateur (taille de page) sans connaître la machine sur laquelle on tourne. Ensuite,
`newproc(runtime.main)` ne lance **rien** dans l'immédiat : elle crée la goroutine G1 — avec sa
propre pile, petite et croissante, allouée exactement comme pour n'importe quelle goroutine — et la
dépose dans la file des goroutines exécutables. C'est `mstart`, en entrant dans la boucle `schedule`,
qui vient la chercher et l'exécute réellement : le même mécanisme que pour une goroutine que **vous**
créez avec `go f()` ([Ch. 28](28-ordonnanceur-gmp.md)). `runtime.main` n'est donc pas un cas spécial
pour l'ordonnanceur — seul son **contenu** (démarrer `sysmon`, initialiser les packages, appeler
`main.main`) l'est.

Point clé : votre `main.main` s'exécute **très tard**, sur une **goroutine ordinaire** (G1), après que
tout le runtime a été mis en place et que toutes les initialisations sont terminées.

## `g0` et `m0` : les amorces

Deux objets sont **statiquement alloués** au démarrage, avant toute allocation dynamique :

- **`m0`** — le **premier thread OS**. Tous les threads suivants sont créés à la demande par
  l'ordonnanceur ([Ch. 28](28-ordonnanceur-gmp.md)).
- **`g0`** — une goroutine **spéciale**, à **pile fixe et large**, qui exécute le **code du runtime
  lui-même** (ordonnancement, GC, création de goroutines). Chaque M possède son `g0`. Vos goroutines,
  elles, ont une petite pile **croissante** (~2 Kio, [Ch. 19](19-goroutines.md), [Ch. 26](26-allocation-escape.md)).

Pourquoi une pile **fixe** pour `g0`, alors que toutes les autres goroutines ont une pile qui grandit
à la demande ? Parce que c'est justement du code exécuté **sur `g0`** qui implémente le mécanisme de
croissance des piles (copier le contenu d'une pile trop petite vers une plus grande,
[Ch. 26](26-allocation-escape.md)). Si la pile de `g0` devait grandir par ce même mécanisme, il
faudrait qu'elle grandisse pour exécuter le code qui la fait grandir — une dépendance circulaire que
le runtime refuse explicitement : tenter de faire croître la pile de `g0` déclenche un arrêt fatal
(`morestack on g0`). Le runtime évite donc le problème à la racine en donnant à `g0` une pile large,
dimensionnée une fois pour toutes et qui ne bouge **jamais**. C'est pour cette même raison que
**votre** code ne s'exécute jamais sur `g0` : il lui faut une pile capable de grandir, donc une
goroutine ordinaire — d'où la bascule `g0` → G1 du schéma précédent.

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

Pourquoi ce sens **unique**, du plus profond du graphe d'imports vers `main` ? Parce que le code d'un
package — ses initialiseurs de variables comme ses `init()` — peut appeler des fonctions de ses
imports. Pour que cet appel soit sûr, le package importé doit déjà être **entièrement** prêt
(variables et `init()` exécutés) ; sans cette garantie, on risquerait d'appeler du code qui s'appuie
sur un état encore à zéro. C'est aussi pour que cet ordre reste **toujours calculable** que Go
**interdit les cycles d'imports** ([Ch. 12](12-packages-modules.md)) : un cycle rendrait la question
« qui doit être prêt avant qui ? » sans réponse, et le programme ne compilerait pas.

À l'échelle d'un programme entier, cette règle s'applique en cascade sur tout le graphe d'imports :

```
  main importe service ; service importe db ; db importe errors

  errors    (aucun import : initialise en premier)
    |
    v
  db        (importe errors, deja initialise)
    |
    v
  service   (importe db, deja initialise)
    |
    v
  main      (importe service, deja initialise)
    |
    v
  main.main()
```

Si deux packages **indépendants** (ni l'un n'importe l'autre) sont tous deux prêts à être initialisés
au même moment, l'ordre entre eux n'est pas laissé au hasard pour autant : la spécification du
langage tranche par l'**ordre lexicographique du chemin d'import**, pas par l'ordre des lignes
`import` dans le source. ⚠️ Ne vous fiez donc jamais à l'ordre d'apparition des `import` pour deviner
quel package s'initialise en premier.

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

Pourquoi les `init()`, eux, suivent simplement l'**ordre du texte** et non un graphe de dépendances ?
Parce qu'une fonction n'a pas de syntaxe pour déclarer « j'ai besoin que celle-ci ait tourné avant
moi » — contrairement à une variable, dont l'initialiseur référence explicitement celles dont elle
dépend. Le langage tranche alors par la règle la plus simple et la plus prévisible : l'ordre
d'apparition dans le source, fichier par fichier, dans l'ordre où le compilateur les reçoit.

⚠️ La détection des dépendances entre variables est elle-même purement **lexicale** : le compilateur
regarde quels identifiants sont _référencés_ dans le texte, pas ce qui se passe réellement à
l'exécution. Une dépendance **cachée** derrière un appel de méthode sur une interface, par exemple,
échappe à cette analyse — l'ordre d'initialisation entre les variables concernées devient alors **non
spécifié** par le langage. C'est rare en pratique, mais c'est une raison de plus pour garder les
initialiseurs de variables de package **simples et directs**.

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
- **Lire un flag dans `init()`** — `flag.Parse()` n'est en pratique appelé que dans `main`, **après**
  toutes les phases d'init. Un `init()` qui lit la valeur d'un `flag.String(...)` y trouve donc
  systématiquement la valeur **par défaut**, jamais celle passée en ligne de commande. Si un réglage
  doit dépendre des flags, lisez-le depuis `main`, pas depuis `init()`.
- **Goroutine lancée depuis un `init()`** — la spécification l'autorise explicitement : elle continue
  de tourner **en parallèle** du reste de l'initialisation, puis de `main.main`. Mais quand
  `main.main` retourne, le programme se termine **aussitôt**, sans attendre les goroutines non
  `main` ([Ch. 19](19-goroutines.md)) : une tâche de fond démarrée trop tôt dans un `init()` peut donc
  ne jamais aller à son terme.
- **`panic` dans un `init()`** — à ce stade, il n'y a ni `main` ni gestionnaire applicatif : le
  programme s'arrête immédiatement avec une trace de pile, avant d'avoir produit la moindre sortie
  utile. Un `recover()` placé dans l'`init()` lui-même peut l'intercepter, mais aucun mécanisme de
  reprise extérieur n'existe à ce stade du démarrage.

## ⚡ Performance

- Le temps de bootstrap est **dominé par les `init()`** de vos dépendances, pas par le runtime
  lui-même (quelques dizaines de µs). `inittrace=1` chiffre chaque package.
- Un binaire Go est **autonome** : pas de coût de démarrage de VM, mais une **empreinte disque** plus
  grande (runtime embarqué). Compromis assumé.
- **`go test` paie ce coût à chaque fois** — un binaire de test est un programme à part entière : il
  réexécute l'init de **toutes** ses dépendances à chaque lancement, par package testé. Sur un module
  à nombreux packages avec des `init()` coûteux, ce coût se répète à chaque `go test ./...`, alors
  qu'un binaire de production ne le paie qu'une fois par démarrage.
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
  (ordre du source) → `main`. Entre packages indépendants, les ex æquo se tranchent par **chemin
  d'import**, jamais par l'ordre des `import` dans le source.
- **`GODEBUG=inittrace=1`** chiffre l'init de chaque package — outil clé contre les démarrages lents.
- `main.main` qui retourne **termine le programme aussitôt**, sans attendre les goroutines non `main`
  — y compris celles lancées depuis un `init()`.

## 🔁 Pour aller plus loin

- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : la pile croissante d'une goroutine.
- [Ch. 27 — Garbage collector](27-garbage-collector.md) : ce que `schedinit` règle pour le GC.
- [Ch. 28 — L'ordonnanceur](28-ordonnanceur-gmp.md) : G-M-P, `sysmon`, `GOMAXPROCS` et cgroups.
- [Ch. 29 — Observabilité](29-observabilite-runtime.md) : toutes les sondes `GODEBUG` et `runtime`.
- Doc : `go doc runtime`, et le fichier `runtime/proc.go` (fonctions `schedinit`, `main`).
