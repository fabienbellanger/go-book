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

⚠️ **N'utilisez pas `==` sur des `time.Time`.** Deux instants peuvent représenter le
même moment tout en différant par leur **fuseau** ou leur **lecture monotone** : `==`
les jugera inégaux. `Equal` compare l'instant réel.

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

---

## Fuseaux horaires : `time.Location`

```go
time.UTC                        // référence universelle
time.Local                      // fuseau de la machine
loc, err := time.LoadLocation("Europe/Paris")   // base IANA (tzdata)
t.In(loc)                       // même instant, autre fuseau d'affichage
```

> 💡 **Règle d'or** : **stocker et calculer en UTC**, ne convertir en local **qu'à
> l'affichage**. On évite ainsi les bugs de changement d'heure et de DST.

⚠️ `LoadLocation` lit la base tzdata du système. Sur une image conteneur minimale
(`scratch`), elle est absente → importez le package `time/tzdata` (embarque la base
dans le binaire) ou installez les fichiers. 🔁 Ch. 46.

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
- **Tickers** : toujours `Stop`. Depuis **1.23**, `After`/`Tick` ne fuient plus, mais
  `NewTimer`+`Reset` reste préférable pour réutiliser/contrôler un timer.
- Pour propager un délai dans une API, passer un **`context` avec deadline**, pas une durée.

## 🔁 Pour aller plus loin

- Formatage & parsing des dates : **Ch. 7**.
- `context` (annulation, deadlines, valeurs) : **Ch. 22**.
- Tests concurrents déterministes (`testing/synctest`, race detector) : **Ch. 23**.
- Embarquer la base tzdata dans un binaire (`time/tzdata`) : **Ch. 46**.
- Doc : `pkg.go.dev/time`, `pkg.go.dev/testing/synctest`.
