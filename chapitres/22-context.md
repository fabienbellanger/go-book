# 22 — `context`

> **Objectif** — Propager une **annulation** et des **délais** à travers tout un arbre d'appels avec
> `context.Context` : `WithCancel`/`WithTimeout`/`WithDeadline`/`WithCancelCause`, `ctx.Done()`/`Err()`/
> `Cause()`, valeurs de contexte, et les **conventions d'API** à respecter.
>
> **Prérequis** — [Ch. 20 — Channels](20-channels-select.md), [Ch. 21 — Synchronisation](21-synchronisation.md), [Ch. 10 — Erreurs](10-erreurs.md)

---

## Introduction

Une requête HTTP est abandonnée, un délai expire, l'utilisateur fait `Ctrl-C` : il faut alors
**arrêter** tout le travail lancé pour cette requête — les goroutines, les appels réseau, les requêtes
SQL — **sans en oublier**. Passer un canal d'arrêt ([Ch. 19](19-goroutines.md)) à la main à chaque
fonction serait ingérable.

`context.Context` **standardise** ce signal : un objet qu'on passe en **premier argument** de chaque
fonction d'une chaîne d'appels, et qui porte à la fois l'**annulation**, la **deadline** et quelques
**valeurs** de portée requête. L'exemple est dans [`code/ch22-context/`](../code/ch22-context/).

---

## L'interface `Context`

Elle tient en quatre méthodes :

```go
type Context interface {
	Done() <-chan struct{}        // canal fermé quand le contexte est annulé
	Err() error                   // nil, ou Canceled / DeadlineExceeded
	Deadline() (time.Time, bool)  // échéance éventuelle
	Value(key any) any            // valeur de portée requête
}
```

Le cœur est **`Done()`** : un canal qu'on **ferme** pour signaler l'annulation — exactement le patron
du [Ch. 19](19-goroutines.md), mais **unifié** et **propagé**.

> 💡 Un `Context` est **sûr pour un usage concurrent** : plusieurs goroutines peuvent appeler ses
> méthodes (`Done()`, `Err()`, `Value()`…) **simultanément**, sans synchronisation supplémentaire de
> votre part. C'est même tout l'intérêt : un même contexte se partage librement entre toutes les
> goroutines lancées pour traiter une même requête.

## Dériver un contexte

On part d'une **racine** (`context.Background()` dans `main`/les tests) et on **dérive** des contextes
enfants. Annuler un parent annule **tous** ses descendants — la propagation descend dans tout le
sous-arbre, jamais vers le parent :

```
      context.Background()                    racine : jamais annulée, Done() == nil
              |
              | ctx1, cancel1 := WithTimeout(parent, 2*time.Second)
              v
            ctx1 -------------------------------+
              |                                 |
              | WithCancel(ctx1)                | WithCancel(ctx1)
              v                                 v
            ctx2                              ctx3

  cancel1() (ou expiration des 2 s)  -->  ctx1.Done() se ferme
                                      -->  ctx2.Done() ET ctx3.Done() se ferment AUSSI
```

| Fonction                  | Annulé quand…                            |
| ------------------------- | ---------------------------------------- |
| `WithCancel(parent)`      | on appelle `cancel()`                    |
| `WithTimeout(parent, d)`  | `cancel()` **ou** après la durée `d`     |
| `WithDeadline(parent, t)` | `cancel()` **ou** à l'instant `t`        |
| `WithCancelCause(parent)` | `cancel(err)` — enregistre une **cause** |

```go
ctx, cancel := context.WithTimeout(parent, 2*time.Second)
defer cancel() // TOUJOURS : libère les ressources même si le travail a fini avant
```

> ⚠️ Le `cancel` renvoyé **doit** être appelé, sinon le contexte (et son minuteur interne) **fuit**
> jusqu'à expiration. Le `defer cancel()` juste après la création est l'idiome.

## Respecter l'annulation

Un contexte ne **force** rien : c'est à **votre** code de le **surveiller**. Le motif central place
`ctx.Done()` dans le **même `select`** que le travail utile, pour rendre la main **immédiatement** :

```go
// code/ch22-context/context.go
func sumUntilCancel(ctx context.Context, in <-chan int) (int, error) {
	sum := 0
	for {
		select {
		case <-ctx.Done():
			return sum, context.Cause(ctx) // pourquoi on s'arrête
		case v, ok := <-in:
			if !ok {
				return sum, nil // entrée fermée : fin normale
			}
			sum += v
		}
	}
}
```

Pour une **boucle de calcul** sans canal, on teste périodiquement `ctx.Err() != nil` (ou
`<-ctx.Done()` en `select` avec `default`) entre deux unités de travail.

## `WithCancelCause` & `Cause`

`ctx.Err()` ne dit que **Canceled** ou **DeadlineExceeded**. Souvent on veut savoir **pourquoi**.
`WithCancelCause` permet d'annuler avec une **erreur métier**, récupérable via `context.Cause` :

```go
// code/ch22-context/main.go
ctx, cancel := context.WithCancelCause(context.Background())
cancel(ErrTooSlow)         // annulation AVEC cause

ctx.Err()           // -> context.Canceled        (toujours générique)
context.Cause(ctx)  // -> ErrTooSlow               (l'erreur précise)
```

C'est le bon moyen de distinguer « annulé par le client » de « budget de temps épuisé » de « erreur
amont », tout en restant compatible avec `errors.Is` ([Ch. 10](10-erreurs.md)).

## Valeurs de contexte

`context.WithValue` attache une donnée de **portée requête** (identifiant de trace, utilisateur
authentifié). La clé **doit** être d'un **type non exporté** pour éviter toute collision :

```go
// code/ch22-context/values.go
type ctxKey int
const requestIDKey ctxKey = iota

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}
func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}
```

> ⚠️ Les valeurs de contexte sont pour ce qui **traverse les frontières** (middleware → handler), pas
> pour les **paramètres** normaux d'une fonction. Une fonction dont le comportement dépend d'une valeur
> de contexte cachée est difficile à lire et à tester. Passez les paramètres **explicitement**.

## `context.AfterFunc` (1.21)

Pour déclencher un **nettoyage** à l'annulation sans écrire de `select`, `AfterFunc` exécute une
fonction (dans sa propre goroutine) dès que le contexte est annulé :

```go
stop := context.AfterFunc(ctx, func() { conn.Close() })
defer stop() // stop() annule l'association si le nettoyage n'est plus utile
```

## Conventions d'API

- Le contexte est le **premier paramètre**, nommé `ctx` : `func F(ctx context.Context, ...)`.
- **Ne le stockez pas** dans une struct ; **passez-le** explicitement le long de la chaîne d'appels.
  (Exception cadrée : `http.Request` le transporte — projet 2.)
- **Racines** : `context.Background()` en haut (main, init, tests) ; `context.TODO()` quand on ne sait
  pas encore quel contexte passer (marqueur temporaire). Ni l'une ni l'autre n'a de deadline ni de
  cause d'annulation : `Done()` y renvoie `nil` (un canal **qui ne se ferme jamais**, donc ne se
  déclenche jamais dans un `select` — 🔁 [Ch. 20](20-channels-select.md)) et `Err()` y renvoie
  toujours `nil`.
- **Ne passez jamais `nil`** comme contexte.

---

## 🆕 Go 1.2x

- **1.21** — **`context.AfterFunc`**, **`WithoutCancel`** (dériver un contexte qui **survit** à
  l'annulation du parent, pour une tâche de fond), **`WithDeadlineCause`** / **`WithTimeoutCause`**.
- **1.20** — **`WithCancelCause`** + **`context.Cause`** : annuler avec une **raison**.
- L'interface `Context` elle-même est **stable depuis 1.7** ; ces ajouts l'enrichissent sans la casser.

## ⚠️ Pièges

- **Oublier `cancel()`** : fuite du contexte et de son minuteur. `defer cancel()` systématique.
- **Ignorer le contexte** : recevoir un `ctx` et ne jamais surveiller `Done()` annule tout l'intérêt.
- **`context.Value` comme sac fourre-tout** : n'y mettez pas de paramètres métier ni de dépendances.
  Clé de **type non exporté** obligatoire.
- **Stocker un `Context` dans une struct** : il devient invisible et son cycle de vie incontrôlable.
- **Confondre `Err()` et `Cause()`** : `Err()` est générique (Canceled/DeadlineExceeded), `Cause()`
  porte la raison précise.

## ⚡ Performance

- `ctx.Done()` renvoie un canal ; le surveiller dans un `select` est **bon marché**. Le seul coût
  notable est de **réveiller** les goroutines en attente à l'annulation.
- `WithValue` construit une **liste chaînée** : chaque appel enveloppe le parent dans un maillon
  `{clé, valeur, parent}`. `Value(k)` la **parcourt** maillon par maillon jusqu'à trouver `k` (coût
  linéaire en **profondeur**, pas en nombre de clés distinctes) :

  ```
  ctx1 := WithValue(ctx0, keyA, "a")   // ctx1 -> {keyA:"a"} -> ctx0
  ctx2 := WithValue(ctx1, keyB, "b")   // ctx2 -> {keyB:"b"} -> ctx1 -> {keyA:"a"} -> ctx0
  ctx3 := WithValue(ctx2, keyC, "c")   // ctx3 -> {keyC:"c"} -> ctx2 -> {keyB:"b"} -> ctx1 -> {keyA:"a"} -> ctx0

  ctx3.Value(keyA) :
    ctx3{keyC} --pas trouvé--> ctx2{keyB} --pas trouvé--> ctx1{keyA} --TROUVÉ--> "a"
  ```

  Gardez les chaînes **courtes** ; ne l'utilisez pas comme une `map`.

- `WithCancel`/`WithTimeout` allouent un peu d'état (et un minuteur pour les variantes temporisées),
  libéré par `cancel()` — d'où l'importance de l'appeler.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch22-context
go test -race ./ch22-context/...
```

À essayer :

1. Annulez un `WithCancelCause` avec une erreur métier et comparez `ctx.Err()` et `context.Cause(ctx)`.
2. Donnez à `sumUntilCancel` un `WithTimeout` très court sur un canal jamais alimenté : observez
   `DeadlineExceeded`.
3. Empilez dix `WithValue` et mesurez le coût d'un `ctx.Value` au fond de la chaîne.

---

## 📌 À retenir

- `context.Context` propage **annulation + deadline + valeurs** le long d'un arbre d'appels ; il est le
  **premier** paramètre (`ctx`).
- On **dérive** d'une racine (`Background`) avec `WithCancel`/`WithTimeout`/`WithDeadline`/
  `WithCancelCause` ; annuler un parent annule les enfants.
- **Toujours** `defer cancel()` ; **toujours** surveiller `ctx.Done()` dans le travail long.
- `Cause(ctx)` donne la **raison** précise (1.20), là où `Err()` reste générique.
- Les **valeurs de contexte** servent aux données de **portée requête** (clé de type non exporté), pas
  aux paramètres.

## 🔁 Pour aller plus loin

- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : `context` dans les pipelines et pools.
- [Ch. 20 — Channels](20-channels-select.md) : `Done()` n'est qu'un canal fermé.
- [Ch. 10 — Erreurs](10-erreurs.md) : `Cause`/`Err` et `errors.Is`.
- Projet 2 (API REST) : `context` de bout en bout (HTTP, base de données, timeouts).
