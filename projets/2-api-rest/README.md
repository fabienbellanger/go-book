# Projet 2 — API REST : `tasksd`

> **Objectif** — Construire une **API REST** idiomatique avec la seule
> bibliothèque standard : routage par méthode (Go 1.22), middlewares chaînés,
> **journalisation structurée `slog`**, JSON + validation, **protection CSRF**
> (Go 1.25), persistance derrière une **interface** (mémoire _ou_
> `database/sql`), et **arrêt propre** sur signal.
>
> **Réinvestit** — [Ch. 9 Interfaces](../../chapitres/09-interfaces.md),
> [Ch. 10 Erreurs](../../chapitres/10-erreurs.md),
> [Ch. 12 Packages](../../chapitres/12-packages-modules.md),
> [Ch. 21 Synchronisation](../../chapitres/21-synchronisation.md),
> [Ch. 22 Context](../../chapitres/22-context.md),
> [Ch. 23 Patrons de concurrence](../../chapitres/23-patterns-concurrence.md).

---

## 1. Cahier des charges

`tasksd` est une petite API de **tâches** (_todo_). Elle expose un CRUD JSON et
un point de santé, **sans aucune dépendance externe** :

| Méthode + route          | Rôle                                                 | Succès |
| ------------------------ | ---------------------------------------------------- | ------ |
| `GET /healthz`           | Sonde de vivacité.                                   | 200    |
| `GET /api/tasks`         | Liste (filtre `?done=`, pagination `?limit&offset`). | 200    |
| `POST /api/tasks`        | Crée une tâche `{title, done}`.                      | 201    |
| `GET /api/tasks/{id}`    | Lit une tâche.                                       | 200    |
| `PUT /api/tasks/{id}`    | Remplace une tâche.                                  | 200    |
| `DELETE /api/tasks/{id}` | Supprime une tâche.                                  | 204    |

Contraintes :

- **Codes HTTP fidèles** : `400` (requête malformée), `404` (absente),
  `405` (méthode non autorisée), `422` (validation métier), `500` (interne).
- **Validation** : `title` obligatoire, ≤ 200 caractères.
- **Robustesse** : corps borné (1 Mio), champs JSON inconnus refusés, délais de
  lecture/écriture, **arrêt propre** des requêtes en cours.
- **Observabilité** : un log structuré par requête, corrélé par `X-Request-Id`.
- **Sécurité navigateur** : protection **CSRF** sur les méthodes non sûres.

```bash
$ curl -s -X POST localhost:8080/api/tasks -d '{"title":"écrire le README"}'
{"id":1,"title":"écrire le README","done":false,"created_at":"2026-06-28T19:44:09Z"}

$ curl -s localhost:8080/api/tasks
{"tasks":[{"id":1,"title":"écrire le README","done":false,"created_at":"..."}],"count":1}
```

---

## 2. Architecture

```
                main.go
                  |  signal.NotifyContext(SIGINT, SIGTERM)
                  |  os.Exit(api.Run(ctx, args, stdout, stderr))
                  v
            +--------------+        Run = configuration + cycle de vie :
            |   api.Run    |        flags, logger, serveur, arrêt propre.
            +--------------+
                  | construit
                  v
   +-------------------------------------------+
   |               api.Server                   |  implémente http.Handler
   |                                            |
   |  recoverPanic → requestID → logging → CSRF |  chaîne de middlewares
   |                     ↓                       |
   |               http.ServeMux                |  routage 1.22 (méthode + {id})
   |                     ↓                       |
   |   handleList / handleCreate / handleGet …  |  handlers JSON
   +-------------------------------------------+
                  | dépend de l'interface
                  v
        +-----------------------+
        |     store.Store       |  Create / Get / List / Update / Delete
        +-----------------------+
            /                \
     MemStore (défaut)   SQLStore (database/sql + migrations)
```

Le **point pivot** est l'interface `store.Store` ([Ch. 9](../../chapitres/09-interfaces.md)) :
les handlers ne connaissent qu'elle. On passe du backend mémoire au backend SQL
**sans toucher au code web** — c'est l'inversion de dépendance en pratique.

> 💡 **Le même patron testable qu'au Projet 1** : `Run(ctx, args, stdout, stderr) int`.
> En injectant le `context` (pour l'arrêt), les flux et **en renvoyant** le code de
> retour, on garde `main` trivial et tout le reste testable.

---

## 3. Construit par étapes

1. **Modèle & interface** — `store.Task`, `store.TaskInput` (+ `Validate`) et
   l'interface `Store`. Chaque méthode prend un `context.Context`.
2. **Backend mémoire** — `MemStore` protégé par `sync.RWMutex`
   ([Ch. 21](../../chapitres/21-synchronisation.md)) ; tri par ID pour une
   pagination déterministe ([Ch. 32](../../chapitres/32-maps-hachage.md)).
3. **Routeur** — `http.ServeMux` avec motifs **méthode + `{id}`** (Go 1.22),
   l'identifiant lu par `r.PathValue("id")`.
4. **Handlers JSON** — décodage borné et strict (`MaxBytesReader`,
   `DisallowUnknownFields`), encodage, codes d'erreur cohérents.
5. **Middlewares** — `recoverPanic`, `requestID`, `logging` (`slog`), puis la
   protection **CSRF** (`http.CrossOriginProtection`).
6. **Cycle de vie** — `Run` : flags, logger multi-sorties, écoute en goroutine,
   **arrêt propre** via `Shutdown` sur signal.
7. **Backend SQL (optionnel)** — `SQLStore` sur `database/sql`, migrations
   embarquées, requêtes paramétrées sous contexte.
8. **Tests** — `httptest` de bout en bout + tests unitaires des middlewares et
   du store.

---

## 4. Routage par méthode (Go 1.22)

```go
mux.HandleFunc("GET /api/tasks/{id}", s.handleGet)
mux.HandleFunc("PUT /api/tasks/{id}", s.handleUpdate)
// ...
id := r.PathValue("id") // segment capturé par le wildcard {id}
```

Le `ServeMux` enrichi gère **gratuitement** ce que l'on codait à la main avant
1.22 :

- une route ne répond qu'à **sa** méthode ; une mauvaise méthode sur un chemin
  connu renvoie **`405 Method Not Allowed`** avec l'en-tête `Allow` ;
- un chemin inconnu renvoie **`404`** ;
- la précédence des motifs est gérée (le plus spécifique gagne).

---

## 5. Middlewares : une chaîne d'oignons

Un middleware est une fonction `http.Handler → http.Handler`. On les empile de
l'intérieur vers l'extérieur ; une requête les traverse dans l'ordre inverse :

```
recoverPanic → requestID → logging → CSRF → ServeMux → handler
```

- **`recoverPanic`** (le plus externe) transforme une panique en `500` propre
  ([Ch. 17](../../chapitres/17-panic-recover.md)) au lieu de couper la connexion.
- **`requestID`** génère (ou reprend) un `X-Request-Id`, le renvoie et le place
  dans le `context` ([Ch. 22](../../chapitres/22-context.md)).
- **`logging`** émet **une ligne `slog`** par requête, avec méthode, chemin,
  statut, taille, durée et identifiant — toutes corrélables :

    ```
    level=INFO msg="requête HTTP" method=POST path=/api/tasks status=201 \
        bytes=93 dur=454µs request_id=83ab74b4d2f2c2a9
    ```

- **CSRF** (`http.CrossOriginProtection`, **Go 1.25**) rejette en `403` les
  requêtes **cross-origin** non sûres (`POST`/`PUT`/`DELETE`) repérées via
  `Sec-Fetch-Site`, tout en laissant passer `GET`/`HEAD`/`OPTIONS`. On déclare
  les origines légitimes avec `-origins https://app.exemple.com`.

> 🆕 **`slog.NewMultiHandler` (Go 1.25)** : `newLogger` diffuse chaque
> enregistrement vers **deux** sorties — un handler **texte** lisible sur
> `stderr` et, si `-audit fichier.log` est donné, un handler **JSON** d'audit.

---

## 6. Arrêt propre (_graceful shutdown_)

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
// ... le serveur écoute dans une goroutine ...
<-ctx.Done()                 // SIGINT/SIGTERM reçu
shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
httpSrv.Shutdown(shutCtx)    // laisse finir les requêtes en cours, puis ferme
```

`Shutdown` cesse d'accepter de nouvelles connexions et **attend** la fin des
requêtes actives, dans la limite du délai. On borne cette attente avec un
**nouveau** `context` (l'original est déjà annulé par le signal).

---

## 7. Base de données : `database/sql` (optionnel)

`SQLStore` montre le câblage d'une vraie base : migrations **embarquées**
(`//go:embed`), requêtes **paramétrées** et **sous contexte**
(`ExecContext`/`QueryContext`, donc soumises aux délais/annulations).

`database/sql` est une API **abstraite** : il faut enregistrer un driver. Pour
rester en **pur Go** (cross-compilation sans cgo), on peut utiliser
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) :

```go
import (
	_ "modernc.org/sqlite" // enregistre le driver "sqlite" par effet de bord
	"example.com/tasksapi/internal/store"
)

st, err := store.NewSQLStore(ctx, "sqlite", "tasks.db")
```

> ⚠️ Les requêtes utilisent le marqueur `?` (SQLite/MySQL). Pour **PostgreSQL**
> (`pgx`, `lib/pq`), remplacer par `$1, $2, …`. Les colonnes `done` (INTEGER 0/1)
> et `created_at` (époque Unix) sont volontairement **portables** entre moteurs.
>
> Le backend par défaut reste `MemStore` : `tasksd` tourne et se teste **sans
> aucune base ni dépendance**.

---

## 8. Tests

```bash
cd projets/2-api-rest
go test -race ./...
```

- **API de bout en bout** (`tasks_test.go`) — via `httptest` : un `*Server`
  monté sur `MemStore`, requêtes `httptest.NewRequest` → `ResponseRecorder`.
  Couvre CRUD, filtres, et la table d'erreurs (`400/404/405/422`), plus la
  **protection CSRF** (POST cross-site → `403`, GET → `200`).
- **Middlewares** (`middleware_test.go`) — récupération de panique (`500`),
  génération/reprise de `X-Request-Id`.
- **Store** (`mem_test.go`) — CRUD, filtre/pagination, `ErrNotFound`,
  annulation par `context`. `sql_test.go` valide le découpage des migrations.

> 🧪 **À tester soi-même** : ajouter `PATCH /api/tasks/{id}` (mise à jour
> partielle) avec un `*string` pour `title` afin de distinguer « absent » de
> « vide », et son test `httptest`.

---

## 9. Build, lancement & cross-compilation

```bash
make run ARGS="-addr :9090"   # lance en local
make build                    # bin/tasksd (version = git describe)
make dist                     # dist/tasksd-<os>-<arch> pour 5 plateformes
```

Options : `-addr` (écoute), `-origins` (origines CSRF de confiance),
`-audit` (fichier JSON d'audit), `-shutdown-timeout`, `-version`.

---

## 10. Points de vigilance

- **Ordre de la chaîne** : `recoverPanic` doit être le **plus externe** pour
  rattraper aussi les paniques des autres middlewares ; `requestID` doit
  précéder `logging` pour que l'identifiant figure dans le log.
- **Réponse déjà commencée** : après le premier `Write`, l'en-tête de statut est
  parti. Un `recover` tardif ne peut plus écrire un `500` ; il coupe la
  connexion (cas accepté, signalé en commentaire).
- **Déterminisme de la liste** : l'itération d'une map est randomisée
  ([Ch. 32](../../chapitres/32-maps-hachage.md)) ; `List` **trie par ID** pour
  une pagination stable.
- **Borne du corps** : sans `MaxBytesReader`, un client peut épuiser la mémoire.
  Le décodage strict (`DisallowUnknownFields`) attrape aussi les fautes de frappe
  dans les payloads.
- **`context` partout** : chaque méthode du `Store` reçoit le `context` de la
  requête ; un client qui abandonne (ou un délai dépassé) annule le travail en
  aval ([Ch. 22](../../chapitres/22-context.md)).

---

## 11. Pour aller plus loin

- Ajouter une **authentification** (jeton `Authorization: Bearer …`) en
  middleware, avant le routeur.
- Brancher `SQLStore` sur PostgreSQL et écrire un **test d'intégration** derrière
  un build tag `//go:build integration`.
- Négocier `application/problem+json` (RFC 9457) pour les erreurs.
- Exposer des **métriques** (compteur de requêtes par statut) et un endpoint
  `/metrics`.
- Mentionner **`encoding/json/v2`** (expérimental, `GOEXPERIMENT=jsonv2`) :
  décodage plus strict et plus rapide, API `MarshalWrite`/`UnmarshalRead`.

---

## 📌 À retenir

- Un `ServeMux` (Go 1.22) suffit pour une API REST : **méthode + `{id}`** +
  `PathValue`, avec `405`/`404` gérés pour soi.
- Un **middleware** = `http.Handler → http.Handler` ; on les **chaîne** comme des
  oignons (recover en tête, CSRF près du routeur).
- **`slog`** donne un log structuré ; `NewMultiHandler` (1.25) diffuse vers
  plusieurs sorties ; `CrossOriginProtection` (1.25) couvre le CSRF.
- Dépendre d'une **interface** (`Store`) découple le web de la persistance :
  mémoire en test, `database/sql` en production.
- L'**arrêt propre** (`signal.NotifyContext` + `Server.Shutdown`) laisse finir
  les requêtes en cours avant de fermer.
