# 45 — `net/http`

> **Objectif** — Comprendre le **modèle** de `net/http` des deux côtés : écrire un **serveur**
> (`Handler`, `ServeMux` et son routage enrichi 1.22, middlewares, timeouts, arrêt propre) et un
> **client** robuste (`Client`, `Transport`, `RoundTripper`, annulation par contexte). Ce chapitre
> **explique** ce que le [Projet 2](../projets/2-api-rest/) **met en œuvre** dans une API complète.
>
> **Prérequis** — [Ch. 9 — Interfaces](09-interfaces.md), [Ch. 22 — `context`](22-context.md),
> [Ch. 41 — I/O & flux](41-io-flux.md)

---

## Introduction

HTTP est le cas d'usage **numéro un** de Go. Le package `net/http` fournit, dans la bibliothèque
standard et sans dépendance, **tout** ce qu'il faut pour un service de production : serveur, routeur,
client, pool de connexions, TLS. L'enjeu de ce chapitre n'est pas d'énumérer l'API mais d'en montrer
le **modèle mental** — deux interfaces symétriques, `Handler` côté serveur et `RoundTripper` côté
client — et les **réglages de production** qu'on oublie trop souvent (timeouts, fermeture du corps,
arrêt propre). Le code est dans [`code/ch45-http/`](../code/ch45-http/).

---

## Le cœur : l'interface `Handler`

Tout le serveur HTTP repose sur **une** interface à une méthode :

```go
type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
}
```

- `*http.Request` — la requête entrante (méthode, URL, en-têtes, `Body`, **contexte**).
- `http.ResponseWriter` — par où on **écrit** la réponse (en-têtes, code, corps).

Implémenter cette interface suffit. Mais le plus souvent on écrit une simple **fonction** et on
l'adapte avec `http.HandlerFunc`, un type qui fait d'une `func(w, r)` un `Handler` :

```go
func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Bonjour")
}
var h http.Handler = http.HandlerFunc(hello) // la fonction DEVIENT un Handler
```

> 💡 `HandlerFunc` est l'exemple canonique d'**adaptateur** : une méthode `ServeHTTP` qui se contente
> de **s'appeler elle-même**. C'est le patron à connaître pour comprendre tout l'écosystème HTTP de Go.

### Écrire la réponse : l'ordre compte

`ResponseWriter` impose une **séquence** stricte :

```
  1. w.Header().Set(...)   <- modifier les en-têtes   (AVANT tout le reste)
  2. w.WriteHeader(code)   <- figer le code de statut  (optionnel : 200 par défaut)
  3. w.Write(corps)        <- écrire le corps          (déclenche WriteHeader(200) si absent)
```

> ⚠️ Une fois le **premier `Write`** effectué, le code de statut et les en-têtes sont **partis sur le
> réseau**. Tout `WriteHeader` ou `Header().Set` **ultérieur** est silencieusement **ignoré**. C'est
> l'erreur classique : positionner un `Content-Type` ou un code d'erreur _après_ avoir déjà écrit.

---

## Le routeur : `ServeMux` & le routage enrichi (🆕 1.22)

`http.ServeMux` associe des **motifs** d'URL à des `Handler`. Depuis **Go 1.22**, le motif peut porter
une **méthode** et des **wildcards nommés** :

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")        // valeur du wildcard nommé
	fmt.Fprintf(w, "item %s", id)
})
mux.HandleFunc("POST /items", createItem)
```

| Élément de motif   | Signification                                                     |
| ------------------ | ----------------------------------------------------------------- |
| `GET /items/{id}`  | matche **GET** uniquement ; capture `{id}` → `r.PathValue("id")`  |
| `/items/{id}`      | matche **toutes** les méthodes                                    |
| `/files/{path...}` | wildcard **final** : capture le reste du chemin (segments inclus) |
| `/items/{$}`       | matche **exactement** `/items/` (pas les sous-chemins)            |
| `example.com/`     | restreint à un **hôte**                                           |

**Précédence** : le motif le **plus spécifique** gagne, indépendamment de l'ordre d'enregistrement
(`/items/{id}` l'emporte sur `/items/`). Si une route existe pour le chemin mais pas pour la méthode,
le mux répond **`405 Method Not Allowed`** automatiquement — plus besoin de le coder à la main. Si
deux motifs se **chevauchent sans que l'un soit strictement plus spécifique** (ex. `GET /` et
`/index.html`, qui matchent tous deux `GET /index.html`), ils sont **ambigus** : `Handle`/`HandleFunc`
**paniquent à l'enregistrement**, pas au routage d'une requête — l'erreur apparaît donc au démarrage,
jamais en production sur une URL précise.

Avant 1.22, router une méthode ou capturer un segment de chemin demandait du code à la main
(`strings.HasPrefix`, découpage de `r.URL.Path`) ou une dépendance tierce qui réimplémentait ce
travail. Le routage enrichi déplace cette logique **dans la stdlib** : pour une API qui reste une
liste **plate** de routes, `ServeMux` suffit désormais. La limite reste la **composition** :
`ServeMux` ne sait ni grouper des routes sous un middleware commun (« tout `/admin/...` passe par
`auth` ») ni déclarer de sous-routeur imbriqué — c'est précisément ce que chi ou gorilla/mux ajoutent
encore, et ce qui justifie d'y recourir quand les groupes de routes se multiplient.

> 🆕 **Avant 1.22**, `ServeMux` ne connaissait ni les méthodes ni les variables de chemin. Le
> [Projet 2](../projets/2-api-rest/) construit son routage sur la version enrichie.

---

## Les middlewares : `func(http.Handler) http.Handler`

Un **middleware** enveloppe un `Handler` pour ajouter un comportement transverse (journalisation,
récupération de panique, authentification, CORS). C'est la troisième pièce du triptyque côté
serveur, après `Handler` et `HandlerFunc` :

| Concept            | Nature                          | Signature                         | Rôle                                                                 |
| ------------------ | ------------------------------- | --------------------------------- | -------------------------------------------------------------------- |
| `http.Handler`     | interface à une méthode         | `ServeHTTP(w, r)`                 | le seul contrat que savent appeler `ServeMux` et `http.Server`       |
| `http.HandlerFunc` | adaptateur fonction → `Handler` | `func(w, r)`                      | écrire un handler comme une fonction simple, sans déclarer de struct |
| middleware         | fonction d'ordre supérieur      | `func(http.Handler) http.Handler` | enveloppe un `Handler` pour ajouter un comportement transverse       |

Sa signature idiomatique :

```go
// code/ch45-http/main.go
func logging(logf func(string, ...any)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)          // on appelle le handler enveloppé
			logf("%s %s en %s", r.Method, r.URL.Path, time.Since(start))
		})
	}
}
```

On les **compose** par emboîtement — la requête traverse les couches dans l'ordre, la réponse en
sens inverse :

```
  requête
     |
     v
  +------------------ logging --------------------+
  |  +------------- recover ------------------+   |
  |  |  +--------- auth ------------------+   |   |
  |  |  |          mux -> handler         |   |   |
  |  |  +--------------------------------+    |   |
  |  +---------------------------------------+    |
  +-----------------------------------------------+
     |
     v
  réponse
```

```go
handler := logging(log.Printf)(recover(auth(mux))) // mux est le cœur, logging la peau
```

`net/http` ne laisse de toute façon jamais une panique de handler **arrêter tout le process** :
chaque connexion est servie sous un `recover` **interne** au serveur, qui journalise la pile puis
**ferme la connexion**. Mais sans middleware `recover` applicatif, le client ne reçoit **aucune
réponse propre** — juste une connexion coupée, ou un corps tronqué si l'écriture avait déjà commencé.
C'est tout l'intérêt d'un `recoverMiddleware` : transformer la panique en un `500` net, **avant** que
`net/http` ne referme quoi que ce soit.

> 💡 Mettez `recover` **à l'extérieur** des middlewares susceptibles de paniquer, et la
> journalisation encore plus à l'extérieur pour mesurer le temps **total**. (Le pattern « frontière de
> recover » est détaillé au [Ch. 17](17-panic-recover.md).)

---

## Le serveur de production : `http.Server` & timeouts

`http.ListenAndServe(addr, handler)` est pratique pour un exemple, mais **dangereux en production** :
il utilise un `http.Server` **sans aucun timeout**, donc une connexion lente peut monopoliser une
ressource **indéfiniment** (attaque _Slowloris_). En production, on **construit** le serveur :

```go
// code/ch45-http/main.go
srv := &http.Server{
	Addr:              ":8080",
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,  // temps max pour lire les en-têtes
	ReadTimeout:       10 * time.Second, // lecture complète de la requête
	WriteTimeout:      10 * time.Second, // écriture complète de la réponse
	IdleTimeout:       60 * time.Second, // keep-alive entre deux requêtes
}
```

| Champ               | Borne                                         | Pourquoi                     |
| ------------------- | --------------------------------------------- | ---------------------------- |
| `ReadHeaderTimeout` | lecture des en-têtes                          | **anti-Slowloris** minimal   |
| `ReadTimeout`       | lecture en-têtes **+** corps                  | corps lent ou jamais terminé |
| `WriteTimeout`      | de la fin des en-têtes à la fin de la réponse | client qui ne lit pas        |
| `IdleTimeout`       | connexion keep-alive inactive                 | recyclage des connexions     |

> ⚠️ Les valeurs **par défaut sont nulles = aucune limite**. C'est l'un des réglages les plus souvent
> oubliés. Au **minimum**, fixez `ReadHeaderTimeout`.

### Arrêt propre : `Shutdown`

`srv.Shutdown(ctx)` ferme les listeners, **laisse finir** les requêtes en cours, puis rend la main —
ou abandonne si le `ctx` expire. C'est l'arrêt gracieux à brancher sur un signal (`SIGINT`/`SIGTERM`) :

```go
go func() {
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}()
<-stop // ex. signal.NotifyContext (Ch. 22) déclenché par SIGINT
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(ctx) // draine les requêtes en cours, borne par le contexte
```

> `ListenAndServe` renvoie **`http.ErrServerClosed`** lors d'un `Shutdown` : ce n'est **pas** une
> erreur à traiter comme un échec. 🔁 Le contexte de timeout : [Ch. 22](22-context.md).

---

## Le client : `http.Client`, pas `http.Get`

`http.Get`/`http.Post` sont des raccourcis sur `http.DefaultClient`, lequel n'a **aucun timeout**. Pour
tout ce qui n'est pas un script jetable, **créez et réutilisez** un `http.Client` :

```go
client := &http.Client{Timeout: 5 * time.Second} // timeout GLOBAL par requête
```

« Global » signifie que ce délai borne **tout** : connexion, redirections suivies, et **lecture du
corps de la réponse**. Le chronomètre continue de tourner après le retour de `Get`/`Do` et peut
interrompre un `io.ReadAll(resp.Body)` en cours — un téléchargement volumineux peut donc échouer en
plein milieu si `Timeout` est trop court pour sa taille, alors même que les en-têtes sont arrivés
sans problème.

> ⚠️ Le `http.Client` est **conçu pour être réutilisé** (et il est sûr en concurrence). En recréer un
> par requête gaspille le **pool de connexions** du `Transport` sous-jacent.

### Toujours fermer — et drainer — le corps

```go
// code/ch45-http/main.go
resp, err := client.Get(url)
if err != nil {
	return "", err
}
defer resp.Body.Close()       // OBLIGATOIRE, sinon fuite de connexion/goroutine
data, err := io.ReadAll(resp.Body) // drainer jusqu'au bout
```

> ⚠️ Oublier `resp.Body.Close()` **fuit** la connexion : elle n'est jamais rendue au pool et le
> _netpoller_ garde une goroutine d'attente. Et un corps **partiellement lu** empêche la **réutilisation**
> de la connexion (keep-alive) — lisez-le entièrement (`io.ReadAll`, ou `io.Copy(io.Discard, body)`)
> avant de le fermer.

### Annulation par contexte

Pour qu'une requête sortante respecte un délai ou l'abandon du client, attachez-lui un **contexte** :

```go
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err := client.Do(req) // annulée dès que ctx est Done()
```

C'est la **brique** qui propage l'annulation d'une requête entrante vers les appels sortants qu'elle
déclenche (🔁 [Ch. 22](22-context.md)) — sans fuite de goroutine bloquée sur une réponse qui ne
viendra jamais.

---

## `Transport` & `RoundTripper` : le pendant client du `Handler`

Le `Client` gère la **politique** (redirections, timeouts, cookies) ; le **transport** gère la
**mécanique** (connexions TCP, TLS, pool keep-alive). Le transport implémente une interface à **une**
méthode, symétrique de `Handler` :

```go
type RoundTripper interface {
	RoundTrip(*Request) (*Response, error)
}
```

On peut donc écrire des **middlewares côté client** — exactement comme côté serveur — en enveloppant un
`RoundTripper` : ajout d'un en-tête d'authentification, traçage, retry, métriques.

```go
// code/ch45-http/main.go
type headerRoundTripper struct {
	key, value string
	next       http.RoundTripper // nil -> http.DefaultTransport
}

func (t headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context()) // ne JAMAIS muter la requête reçue
	clone.Header.Set(t.key, t.value)
	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(clone)
}

client := &http.Client{Transport: headerRoundTripper{key: "X-Trace", value: "abc-123"}}
```

> ⚠️ Contrat du `RoundTripper` : il **ne doit pas** modifier la requête reçue (la **cloner**), ni
> interpréter les redirections (c'est le rôle du `Client`). Pour ajuster le pool, configurez un
> `*http.Transport` : `MaxIdleConns`, `MaxIdleConnsPerHost`, `IdleConnTimeout`.

Symétrie à retenir :

```
  SERVEUR                           CLIENT
  http.Handler.ServeHTTP(w, r)      http.RoundTripper.RoundTrip(r) -> resp
        ^  on enveloppe                     ^  on enveloppe
  middleware: Handler -> Handler    middleware: RoundTripper -> RoundTripper
```

---

## Servir des fichiers (& des fichiers embarqués)

`http.FileServer` sert une arborescence ; `http.ServeFileFS` sert **un** fichier depuis n'importe quel
`fs.FS` — y compris un système de fichiers **embarqué** dans le binaire avec `embed.FS` :

```go
//go:embed static
var assets embed.FS
mux.Handle("GET /static/", http.FileServerFS(assets))
```

> 🔁 L'embarquement de fichiers (`embed`) et le déploiement d'un binaire **autonome** sont traités au
> [Ch. 46](46-embed-build-deploiement.md).

---

## 🆕 Go 1.2x

- **1.22** — **routage enrichi** du `ServeMux` : méthodes, wildcards `{id}`/`{path...}`, `r.PathValue`,
  `405` automatique, précédence par spécificité.
- **1.25** — **`http.CrossOriginProtection`** : protection **anti-CSRF** intégrée, fondée sur l'en-tête
  `Sec-Fetch-Site` (et la comparaison `Origin`/`Host`). Les méthodes sûres (GET/HEAD/OPTIONS) sont
  toujours autorisées ; on l'active comme un middleware :

  ```go
  csrf := http.NewCrossOriginProtection()
  _ = csrf.AddTrustedOrigin("https://app.example.com")
  handler := csrf.Handler(mux) // rejette les requêtes cross-origin non sûres
  ```

- Le `net/http/pprof` expose les profils via le mux par défaut — utilisé au **Projet 7** (🔁
  [Ch. 37](37-profiling-pprof.md)).

> Vérifié sur Go 1.26.4 (`go doc net/http.CrossOriginProtection`).

## ⚠️ Pièges

- **Serveur sans timeout** (`http.ListenAndServe` brut) : exposition au _Slowloris_. Construisez un
  `http.Server` avec au moins `ReadHeaderTimeout`.
- **Oublier `resp.Body.Close()`** côté client : fuite de connexion **et** de goroutine.
- **Corps non drainé** : empêche la réutilisation keep-alive de la connexion.
- **Écrire après `WriteHeader`** (ou modifier les en-têtes après le premier `Write`) : ignoré
  silencieusement.
- **Handler bloquant sans contexte** : si le client abandonne, la goroutine du handler **fuit**.
  Surveillez `r.Context().Done()` dans les attentes longues.
- **Recréer un `http.Client` par appel** : on perd le pool de connexions.
- **`http.Get`/`DefaultClient` en production** : pas de timeout, blocage potentiel infini.

## ⚡ Performance

- **Réutiliser** le `Client`/`Transport` : la mise en cache des connexions (keep-alive) et des sessions
  TLS évite des poignées de main coûteuses. Réglez `MaxIdleConnsPerHost` si vous frappez un même hôte.
- **Drainer puis fermer** le corps conditionne la réutilisation de la connexion.
- Chaque requête entrante tourne dans **sa propre goroutine** : le serveur est concurrent par défaut.
  Bornez le travail par **contexte** et les ressources partagées par les primitives du
  [Ch. 21](21-synchronisation.md).
- `httptest` permet de **benchmarker** un handler sans réseau (🔁 [Ch. 36](36-tests-benchmarks-fuzzing.md)).

## 🧪 À tester soi-même

```bash
cd code
go test -race ./ch45-http/...
go run ./ch45-http      # serveur sur :8080 ; curl http://localhost:8080/items/42
```

À essayer :

1. Ajoutez une route `GET /files/{path...}` et affichez le chemin capturé.
2. Écrivez un middleware `recover` qui transforme une panique en `500`, et placez-le autour du mux.
3. Donnez au `headerRoundTripper` un compteur de retries sur erreur réseau, et testez-le contre un
   `httptest.Server` qui échoue les deux premières fois.

---

## 📌 À retenir

- Tout le serveur tient dans **`Handler.ServeHTTP(w, r)`** ; `HandlerFunc` adapte une simple fonction.
- Le **`ServeMux` enrichi (1.22)** gère méthodes, wildcards `{id}`/`{path...}`, `PathValue` et le `405`
  automatique — souvent suffisant sans routeur tiers.
- Un **middleware** est `func(http.Handler) http.Handler` ; on les **compose** par emboîtement.
- En production : **construisez** un `http.Server` avec des **timeouts**, et arrêtez-le par
  **`Shutdown(ctx)`**.
- Côté client : **réutilisez** un `http.Client` à `Timeout`, **`defer Body.Close()` + drainer**,
  annulez par **contexte**, et étendez via **`RoundTripper`** (le pendant client du `Handler`).

## 🔁 Pour aller plus loin

- [Projet 2 — API REST](../projets/2-api-rest/) : routage, middlewares, `slog`, base de données,
  validation et arrêt propre **mis en œuvre** de bout en bout.
- [Ch. 22 — `context`](22-context.md) : annulation et deadlines de bout en bout.
- [Ch. 43 — `log/slog`](43-journalisation-slog.md) : journaliser les requêtes proprement.
- [Ch. 46 — Embarquer & déployer](46-embed-build-deploiement.md) : `embed.FS` servie, binaire autonome.
- [Ch. 37 — Profiling](37-profiling-pprof.md) & Projet 7 : `net/http/pprof` sur un service réel.
- Doc : [`pkg.go.dev/net/http`](https://pkg.go.dev/net/http), [`net/http/httptest`](https://pkg.go.dev/net/http/httptest).
