# 43 — Journalisation structurée

> **Objectif** — Produire des logs **structurés** (clés/valeurs), exploitables par une machine, avec
> `log/slog` : niveaux, handlers texte/JSON, attributs typés, contexte commun (`With`/`WithGroup`),
> rédaction paresseuse via `LogValuer`, intégration au `context`, et niveau ajustable à chaud.
>
> **Prérequis** — [Ch. 9 — Interfaces](09-interfaces.md), [Ch. 22 — `context`](22-context.md)

---

## Introduction

`fmt.Println("user", id, "a échoué")` produit une ligne **pour un humain** : pour la retrouver en
production, il faut une expression régulière qui casse au moindre changement de formulation du
message. Le **logging structuré** journalise des **paires clé/valeur** plutôt qu'une phrase : chaque
clé (`user_id`, `level`, `addr`…) devient un **champ indexable** côté outil d'exploitation (Loki,
Elasticsearch, Datadog…), interrogeable **indépendamment du texte** — une requête `user_id:7 AND
level:ERROR` reste valide même si le message passe de `"a échoué"` à `"connexion refusée"`. C'est ce
qui distingue le **diagnostic ponctuel** (lire des logs) de l'**exploitation outillée** (chercher,
agréger, alerter sur un champ précis, à l'échelle d'une flotte de services).

Depuis Go 1.21, le paquet **`log/slog`** est la réponse standard : une API stable, des **niveaux**
(`Debug`/`Info`/`Warn`/`Error`), des sorties **texte** (lisible en dev) ou **JSON** (ingérable par
Loki, Elasticsearch, Datadog…), le tout sans dépendance tierce. L'exemple complet est dans
[`code/ch43-slog/`](../code/ch43-slog/).

```
  log / fmt.Println                 log/slog
  ----------------                  --------
  "user 7 failed at :8080"   --->   {"level":"ERROR","user_id":7,"addr":":8080","msg":"failed"}
  texte plat, à parser              clés/valeurs, filtrable & agrégeable
```

---

## Premiers logs

Le logger **par défaut** (avant toute configuration) écrit sur `stderr`, mais **pas** au format d'un
`TextHandler` : tant que `slog.SetDefault` n'a pas été appelé, les fonctions de paquet passent par le
logger par défaut du paquet historique `log` (🔁 voir « Pont avec l'ancien `log` » plus bas pour ce
mécanisme). Ce sont malgré tout les plus directes pour démarrer :

```go
slog.Info("service démarré", "addr", ":8080", "pid", 4242)
slog.Warn("file presque pleine", "ratio", 0.92)
slog.Error("échec base", "err", err)
// 2009/11/10 23:00:00 INFO service démarré addr=:8080 pid=4242
```

Après le message viennent des **paires** clé (string) / valeur (any).

| Niveau            | Valeur | Usage typique                                                                |
| ----------------- | ------ | ---------------------------------------------------------------------------- |
| `slog.LevelDebug` | -4     | détail d'investigation (requête SQL, état intermédiaire) — bruyant en prod   |
| `slog.LevelInfo`  | 0      | événement normal du cycle de vie (démarrage, requête traitée) — défaut       |
| `slog.LevelWarn`  | 4      | anomalie absorbée automatiquement (retry, repli) — à surveiller sans alerter |
| `slog.LevelError` | 8      | échec qui requiert une action — déclenche typiquement une alerte             |

Les valeurs numériques ne sont pas arbitraires : l'écart de 4 entre niveaux laisse de la place pour des
niveaux intermédiaires propres à un écosystème (ex. `NOTICE` chez certains fournisseurs cloud), sans
toucher aux constantes `slog`.

Pour produire une sortie homogène (texte structuré ou JSON) plutôt que ce format hérité, on installe un
handler avec `slog.SetDefault` :

```go
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
```

## Handlers : texte ou JSON

Un **`Handler`** décide du **format** et de la **destination**. C'est la pièce qu'on **substitue** pour
changer la sortie sans toucher au code métier qui appelle `logger.Info(...)` :

```
  logger.Info(msg, attrs...)
         |
         v
   slog.Logger   ----Handle(ctx, Record)---->   slog.Handler   ---->   sortie
  (Info/Warn/Error,                            (Enabled, Handle,      (stdout, fichier,
   With, WithGroup)                             WithAttrs, WithGroup)  réseau...)
```

`Logger.With(...)` ne change **pas** de `Handler` : il renvoie un nouveau `Logger` qui pointe vers le
**même** handler, avec des attributs déjà attachés (détail au prochain encart) — c'est le `Handler`,
pas le `Logger`, qui décide concrètement du format texte/JSON. La librairie en fournit deux, plus un
handler nul :

| Handler                    | Sortie                  | Usage                         |
| -------------------------- | ----------------------- | ----------------------------- |
| `slog.NewTextHandler(w,o)` | `key=value` aligné      | développement, terminal       |
| `slog.NewJSONHandler(w,o)` | un objet JSON par ligne | production, ingestion machine |
| `slog.DiscardHandler`      | rien                    | tests / désactiver les logs   |

Le 3ᵉ argument est un `*slog.HandlerOptions` qui pilote le comportement :

```go
opts := &slog.HandlerOptions{
	Level:     slog.LevelDebug, // seuil minimal (Leveler) ; nil = Info
	AddSource: true,            // ajoute fichier:ligne de l'appel
	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{} // un Attr vide est retiré de la sortie
		}
		return a
	},
}
```

**`ReplaceAttr`** est appelé pour **chaque** attribut avant écriture : on s'en sert pour **renommer**
les clés built-in (`time`, `level`, `msg`, `source`), **convertir** un type, ou **caviarder** une
donnée personnelle. Renvoyer un `slog.Attr{}` (zéro) **supprime** l'attribut — c'est ainsi qu'on rend
les tests déterministes en retirant l'horodatage.

### Niveau ajustable à chaud : `LevelVar`

Si `Level` pointe vers un `*slog.LevelVar`, on change le seuil **pendant** l'exécution, sans
reconstruire le logger — idéal pour un endpoint d'administration qui passe en `Debug` à la demande :

```go
level := new(slog.LevelVar)                 // zéro = LevelInfo
h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
// ... plus tard, en réaction à un signal ou une requête :
level.Set(slog.LevelDebug)                  // sûr entre goroutines
```

## Attributs : typés plutôt que paires

La forme `"clé", valeur` est pratique mais **non typée** (et piégeuse, voir ⚠️). Les **constructeurs
d'attributs** sont typés et plus sûrs : en interne, `slog.Value` stocke un `int`, une `string`, une
`Duration`… directement dans ses champs plutôt que de les emballer dans une interface `any` — c'est
**cette représentation interne** qui évite le _boxing_, pas l'appel à `Info`/`Warn`/`Error` lui-même
(qui reste `args ...any`, voir ⚡ Performance pour la nuance) :

```go
logger.Info("connexion",
	slog.String("addr", ":8080"),
	slog.Int("retries", 3),
	slog.Duration("timeout", 5*time.Second),
	slog.Bool("tls", true),
	slog.Any("user", u),               // pour un type quelconque
	slog.Group("req",                  // sous-objet imbriqué
		slog.String("method", "GET"),
		slog.Int("status", 200),
	),
)
```

`slog.Group` produit un objet imbriqué : `"req":{"method":"GET","status":200}` en JSON.

## Contexte commun : `With` et `WithGroup`

`Logger.With` renvoie un logger **dérivé** dont les attributs sont **pré-calculés** une fois et
ajoutés à **chaque** message — parfait pour un identifiant de composant ou de connexion :

```go
reqLog := logger.With(slog.String("component", "auth"))
reqLog.Info("ok")      // ...component=auth msg=ok
reqLog.Warn("retry")   // ...component=auth msg=retry
```

`WithGroup("db")` préfixe **tous** les attributs suivants par `db.` (handler texte) ou les imbrique
dans un sous-objet (handler JSON) :

```go
dbLog := logger.WithGroup("db")
dbLog.Info("requête", slog.String("table", "users"), slog.Int("rows", 3))
// texte : ...db.table=users db.rows=3
// JSON  : ..."db":{"table":"users","rows":3}
```

### Schéma — dérivation d'un logger

```
  logger ---- With(component=auth) ----> reqLog ---- With(conn=12) ----> connLog
    |                                       |                               |
  attrs: -                             attrs: component=auth         attrs: component=auth, conn=12
                                       (pré-calculés, partagés)      (chaque message en hérite)
```

## `LogValuer` : rédaction & calcul paresseux

Un type peut décider **lui-même** de sa représentation dans les logs en implémentant
`slog.LogValuer` (parent de `Stringer`, [Ch. 9](09-interfaces.md)). Deux usages :

1. **Masquer un secret** — un mot de passe ne doit **jamais** apparaître en clair.
2. **Différer un calcul coûteux** — `LogValue()` n'est appelé **que si** l'enregistrement est émis.

```go
// code/ch43-slog/main.go
type Password string

func (Password) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

type User struct {
	ID   int
	Name string
	Pass Password
}

func (u User) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("id", u.ID),
		slog.String("name", u.Name),
		slog.Any("password", u.Pass),
	)
}
```

Le mot de passe passe par son propre `LogValuer` (celui de `Password`) : résultat,
`"user":{"id":7,"name":"ada","password":"[REDACTED]"}` — le secret ne fuit jamais, même imbriqué.

## Intégration au `context`

Les variantes **`*Context`** (`InfoContext`, `ErrorContext`, …) transportent le `context.Context`
jusqu'au handler. Un **handler personnalisé** peut alors en extraire des champs de **portée requête**
(identifiant de trace, utilisateur), sans les répéter à chaque appel :

```go
// code/ch43-slog/main.go — handler qui injecte le request_id du contexte
type contextHandler struct{ slog.Handler }

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}
```

> ⚠️ **Le piège de l'embedding.** En englobant `slog.Handler`, les méthodes `WithAttrs` et `WithGroup`
> sont **promues** et renvoient le handler **interne nu** : après un `logger.With(...)`, votre `Handle`
> (donc le `request_id`) serait **perdu**. Il faut les **réimplémenter** pour ré-emballer :
>
> ```go
> func (h contextHandler) WithAttrs(a []slog.Attr) slog.Handler {
> 	return contextHandler{h.Handler.WithAttrs(a)}
> }
> func (h contextHandler) WithGroup(n string) slog.Handler {
> 	return contextHandler{h.Handler.WithGroup(n)}
> }
> ```

## 🆕 Go 1.2x

- **1.21** — introduction de **`log/slog`** (API GA) et de **`slog.LogValuer`**.
- **1.22** — **`slog.SetLogLoggerLevel`** : règle le niveau minimal du pont implicite vers le paquet
  `log` (voir « Pont avec l'ancien `log` »).
- **1.24** — **`slog.DiscardHandler`** : une valeur prête à l'emploi pour jeter tous les logs (tests,
  désactivation), plus simple qu'un handler vers `io.Discard`.
- **1.25** — **`slog.GroupAttrs(key, attrs...)`** : variante de `slog.Group` qui accepte directement
  des `Attr` plutôt que `...any`, même logique d'optimisation que `Logger.LogAttrs`.
- **1.26** — **`slog.NewMultiHandler(h1, h2, …)`** : diffuse **chaque** enregistrement vers **plusieurs**
  handlers à la fois (par ex. un `TextHandler` lisible en console **et** un `JSONHandler` vers un
  fichier d'audit) :

  ```go
  logger := slog.New(slog.NewMultiHandler(
  	slog.NewTextHandler(os.Stderr, nil),
  	slog.NewJSONHandler(auditFile, nil),
  ))
  ```

## Pont avec l'ancien `log`

Le pont entre `log` et `slog` fonctionne dans **les deux sens**.

**`slog` vers `log`** — du code (ou une dépendance) qui écrit via `log.Printf` peut être **redirigé**
dans le pipeline structuré avec `slog.NewLogLogger`, qui fabrique un `*log.Logger` reversant vers un
handler `slog` :

```go
h := slog.NewJSONHandler(os.Stdout, nil)
std := slog.NewLogLogger(h, slog.LevelWarn) // *log.Logger
http.Server{ErrorLog: std}                  // les erreurs du serveur passent par slog
```

**`log` vers `slog`** — dans l'autre sens, `slog.SetDefault` met **aussi** à jour le logger par défaut
du paquet `log` : du code existant qui appelle encore `log.Printf` se met, **sans modification**, à
traverser le handler installé :

```go
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
log.Printf("legacy line %d", 42)
// {"time":"...","level":"INFO","msg":"legacy line 42"}
```

Avant ce premier `SetDefault`, c'est l'**inverse** qui se produit (cf. « Premiers logs » plus haut) :
les appels `slog.Info`/`Warn`/`Error` passent par le logger par défaut de `log`.
`slog.SetLogLoggerLevel` (1.22) règle le **niveau minimal** de ce pont implicite — utile pour laisser
passer du `Debug` avant même d'avoir configuré un vrai handler.

## ⚠️ Pièges

- **Nombre impair d'arguments** : `logger.Info("msg", "oops")` laisse `"oops"` **sans valeur** → la
  sortie contient `!BADKEY`. `go vet` détecte la forme littérale ; **préférez les attributs typés**
  (`slog.String(...)`), immunisés contre l'erreur.
- **Logguer un secret** : mot de passe, jeton, numéro de carte. Utilisez `LogValuer` ou `ReplaceAttr`
  pour les **caviarder à la source**.
- **Coût des arguments même filtrés** : `slog.Info("x", "dump", expensive())` **évalue** `expensive()`
  même si le niveau Info est désactivé. Pour différer, passez un type à `LogValuer` (lazy) ou gardez
  l'appel sous `if logger.Enabled(ctx, slog.LevelDebug)`.
- **Embedding d'un `Handler`** : réimplémentez `WithAttrs`/`WithGroup` (voir ci-dessus).
- **`slog.SetDefault` n'est pas rétroactif** : il change ce que renvoient `slog.Default()` et les
  fonctions de paquet **à partir de cet appel**, mais un `*slog.Logger` déjà construit via
  `slog.New(...)` garde **son** handler d'origine, même après un `SetDefault` ultérieur. Il met aussi
  à jour le logger par défaut du paquet `log` (🔁 « Pont avec l'ancien `log` » ci-dessus), ce qui
  surprend si on l'ignore.

## ⚡ Performance

- `With` **pré-calcule** et **mémorise** les attributs communs : moins de travail par message qu'en les
  répétant à chaque appel.
- Les **attributs typés** évitent le _boxing_ de la **valeur** dans `slog.Value`
  (🔁 [Ch. 33](33-interfaces-profondeur.md)). Mais `Info`/`Warn`/`Error`/`Debug` restent `args ...any` :
  chaque `Attr` est quand même reconverti en interface pour construire le slice variadique. Sur un
  chemin chaud, `Logger.LogAttrs` (et `slog.GroupAttrs` pour les groupes, 🆕 1.25) accepte `...Attr`
  directement et évite cette conversion :

  ```go
  logger.LogAttrs(ctx, slog.LevelInfo, "requête traitée",
  	slog.String("method", "GET"),
  	slog.Int("status", 200),
  )
  ```

- `AddSource` capture la position d'appel via `runtime.Callers` à **chaque** enregistrement émis : un
  coût mesurable sur un chemin à fort volume, à réserver aux niveaux `Warn`/`Error` plutôt qu'à tous
  les `Info`.
- Le `JSONHandler` réutilise ses buffers ; un handler personnalisé doit rester **léger** (déléguer au
  handler englobé plutôt que reformater).
- `Enabled` / `LogValuer` permettent d'**éviter** tout coût quand un niveau est désactivé.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch43-slog
go test -race ./ch43-slog/...
```

À essayer :

1. Remplacez le `JSONHandler` par un `TextHandler`, puis par `slog.NewMultiHandler` des deux.
2. Ajoutez un champ secret à `User` et vérifiez qu'il n'apparaît **jamais** dans la sortie.
3. Mettez `Level: slog.LevelWarn` et confirmez que les `Info`/`Debug` disparaissent ; basculez le
   `LevelVar` à `Debug` à chaud.

---

## 📌 À retenir

- `log/slog` produit des logs **structurés** (clés/valeurs) avec **niveaux**, via un **handler**
  `Text` (dev) ou `JSON` (prod) — standard, sans dépendance.
- Préférez les **attributs typés** (`slog.String`, `Int`, `Duration`, `Group`) aux paires `"clé", v` :
  plus sûrs, plus rapides, à l'abri du piège `!BADKEY`.
- `Logger.With` factorise un **contexte commun** pré-calculé ; `WithGroup` imbrique.
- `LogValuer` **masque** les secrets et **diffère** les calculs coûteux.
- Les variantes **`*Context`** + un handler maison propagent des champs de **portée requête**
  (request_id) — pensez à réimplémenter `WithAttrs`/`WithGroup` si vous englobez un handler.
- `slog.SetDefault` bascule les fonctions de paquet **et** le pont avec l'ancien `log` ; il n'affecte
  jamais un `*slog.Logger` déjà construit via `slog.New`.
- 🆕 `NewMultiHandler` (1.26) diffuse vers plusieurs sorties ; `DiscardHandler` (1.24) les jette ;
  `GroupAttrs` (1.25) complète `LogAttrs` (présent depuis 1.21) pour éviter le passage par `any` sur
  un chemin chaud.

## 🔁 Pour aller plus loin

- [Ch. 22 — `context`](22-context.md) : la source des champs de portée requête.
- [Ch. 9 — Interfaces](09-interfaces.md) : `LogValuer` est cousin de `Stringer`.
- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : coût du _boxing_ dans `any`.
- Projet 2 (API REST) : `slog` câblé de bout en bout dans un service HTTP (middleware de log,
  `request_id`, `slog.NewMultiHandler`).
- Doc : [`pkg.go.dev/log/slog`](https://pkg.go.dev/log/slog).
