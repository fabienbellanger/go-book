# Annexe A — Glossaire

> **Objectif** — Définir, de façon brève et précise, les termes du langage et du
> runtime Go employés tout au long du livre. Chaque entrée renvoie, quand c'est
> utile, au chapitre qui la développe.

---

Les définitions sont volontairement courtes : un repère, pas un cours. Le chapitre
indiqué par 🔁 donne le détail.

### A

- **Allocation** — Réservation de mémoire pour une valeur. Sur la **pile** elle est
  quasi gratuite (libérée au retour de fonction) ; sur le **tas** elle a un coût et
  pèse sur le GC. 🔁 voir Ch. 26.
- **Atomic** (`sync/atomic`) — Opérations indivisibles (lecture, écriture,
  `Add`, `CompareAndSwap`) sur un entier ou un pointeur, sans verrou. Les types
  `atomic.Int64`, `atomic.Pointer[T]`… garantissent l'atomicité et un ordre
  mémoire. 🔁 voir Ch. 21.
- **AddCleanup** (`runtime.AddCleanup`) — Mécanisme moderne (remplaçant
  recommandé de `SetFinalizer`) pour exécuter une fonction quand un objet devient
  inatteignable. 🔁 voir Ch. 27.

### B

- **Build tags** (contraintes de build) — Commentaires `//go:build` en tête de
  fichier qui conditionnent sa compilation (OS, architecture, version, étiquettes
  personnalisées). 🔁 voir Ch. 12.
- **Bucket** — Compartiment interne d'une map contenant un petit groupe de
  paires clé/valeur. 🔁 voir Ch. 32.
- **byte** — Alias de `uint8`. Une chaîne est une suite immuable d'octets ; un
  `[]byte` en est la version modifiable. 🔁 voir Ch. 31.

### C

- **cgo** — Pont permettant d'appeler du C depuis Go (et inversement). Puissant
  mais coûteux (transition de pile, perte d'inlining) et ennemi de la portabilité.
  🔁 voir Ch. 35.
- **Channel** (`chan T`) — Canal typé de communication entre goroutines, avec ou
  sans tampon. « Ne communiquez pas en partageant la mémoire ; partagez la mémoire
  en communiquant. » 🔁 voir Ch. 20.
- **Closure** (fermeture) — Fonction anonyme qui capture des variables de son
  contexte englobant, par référence. 🔁 voir Ch. 15.
- **Contrainte** (constraint) — Interface qui limite les types acceptés par un
  paramètre de type générique (méthodes et/ou **type set**). 🔁 voir Ch. 11.

### D

- **Data race** (course de données) — Deux goroutines accèdent à la même mémoire
  « en même temps » dont au moins une en écriture, sans synchronisation. Bug non
  déterministe, détecté par `go test -race`. 🔁 voir Ch. 23.
- **Deadlock** (interblocage) — Situation où des goroutines s'attendent mutuellement
  sans pouvoir progresser. Le runtime panique s'il détecte que **toutes** les
  goroutines sont bloquées. 🔁 voir Ch. 20.
- **defer** — Diffère l'exécution d'un appel jusqu'au retour de la fonction
  englobante ; les `defer` s'empilent (dernier entré, premier sorti). 🔁 voir Ch. 16.

### E

- **eface** (empty interface) — Représentation interne d'une interface vide
  (`any`) : une paire (type, pointeur vers la donnée). 🔁 voir Ch. 33.
- **Escape analysis** — Analyse du compilateur déterminant si une valeur peut
  rester sur la pile ou doit « s'échapper » vers le tas. `go build -gcflags=-m`
  l'expose. 🔁 voir Ch. 26.

### F

- **Finalizer** (`runtime.SetFinalizer`) — Fonction appelée par le GC avant de
  récupérer un objet. Fragile et déconseillé ; préférer `runtime.AddCleanup`.
  🔁 voir Ch. 27.
- **FlightRecorder** (`runtime/trace`, 🆕 1.25) — Enregistreur de vol : garde en
  mémoire une fenêtre glissante de la trace d'exécution, que l'on fige dans un
  fichier juste après un évènement rare (ex. requête lente). 🔁 voir Ch. 38.
- **Fuite de goroutine** — Goroutine qui ne se termine jamais (bloquée sur un
  canal, une lecture réseau…), retenant sa pile et ses captures. Cause fréquente de
  fuite mémoire. 🔁 voir Ch. 23.

### G

- **GC tricolore** — Ramasse-miettes concurrent à marquage tricolore (blanc =
  candidat, gris = à explorer, noir = atteignable et exploré). Concurrent et à
  faible pause. 🔁 voir Ch. 27.
- **Génériques** (types/fonctions paramétrés) — Code paramétré par des **types**
  (`func Max[T cmp.Ordered](a, b T) T`), vérifié à la compilation. 🔁 voir Ch. 11.
- **GMP** — Modèle d'ordonnancement de Go : **G** (goroutine), **M** (thread OS,
  _machine_), **P** (processeur logique, _processor_, jeton d'exécution). Un M doit
  détenir un P pour exécuter une G. 🔁 voir Ch. 28.
- **GODEBUG** — Variable d'environnement activant des diagnostics runtime
  (`gctrace=1`, `schedtrace=1000`…) et des bascules de compatibilité. 🔁 voir Ch. 29.
- **GOGC** — Règle l'agressivité du GC : pourcentage de croissance du tas avant
  déclenchement (100 par défaut). 🔁 voir Ch. 27.
- **GOMAXPROCS** — Nombre maximal de P, donc de goroutines exécutées en
  parallèle. Par défaut le nombre de cœurs (ou la limite CPU du conteneur,
  🆕 1.25). 🔁 voir Ch. 28.
- **GOMEMLIMIT** — Limite souple de mémoire totale que le runtime vise à ne pas
  dépasser, en ajustant le GC. 🔁 voir Ch. 27.
- **Goroutine** — Fil d'exécution léger géré par le runtime (pile initiale de
  quelques kio, extensible). On en lance des centaines de milliers sans peine.
  🔁 voir Ch. 19.

### H

- **Happens-before** — Relation d'ordre du **modèle mémoire** garantissant qu'une
  écriture est visible par une lecture. Établie par les canaux, les mutex, `sync`,
  les atomiques. 🔁 voir Ch. 25.
- **hmap** — Structure interne d'en-tête d'une map (compteur, buckets, graine de
  hachage…). 🔁 voir Ch. 32.

### I

- **iface** — Représentation interne d'une interface **non vide** : une paire
  (`itab`, pointeur vers la donnée). 🔁 voir Ch. 33.
- **Inlining** — Optimisation insérant le corps d'une petite fonction sur le site
  d'appel, supprimant le coût de l'appel et ouvrant d'autres optimisations.
  🔁 voir Ch. 26.
- **Interface** — Ensemble de méthodes ; un type la satisfait **implicitement**.
  Valeur = (type dynamique, valeur dynamique). 🔁 voir Ch. 9.
- **iota** — Compteur prédéclaré valant l'indice de la spec courante dans un bloc
  `const`, idéal pour les énumérations. 🔁 voir Ch. 3.
- **Itérateur** (`iter.Seq[T]`, `iter.Seq2[K,V]`, 🆕 1.23) — Fonction de
  _range-over-func_ : `func(yield func(T) bool)`, parcourue par `for x := range`.
  🔁 voir Ch. 18.
- **itab** (interface table) — Table associant un type concret à une interface :
  type dynamique + pointeurs vers les implémentations de méthodes. 🔁 voir Ch. 33.

### M

- **Map** — Table de hachage intégrée (`map[K]V`). Depuis Go 1.24, implémentation
  fondée sur les **Swiss Tables**. Non sûre en accès concurrent. 🔁 voir Ch. 32.
- **mark/sweep** — Les deux phases du GC : **marquage** des objets atteignables,
  puis **balayage** (récupération) des autres. 🔁 voir Ch. 27.
- **mcache / mcentral / mheap** — Hiérarchie d'allocation du runtime : cache par
  P (`mcache`, sans verrou), réserve centrale par classe de taille (`mcentral`),
  tas global (`mheap`). 🔁 voir Ch. 27.
- **Modèle mémoire** — Spécification (`go.dev/ref/mem`) des garanties de
  visibilité des écritures entre goroutines. 🔁 voir Ch. 25.
- **Module** — Unité de versionnement et de dépendances, définie par `go.mod`
  (chemin du module + version de Go + dépendances). 🔁 voir Ch. 12.
- **mspan / span** — Plage contiguë de pages mémoire gérée par le runtime,
  découpée en objets d'une même **classe de taille**. 🔁 voir Ch. 27.
- **Mutex** (`sync.Mutex`) — Verrou d'exclusion mutuelle ; un seul détenteur à la
  fois. Ne pas copier après usage. 🔁 voir Ch. 21.

### N

- **nil** — Valeur zéro des pointeurs, slices, maps, channels, fonctions et
  interfaces. ⚠️ Une interface valant nil n'est pas forcément « nil » si son type
  dynamique est renseigné. 🔁 voir Ch. 10, Ch. 33.

### O

- **Ordonnanceur** (scheduler) — Composant du runtime qui multiplexe les
  goroutines sur les threads via le modèle GMP, de façon coopérative et préemptive.
  🔁 voir Ch. 28.

### P

- **package** — Unité de compilation et d'encapsulation ; la visibilité est
  fixée par la casse de l'initiale (Majuscule = exporté). 🔁 voir Ch. 12.
- **panic / recover** — `panic` déroule la pile en exécutant les `defer` ;
  `recover`, appelé dans un `defer`, l'intercepte. À réserver aux cas vraiment
  exceptionnels. 🔁 voir Ch. 17.
- **PGO** (Profile-Guided Optimization) — Optimisation guidée par un profil :
  `go build` lit `default.pgo` et inline/dévirtualise plus agressivement les
  chemins chauds. 🔁 voir Ch. 39.
- **Pile / Tas** (stack / heap) — La **pile** est propre à chaque goroutine
  (rapide, libérée au retour) ; le **tas** est partagé et géré par le GC.
  🔁 voir Ch. 26.
- **pprof** — Format et outil de profils (CPU, tas, goroutines, blocages,
  mutex). Exploité via `go tool pprof`. 🔁 voir Ch. 37.
- **Préemption** — Capacité du runtime à interrompre une goroutine (préemption
  asynchrone depuis 1.14) pour rendre la main, même sans point de blocage.
  🔁 voir Ch. 28.

### R

- **Reflection** (`reflect`) — Inspection et manipulation des valeurs et types à
  l'exécution, via `reflect.Type`, `reflect.Value` et leur `Kind`. Puissant mais
  lent et non vérifié à la compilation. 🔁 voir Ch. 34.
- **rune** — Alias de `int32` représentant un point de code Unicode. Itérer une
  chaîne avec `for range` la décode rune par rune. 🔁 voir Ch. 31.
- **RWMutex** (`sync.RWMutex`) — Verrou lecteurs/écrivain : plusieurs lecteurs
  **ou** un seul écrivain. Avantageux quand les lectures dominent. 🔁 voir Ch. 21.

### S

- **select** — Attend sur plusieurs opérations de canal ; choisit un cas prêt (au
  hasard si plusieurs le sont). Un `default` le rend non bloquant. 🔁 voir Ch. 20.
- **Slice header** — En-tête d'une tranche : (pointeur vers le tableau
  sous-jacent, **len**, **cap**). Copier une slice copie l'en-tête, pas les données.
  🔁 voir Ch. 30.
- **Stack growth** — Croissance/rétrécissement automatique de la pile d'une
  goroutine : le runtime la recopie dans une plus grande zone quand elle déborde.
  🔁 voir Ch. 26.
- **String header** — En-tête d'une chaîne : (pointeur, longueur). Immuable ; la
  convertir en `[]byte` recopie en général les octets. 🔁 voir Ch. 31.
- **Swiss Table** — Conception de table de hachage adoptée par les maps Go 1.24
  pour de meilleures performances et localité. 🔁 voir Ch. 32.

### T

- **Type alias** (`type A = B`) — Deuxième nom pour un type existant. Depuis
  🆕 1.24, un alias peut être **générique** (`type Set[T comparable] = map[T]struct{}`).
  🔁 voir Ch. 11.
- **Type set** — Ensemble des types satisfaisant une contrainte (ex. `~int | ~string`,
  où `~T` inclut les types sous-jacents). 🔁 voir Ch. 11.
- **Trace d'exécution** (`runtime/trace`) — Journal chronologique fin des
  évènements du runtime (ordonnancement, GC, blocages), lu par `go tool trace`.
  🔁 voir Ch. 38.

### U

- **unsafe.Pointer** — Pointeur sans type permettant des conversions de bas
  niveau, hors des garanties habituelles. À manier avec une extrême prudence.
  🔁 voir Ch. 35.

### V

- **Valeur zéro** — Valeur par défaut d'un type non initialisé (`0`, `""`,
  `false`, `nil`, struct à champs zéro). Principe « rendre le zéro utile ».
  🔁 voir Ch. 3, Ch. 8.

### W

- **Weak pointer** (`weak.Pointer[T]`, 🆕 1.24) — Référence n'empêchant pas le GC
  de récupérer l'objet pointé ; utile pour des caches. 🔁 voir Ch. 27.
- **Work-stealing** — Stratégie d'équilibrage : un P sans travail « vole » des
  goroutines dans la file d'un autre P, ou puise dans la file globale.
  🔁 voir Ch. 28.
- **Write barrier** (barrière d'écriture) — Code inséré par le compilateur pendant
  le marquage du GC pour suivre les pointeurs modifiés et préserver la correction
  du marquage concurrent. 🔁 voir Ch. 27.

---

## 📌 À retenir

- Ce glossaire est un **index de repères** : chaque terme est développé dans le
  chapitre signalé par 🔁.
- Les notions reviennent par familles : **runtime** (GMP, GC, spans), **mémoire**
  (pile/tas, escape, modèle mémoire), **langage** (génériques, interfaces,
  itérateurs).
- En cas de doute sur un comportement subtil (nil d'interface, course de données,
  visibilité mémoire), retournez au chapitre dédié plutôt qu'à la définition seule.
