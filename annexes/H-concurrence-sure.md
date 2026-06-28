# Annexe H — Concurrence sûre : éviter data races & deadlocks

> **Objectif** — Une **référence transversale** pour écrire du code concurrent
> correct en Go et **traquer** les deux fautes les plus coûteuses : la **data
> race** (résultat indéfini, invisible en test) et le **deadlock** (blocage). Cette
> annexe condense les règles, le catalogue des pièges avec leur correctif, et le
> **mode opératoire de débogage**. Le code des patterns sûrs vit dans
> [`code/annexe-H-concurrence/`](../code/annexe-H-concurrence/) et passe
> `go test -race`.
>
> **Réinvestit** — [Ch. 19 Goroutines](../chapitres/19-goroutines.md),
> [Ch. 20 Channels & select](../chapitres/20-channels-select.md),
> [Ch. 21 Synchronisation](../chapitres/21-synchronisation.md),
> [Ch. 22 context](../chapitres/22-context.md),
> [Ch. 23 Patterns de concurrence](../chapitres/23-patterns-concurrence.md),
> [Ch. 25 Modèle mémoire](../chapitres/25-modele-memoire.md).

---

## 1. Go n'est pas Rust : la sûreté est une discipline, pas une garantie

Rust prouve l'absence de data races **à la compilation** (borrow checker : une seule
référence mutable à la fois). Go fait un autre choix : **la concurrence est facile à
écrire, mais sa correction n'est pas vérifiée par le compilateur**. Le langage vous
laisse partager une variable entre goroutines sans la protéger — et compilera.

```
  Rust   : data race  = ERREUR DE COMPILATION (le code ne build pas)
  Go     : data race  = comportement INDÉFINI à l'exécution (le code build et « marche »… parfois)
```

La sûreté en Go repose donc sur **trois piliers** :

1. **La discipline** : des règles de conception (cette annexe) appliquées systématiquement.
2. **Les outils** : le détecteur de courses `-race`, `go vet`, le détecteur de
   deadlock du runtime, les profils `goroutine`/`mutex`/`block`.
3. **La revue** : un humain qui vérifie « qui possède cette donnée ? qui la lit, qui
   l'écrit, sous quelle synchronisation ? ».

> ⚠️ Une data race n'est **pas** « lire une valeur périmée ». Le modèle mémoire
> ([Ch. 25](../chapitres/25-modele-memoire.md)) la déclare **indéfinie** : le
> compilateur et le CPU peuvent réordonner, déchirer une valeur (lecture partielle),
> ou produire un résultat impossible à reproduire. C'est pourquoi elle est si dure à
> déboguer : elle passe 10 000 fois, puis casse en production.

---

## 2. Les règles d'or

| #   | Règle                                                                | Pourquoi                                                                         |
| --- | -------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| 1   | **Partagez la mémoire en communiquant**, pas l'inverse               | Un canal cède la _propriété_ : une seule goroutine touche la donnée à la fois.   |
| 2   | **Une donnée = un propriétaire** à un instant donné                  | Pas de propriétaire = pas de course possible à raisonner.                        |
| 3   | Si vous **devez** partager, **protégez** (mutex / atomic)            | L'accès non synchronisé à une mémoire écrite est une course.                     |
| 4   | **Protégez la lecture aussi**, pas seulement l'écriture              | Lire pendant qu'on écrit est déjà une course.                                    |
| 5   | **`defer mu.Unlock()`** juste après `Lock()`                         | Libère sur tous les chemins (retour anticipé, panique).                          |
| 6   | **Un ordre global de verrouillage** quand plusieurs verrous          | Élimine le deadlock « AB-BA ».                                                   |
| 7   | **Le producteur ferme le canal**, jamais le consommateur             | Fermer ce qu'on n'envoie pas = panique ; double close = panique.                 |
| 8   | **Sachez comment chaque goroutine s'arrête** avant de la lancer      | Une goroutine sans sortie = une fuite ([Ch. 19](../chapitres/19-goroutines.md)). |
| 9   | **`go test -race` en CI**, sur des tests qui exercent la concurrence | L'outil ne voit que les courses _exécutées_.                                     |
| 10  | **Ne testez pas la correction par le timing** (`time.Sleep`)         | Lent et instable ; utilisez `synctest` ou des signaux explicites.                |

> 💡 **Le mantra de Go** : _« Don't communicate by sharing memory; share memory by
> communicating. »_ Quand un canal suffit, préférez-le à un mutex : il n'y a alors
> plus rien à protéger.

---

## 3. Data races : catalogue & correctifs

Chaque cas : ❌ la forme fautive, ✅ le correctif. Toutes les formes ✅ passent `-race`.

### 3.1 Compteur partagé non protégé

```go
// ❌ COURSE : n++ est lecture + écriture, deux goroutines sans synchronisation.
var n int
go func() { n++ }()
go func() { n++ }()
```

```go
// ✅ Mutex (ou sync/atomic pour un simple entier) — cf. Counter / AtomicCounter.
type Counter struct { mu sync.Mutex; n int }
func (c *Counter) Inc()   { c.mu.Lock(); defer c.mu.Unlock(); c.n++ }
func (c *Counter) Value() int { c.mu.Lock(); defer c.mu.Unlock(); return c.n }
```

### 3.2 Map en accès concurrent

```go
// ❌ COURSE (et parfois « fatal error: concurrent map writes », qui CRASHE le process).
m := map[string]int{}
go func() { m["a"]++ }()
go func() { m["b"]++ }()
```

```go
// ✅ Mutex autour de la map, ou sync.Map pour un cache très concurrent (🔁 Ch. 21).
var mu sync.Mutex
mu.Lock(); m["a"]++; mu.Unlock()
```

> ⚠️ L'écriture concurrente d'une map déclenche un **`fatal error`** non
> récupérable (pas un simple avertissement `-race`) : le runtime tue le programme.

### 3.3 Variable de boucle capturée (corrigé en 🆕 Go 1.22)

```go
// AVANT 1.22 : toutes les goroutines partageaient le MÊME i -> course + valeurs fausses.
for i := 0; i < 3; i++ {
	go func() { fmt.Println(i) }() // imprimait souvent 3 3 3
}
```

Depuis **Go 1.22**, `i` est **redéclaré à chaque itération** : le piège historique a
disparu ([Ch. 15](../chapitres/15-closures.md)). Sur du code ancien (ou avec un
`go` directive < 1.22), capturez explicitement : `i := i` en tête de boucle.

### 3.4 Slice/`append` partagé

```go
// ❌ COURSE : append peut réallouer ; deux goroutines qui appendent au même slice
// se marchent dessus (header partagé, réallocations concurrentes).
var s []int
go func() { s = append(s, 1) }()
go func() { s = append(s, 2) }()
```

```go
// ✅ Chaque goroutine écrit à un INDICE pré-réservé qui lui est propre (pas de partage d'écriture)…
results := make([]int, n)
for i := range n {
	go func() { results[i] = work(i) }() // indices disjoints : sûr
}
// …ou collectez via un canal (cession de propriété).
```

### 3.5 Initialisation paresseuse (check-then-act)

```go
// ❌ COURSE : deux goroutines voient conn == nil et initialisent en double.
var conn *Conn
if conn == nil { conn = dial() }
```

```go
// ✅ sync.Once garantit une initialisation unique et publiée correctement (🔁 Ch. 21).
var once sync.Once
once.Do(func() { conn = dial() })
// ou : var conn = sync.OnceValue(dial)  // 1.21
```

### 3.6 Publier un état entier : `atomic.Pointer[T]`

```go
// ✅ Remplacer atomiquement une configuration immuable, lue par milliers de goroutines.
var cfg atomic.Pointer[Config]
cfg.Store(loaded)        // écrivain
c := cfg.Load()          // lecteurs : sans verrou, toujours une vue cohérente
```

---

## 4. Deadlocks : les causes & leurs correctifs

Un **deadlock** (interblocage) : des goroutines s'attendent en cercle, plus rien
n'avance. Go détecte **un seul** cas automatiquement.

> 🆕 **Le détecteur de deadlock du runtime** affiche
> `fatal error: all goroutines are asleep - deadlock!` — mais **uniquement** si
> **TOUTES** les goroutines sont bloquées. Un deadlock **partiel** (le `main`
> tourne, deux goroutines se bloquent) **n'est pas** détecté : le programme se fige
> sans message. D'où l'importance des sections 5 et 6.

### 4.1 Verrous pris dans des ordres opposés (« AB-BA »)

C'est **le** deadlock classique des verrous multiples :

```go
// ❌ DEADLOCK possible : g1 prend A puis B ; g2 prend B puis A. Si chacune tient
// son premier verrou et attend le second, blocage définitif.
go func() { a.Lock(); b.Lock(); /* … */ }()
go func() { b.Lock(); a.Lock(); /* … */ }()
```

```go
// ✅ ORDRE GLOBAL : toujours verrouiller dans le même ordre (ici, par id croissant).
func Transfer(from, to *Account, amount int64) {
	if from == to { return }              // éviter le double-Lock du même mutex
	first, second := from, to
	if first.id > second.id { first, second = second, first }
	first.mu.Lock();  defer first.mu.Unlock()
	second.mu.Lock(); defer second.mu.Unlock()
	from.balance -= amount; to.balance += amount
}
```

> 📌 Règle générale : **définissez un ordre total sur vos verrous** (par adresse, id,
> ou niveau hiérarchique) et **respectez-le partout**. C'est le correctif universel
> du deadlock multi-verrous.

### 4.2 Auto-interblocage : verrou non réentrant

Les mutex Go **ne sont pas réentrants** : reverrouiller dans la même goroutine bloque.

```go
// ❌ DEADLOCK : f() prend le verrou, puis appelle g() qui le reprend.
func (s *S) f() { s.mu.Lock(); defer s.mu.Unlock(); s.g() }
func (s *S) g() { s.mu.Lock(); defer s.mu.Unlock(); /* … */ } // se bloque sur soi-même
```

```go
// ✅ Séparer la méthode publique (qui verrouille) de la logique privée (qui suppose le verrou tenu).
func (s *S) F() { s.mu.Lock(); defer s.mu.Unlock(); s.gLocked() }
func (s *S) gLocked() { /* suppose mu déjà tenu, ne reverrouille pas */ }
```

Même piège avec `RWMutex` : prendre `Lock()` alors qu'on tient déjà `RLock()` (ou
`RLock` deux fois alors qu'un écrivain attend) **bloque**.

### 4.3 Oublier `Unlock` (chemin d'erreur / panique)

```go
// ❌ Si validate() renvoie une erreur, le verrou n'est JAMAIS libéré -> deadlock du suivant.
mu.Lock()
if err := validate(); err != nil { return err } // fuite de verrou !
mu.Unlock()
```

```go
// ✅ defer libère sur TOUS les chemins, y compris panique.
mu.Lock()
defer mu.Unlock()
if err := validate(); err != nil { return err }
```

### 4.4 Deadlocks de canaux

```go
// ❌ Envoi sur un canal non bufferisé sans receveur -> blocage (et « all goroutines asleep »
// si c'est la seule goroutine).
ch := make(chan int)
ch <- 1            // personne ne reçoit : bloque pour toujours
```

```go
// ❌ Lecture d'un canal jamais fermé -> range bloque indéfiniment.
for v := range ch { use(v) } // si l'émetteur ne ferme jamais : fuite/blocage
```

```go
// ❌ Canal nil : un envoi OU une réception sur un canal nil bloque POUR TOUJOURS.
var ch chan int    // nil
<-ch               // bloque définitivement (utile dans un select pour « désactiver » un cas)
```

✅ Correctifs : **le producteur `close()`** quand il a fini (le `range` se termine) ;
bornez l'attente avec un **`select` + `ctx.Done()`** ou `time.After` ([Ch. 20](../chapitres/20-channels-select.md),
[Ch. 22](../chapitres/22-context.md)) ; ne lisez/écrivez jamais un canal `nil` par
mégarde.

### 4.5 `WaitGroup` mal orchestré

```go
// ❌ DEADLOCK : Add() appelé DANS la goroutine -> Wait() peut partir avant, ou le compteur
// ne retombe jamais à zéro. (go vet waitgroup, 1.25, attrape souvent ce cas.)
go func() { wg.Add(1); defer wg.Done(); work() }()
wg.Wait()
```

```go
// ✅ Add AVANT de lancer la goroutine — ou, mieux, WaitGroup.Go (1.25) qui fait Add+Done.
wg.Go(work)   // Go 1.25 : équivaut à wg.Add(1); go func(){ defer wg.Done(); work() }()
wg.Wait()
```

---

## 5. Le mode opératoire de détection

| Outil                               | Cible                              | Commande                                                                      |
| ----------------------------------- | ---------------------------------- | ----------------------------------------------------------------------------- |
| **Détecteur de courses**            | data races                         | `go test -race ./...` · `go build -race`                                      |
| **Détecteur de deadlock (runtime)** | blocage **total**                  | automatique : `fatal error: all goroutines are asleep`                        |
| **Vidage des goroutines**           | deadlock **partiel**               | `SIGQUIT` (Ctrl-\\) ou `GOTRACEBACK=all` → piles de **toutes** les goroutines |
| **Profil `goroutine`**              | goroutines bloquées                | `/debug/pprof/goroutine?debug=2` (pile complète) — 🔁 Ch. 29                  |
| **`goroutineleak`** (🆕 1.26)       | goroutines bloquées **à jamais**   | `GOEXPERIMENT=goroutineleakprofile` — 🔁 Ch. 23                               |
| **Profil `mutex`**                  | contention de verrous              | `go test -mutexprofile=mu.out` → `go tool pprof`                              |
| **Profil `block`**                  | attentes longues (canaux, verrous) | `go test -blockprofile=bl.out`                                                |
| **`go vet`**                        | bugs statiques                     | `waitgroup` (Add mal placé), `loopclosure`, `copylocks`                       |
| **`testing/synctest`** (🆕 1.25)    | logique temporelle                 | horloge virtuelle, déterministe — 🔁 Ch. 23                                   |

**Lire un rapport `-race`** : il donne **deux** piles — l'accès courant et l'accès
précédent en conflit — plus l'adresse mémoire. Cherchez la variable commune et
ajoutez la synchronisation manquante.

```
WARNING: DATA RACE
Write at 0x… by goroutine 8:   <- accès fautif n°1 (pile)
Previous write at 0x… by goroutine 7:   <- accès fautif n°2 (pile)
```

**Diagnostiquer un figement** (deadlock partiel, pas de message) : envoyez
`SIGQUIT` (ou `kill -QUIT <pid>`). Le runtime imprime la pile de **chaque**
goroutine ; celles arrêtées sur `sync.(*Mutex).Lock` ou `chan receive` depuis
longtemps désignent le cycle.

---

## 6. Liste de contrôle avant de fusionner (pre-merge)

- [ ] Chaque variable partagée est **protégée** (mutex/atomic) **ou** confinée à une
      seule goroutine (cession par canal).
- [ ] **Lectures comprises** : pas seulement les écritures.
- [ ] Chaque `Lock()` a son **`defer Unlock()`**.
- [ ] **Un ordre de verrouillage global** est défini et respecté (si ≥ 2 verrous).
- [ ] Aucune méthode ne **reverrouille** un mutex déjà tenu par la même goroutine.
- [ ] Chaque goroutine a une **condition d'arrêt claire** (canal fermé, `ctx.Done()`).
- [ ] Les canaux sont **fermés par le producteur**, une seule fois.
- [ ] `WaitGroup.Add` est **hors** des goroutines (ou `wg.Go`).
- [ ] Les tests **exercent** la concurrence et tournent sous **`-race` en CI**.
- [ ] Pas de `time.Sleep` pour synchroniser/tester : `synctest` ou signaux explicites.
- [ ] `go vet ./...` propre (`waitgroup`, `copylocks`, `loopclosure`).

---

## 📌 À retenir

- Go **ne prouve pas** l'absence de courses : la sûreté = **discipline + outils +
  revue**. Le compilateur vous laissera écrire le bug.
- **Communiquer plutôt que partager** ; sinon **protéger** (et la lecture aussi) ;
  **`defer Unlock`** toujours.
- Le deadlock multi-verrous se tue par un **ordre de verrouillage global** ; les
  mutex Go ne sont **pas réentrants**.
- Le runtime ne détecte que le deadlock **total** ; pour le reste, **vidage des
  goroutines** (`SIGQUIT`), profils `goroutine`/`mutex`/`block`, `goroutineleak`.
- **`go test -race` en CI** sur des tests qui exercent vraiment la concurrence : une
  course non exécutée ne sera pas vue.
