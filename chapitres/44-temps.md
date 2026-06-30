# 44 — Le temps en pratique

> **Objectif** — Manipuler des instants et des durées sans se tromper : horloge
> monotone, comparaisons, fuseaux, timers/tickers correctement arrêtés, et
> intégration avec `context`.

> **Prérequis** — Ch. 7 (formatage/parsing des dates), Ch. 20 (`select`), Ch. 22
> (`context`), Ch. 23 (tests concurrents).

---

## Introduction

Le Ch. 7 a couvert le **formatage** et le **parsing** (la fameuse date de référence
`2006-01-02 15:04:05`). Ce chapitre traite tout le **reste** du package `time` :
représenter un instant, mesurer une durée, planifier une action, gérer les fuseaux.

Trois types portent l'essentiel :

```
  time.Time      un INSTANT (date + heure + fuseau + lecture monotone)
  time.Duration  une DURÉE  (int64 de nanosecondes)
  time.Location  un FUSEAU horaire
```

> 🔁 Pour `Format`/`Parse` et la table exhaustive des tokens, voir **Ch. 7**.

---

## `time.Time` : un instant

```go
now := time.Now()                                  // instant courant (+ monotone)
t := time.Date(2025, time.June, 28, 14, 30, 0, 0, time.UTC)

t.Year(); t.Month(); t.Day(); t.Hour(); t.Weekday() // composantes
t.Unix(); t.UnixMilli(); t.UnixNano()               // secondes/ms/ns depuis 1970
time.Unix(1750000000, 0)                            // reconstruire depuis un epoch
```

> ⚡ `Year()`, `Month()` et `Day()` appelés séparément refont chacun, indépendamment, la
> conversion depuis le temps absolu vers le calendrier grégorien. Si vous avez besoin de
> plusieurs composantes, `year, month, day := t.Date()` les calcule en un seul passage —
> un appel au lieu de trois.

Arithmétique : on **ajoute une durée** ou on **décale par calendrier**.

```go
t.Add(90 * time.Minute)       // + 1h30 (durée absolue)
t.AddDate(0, 1, 0)            // + 1 mois (calendaire : gère les longueurs de mois)
later.Sub(t)                  // time.Duration entre deux instants
```

⚠️ `Add` raisonne en durée fixe ; `AddDate` raisonne en calendrier. `Add(24*time.Hour)`
n'est **pas** toujours « demain même heure » (changements d'heure été/hiver).

### Comparer : `Before`/`After`/`Equal`, jamais `==`

```go
a.Before(b); a.After(b); a.Equal(b)
```

⚠️ **N'utilisez pas `==` sur des `time.Time`.** `time.Time` est un **struct** (champs
internes non exportés portant le temps mural, un éventuel temps monotone et un pointeur
vers la `*Location`) ; `==` compare ces champs **un à un**, comme pour n'importe quel
struct comparable. Deux instants peuvent représenter le même moment tout en différant
par leur **fuseau** (pointeur `*Location` différent) ou leur **lecture monotone**
(présente sur l'un, absente sur l'autre) : `==` les jugera inégaux. `Equal` compare
l'instant réel, indépendamment de ces détails de représentation.

```go
paris, _ := time.LoadLocation("Europe/Paris")
u := time.Date(2025, 6, 28, 14, 0, 0, 0, time.UTC)
p := u.In(paris)            // 16:00 +0200 — même instant
u.Equal(p)                 // true
u == p                     // false (fuseaux différents) ⚠️
```

### Tronquer / arrondir

```go
t.Truncate(time.Hour)   // vers le bas (multiple de d depuis l'instant zéro)
t.Round(time.Minute)    // au plus proche (0.5 arrondi vers le haut)
```

⚠️ `Truncate`/`Round` raisonnent sur la durée **absolue** écoulée depuis l'instant zéro
(`0001-01-01 00:00:00 UTC`), **pas** sur l'heure affichée dans le fuseau de `t`. Avec un
fuseau dont le décalage UTC n'est pas un multiple entier de `d`, le résultat peut
surprendre : `t.Truncate(24*time.Hour)` ne tombe pas forcément sur minuit **local**
(seulement sur minuit **UTC**, qui peut être 1h ou 2h du matin à Paris selon la saison).
Pour tronquer au jour local, passez par les composantes (`time.Date(t.Year(), t.Month(),
t.Day(), 0, 0, 0, 0, t.Location())`) plutôt que par `Truncate`.

---

## ⚠️ L'horloge monotone

`time.Now()` lit **deux** horloges et les emballe dans le même `time.Time` :

```
  +--------------------- time.Now() ---------------------+
  |  horloge MURALE (wall)        horloge MONOTONE       |
  |  "28 juin 2025 14:30:00"      compteur depuis le boot |
  |  peut SAUTER (NTP, DST,        ne recule JAMAIS,      |
  |  réglage manuel)              ne saute pas           |
  +------------------------------------------------------+
            |                              |
       Format / affichage            Sub / Since (mesure de durée)
```

- **Mesurer un délai** (`b.Sub(a)`, `time.Since(a)`) utilise la composante **monotone** :
  le résultat reste correct même si l'horloge murale est réajustée entre `a` et `b`.
  C'est pourquoi on mesure **toujours** une durée avec `time.Since`, jamais en
  soustrayant deux `Unix()`.
- **Afficher / sérialiser** utilise la composante murale.

Pourquoi deux horloges plutôt qu'une seule ? Chacune répond à un besoin que l'autre ne
peut pas couvrir. La murale donne une **date civile** (utile pour afficher, journaliser,
comparer à une échéance fixée à l'avance), mais elle peut sauter en arrière ou en avant à
tout moment, hors du contrôle du programme. La monotone garantit une **progression
régulière** (indispensable pour mesurer un écart sans risque), mais elle n'a de sens
qu'à l'intérieur du processus courant — généralement un compteur depuis le démarrage de
la machine, sans rapport avec une date calendaire, donc impossible à afficher ou à
transmettre à un autre processus.

⚠️ **Comparaison entre deux `time.Time`** (`Before`/`After`/`Equal`/`Sub`/`Compare`) :
si **les deux** valeurs portent une lecture monotone, l'opération l'utilise exclusivement
et ignore le temps mural. Si **l'une des deux seulement** en a une (par exemple un
`time.Time` reconstruit avec `time.Date` comparé à un `time.Now()` brut), l'opération
**bascule silencieusement** sur le temps mural pour les deux valeurs — aucune erreur,
aucun avertissement, juste un résultat un peu moins fiable face aux sauts d'horloge.

⚠️ Certaines opérations **retirent** la lecture monotone : `t.Round(0)`, `t.Truncate(0)`,
le passage par un fuseau (`In`, `UTC`, `Local`), la sérialisation (`MarshalJSON`,
`Format` puis `Parse`). Après ça, l'instant est intact mais une mesure de durée
ultérieure retombe sur l'horloge murale.

```go
start := time.Now()
saved := start.Round(0)   // murale seule, plus de monotone
// time.Since(saved) serait sensible aux sauts d'horloge — à éviter pour mesurer.
```

> 💡 Règle simple : **mesurer** → garder le `time.Time` brut de `time.Now()` et utiliser
> `Since`. **Stocker / transmettre** → `Round(0)` ou sérialiser (le monotone n'a aucun
> sens hors du process courant de toute façon).

---

## `time.Duration` : une durée

`Duration` est un `int64` de **nanosecondes**, avec des constantes lisibles :

```go
const timeout = 2*time.Second + 500*time.Millisecond
timeout.String()             // "2.5s"
timeout.Seconds()            // 2.5 (float64)
d, err := time.ParseDuration("1h30m")  // parsing
```

⚠️ Multiplier une durée par une **variable** entière demande une conversion explicite —
`time.Duration` et `int` sont des types distincts :

```go
n := 5
time.Duration(n) * time.Second   // ✅ 5s
// n * time.Second               // ❌ ne compile pas (types incompatibles)
```

💡 `5 * time.Second` compile car `5` est une constante non typée (Ch. 3) ; `n * time.Second`
ne compile pas car `n` a le type `int`.

⚠️ Un `int64` de nanosecondes ne couvre qu'**environ 290 ans** (positifs ou négatifs). Au
delà — typiquement en multipliant une durée par une valeur calculée dynamiquement (jours
× heures × ...) sans borne de contrôle — la valeur **déborde silencieusement** et repart
côté négatif, sans panique ni erreur à la compilation ou à l'exécution. Validez les
durées calculées dynamiquement (configuration, entrée utilisateur) plutôt que de
multiplier en aveugle.

---

## Fuseaux horaires : `time.Location`

```go
time.UTC                        // référence universelle
time.Local                      // fuseau de la machine
loc, err := time.LoadLocation("Europe/Paris")   // base IANA (tzdata)
t.In(loc)                       // même instant, autre fuseau d'affichage
fixed := time.FixedZone("CET+1", 3600)          // décalage fixe, sans règle DST
```

> 💡 **Règle d'or** : **stocker et calculer en UTC**, ne convertir en local **qu'à
> l'affichage**. On évite ainsi les bugs de changement d'heure et de DST.

`time.Local` reflète le fuseau du **système** (variable d'environnement `TZ`, ou
configuration OS à défaut) : un même binaire affiche une heure différente selon la
machine qui l'exécute. Les conteneurs minimaux et la plupart des plateformes CI
démarrent sans `TZ` réglée, donc avec `time.Local == time.UTC` — une bonne raison
supplémentaire de ne **jamais** dépendre du fuseau local pour la logique métier, et de le
réserver à l'affichage final, explicitement converti avec `In`.

⚠️ `LoadLocation` lit la base tzdata du système. Sur une image conteneur minimale
(`scratch`), elle est absente → importez le package `time/tzdata` (embarque la base
dans le binaire) ou installez les fichiers. 🔁 Ch. 46.

⚡ `LoadLocation` **reparse** les fichiers tzdata à chaque appel, sans cache interne :
appelée dans un chemin chaud (par requête HTTP, par itération de boucle), elle coûte
inutilement cher. Chargez chaque `*Location` une fois (variable de niveau paquet ou
`init`) et réutilisez le pointeur obtenu.

---

## Timers & tickers

### `time.Sleep`, `time.After`

```go
time.Sleep(100 * time.Millisecond)         // bloque la goroutine courante

select {
case v := <-ch:
    use(v)
case <-time.After(2 * time.Second):        // timeout
    return errTimeout
}
```

### 🆕 Go 1.23 — l'ancien avertissement de fuite est obsolète

Avant Go 1.23, `time.After` et `time.Tick` étaient réputés **fuir** : le timer
sous-jacent n'était pas récupéré par le GC avant son déclenchement. **Depuis Go 1.23,
le GC récupère les timers/tickers non référencés même non stoppés.** `time.After`
dans une boucle `select` ne fuit donc plus.

⚠️ Il reste deux raisons de préférer `NewTimer` explicitement :

1. **Réutiliser** un seul timer sur plusieurs itérations (`Reset`) plutôt qu'en allouer
   un par tour — moins d'allocations.
2. **Contrôler** précisément l'arrêt (annuler un délai en cours).

Le changement de 1.23 vaut aussi pour `NewTimer`/`NewTicker` eux-mêmes : la documentation
du package précise désormais que `Stop` « n'est plus nécessaire pour aider le
ramasse-miettes ». Mais cela ne dispense **pas** d'appeler `Stop` dans le cas le plus
courant : tant qu'une référence vivante subsiste — un ticker stocké dans un champ de
struct, ou simplement une goroutine qui lit encore `ticker.C` — le GC ne peut **pas** le
récupérer, et le ticker continue de produire des tops jusqu'à un `Stop` explicite. Le fix
de 1.23 supprime la fuite **mémoire** sur un timer devenu inaccessible ; il ne supprime
pas le besoin de `Stop` pour arrêter un timer encore référencé.

### `NewTimer` / `Reset` / `Stop` et le drainage du canal

```go
timer := time.NewTimer(d)
defer timer.Stop()

if !timer.Stop() {       // Stop renvoie false si le timer a DÉJÀ expiré
    <-timer.C            // ... auquel cas une valeur attend dans le canal
}
timer.Reset(d)           // réutiliser proprement après drainage
```

⚠️ **Drainage** : avant un `Reset`, si `Stop()` renvoie `false`, le canal peut
contenir une valeur déjà émise ; la draîner évite de lire un vieux top par erreur.
(Voir `drainTimer` dans le code du chapitre.)

### `NewTicker`, `AfterFunc`, `Tick`

```go
ticker := time.NewTicker(time.Second)
defer ticker.Stop()             // toujours Stop : un ticker répète indéfiniment
for range ticker.C { /* ... */ }

time.AfterFunc(d, func() { ... })  // exécute func dans une goroutine après d

for range time.Tick(time.Second) { ... }  // pratique mais réservé au main/longue durée
```

⚠️ `time.Tick` ne donne **aucun moyen de l'arrêter** : ne l'utilisez que pour un
process qui tourne pour toute la vie du programme. Partout ailleurs, `NewTicker` + `Stop`.

```
  NewTicker(10ms) :   |----+----+----+----+----> (répète jusqu'à Stop)
                      10   20   30   40   50 ms

  NewTimer(50ms)  :   |-------------------+      (un seul tir)
                                          50 ms
```

| Aspect           | `time.Timer`                                                                       | `time.Ticker`                                                                                   |
| ---------------- | ---------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Émission         | **une seule fois**, après `d`                                                      | en **boucle**, toutes les `d`                                                                   |
| Constructeurs    | `time.NewTimer(d)`, `time.After(d)`                                                | `time.NewTicker(d)`, `time.Tick(d)`                                                             |
| Réutilisation    | `Reset(d)` reprogramme un nouveau tir unique                                       | sans objet — tourne en continu tant qu'actif                                                    |
| `Stop`           | annule un tir **en attente**                                                       | **seul** moyen de mettre fin aux tops                                                           |
| Si `Stop` oublié | depuis 1.23, pas de fuite mémoire si non référencé ; sinon le tir reste en attente | depuis 1.23, pas de fuite mémoire si non référencé ; sinon **continue de tourner indéfiniment** |
| Usage typique    | timeout, deadline ponctuelle                                                       | heartbeat, polling, rafraîchissement périodique                                                 |

⚠️ La durée passée à `NewTicker` doit être **strictement positive** : `NewTicker(0)` ou
une durée négative provoque une **panique** (`non-positive interval for NewTicker`).
`NewTimer`, à l'inverse, accepte une durée nulle ou négative — le timer se déclenche
alors **dès que possible**, sans paniquer ; c'est un moyen valide (bien que peu lisible)
d'envoyer immédiatement une valeur dans `timer.C`.

---

## Intégration avec `context`

Pour propager un délai à travers une chaîne d'appels, on ne passe pas une `Duration` :
on passe un `context` porteur d'une **deadline** (🔁 Ch. 22).

```go
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()                 // toujours libérer le timer du contexte

select {
case res := <-work(ctx):
    return res, nil
case <-ctx.Done():
    return zero, ctx.Err()     // context.DeadlineExceeded
}
```

C'est le motif `slowDouble` du code du chapitre : un `select` entre un `time.NewTimer`
et `ctx.Done()`.

---

## 🧪 À tester soi-même

Tester un **timeout** sans faire dormir la suite de tests pendant des secondes : le
package `testing/synctest` (🆕 GA en 1.25) exécute le test dans une **bulle** à
**horloge virtuelle**. Le temps n'avance que lorsque toutes les goroutines de la bulle
sont durablement bloquées — un `time.Sleep(200ms)` se résout **instantanément**.

```go
func TestSlowDoubleTimeout(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
        defer cancel()
        _, err := slowDouble(ctx, 21, 200*time.Millisecond) // deadline < délai
        if err != context.DeadlineExceeded {
            t.Errorf("err = %v, voulu DeadlineExceeded", err)
        }
    })
}
```

Le code complet (`measure`, `countTicks`, `drainTimer`, `slowDouble`) et ses tests
(dont le comptage de tops sous horloge virtuelle) vivent dans `code/ch44-time/`.

```bash
cd code && go test -race ./ch44-time/...
```

🔁 `testing/synctest` est détaillé au **Ch. 23** (tests concurrents déterministes).

---

## 📌 À retenir

- **Comparer** des instants avec `Equal`/`Before`/`After`, **jamais `==`** (fuseau +
  monotone faussent l'égalité).
- **Mesurer** une durée avec `time.Since` (horloge **monotone**, insensible aux sauts) ;
  `Round(0)`/sérialisation **retirent** le monotone.
- **`Duration`** est un `int64` de nanosecondes : `time.Duration(n) * time.Second` pour
  une variable (pas `n * time.Second`).
- **Fuseaux** : stocker en **UTC**, convertir en local seulement à l'affichage.
- **Tickers** : toujours `Stop` tant qu'une référence subsiste (struct, goroutine de
  lecture) — depuis **1.23**, le GC ne récupère que les timers/tickers devenus
  **inaccessibles**, quel que soit leur mode de création (`After`, `Tick`, `NewTimer`,
  `NewTicker`). `NewTimer`+`Reset` reste préférable pour réutiliser/contrôler un timer.
- Pour propager un délai dans une API, passer un **`context` avec deadline**, pas une durée.

## 🔁 Pour aller plus loin

- Formatage & parsing des dates : **Ch. 7**.
- `context` (annulation, deadlines, valeurs) : **Ch. 22**.
- Tests concurrents déterministes (`testing/synctest`, race detector) : **Ch. 23**.
- Embarquer la base tzdata dans un binaire (`time/tzdata`) : **Ch. 46**.
- Doc : `pkg.go.dev/time`, `pkg.go.dev/testing/synctest`.
