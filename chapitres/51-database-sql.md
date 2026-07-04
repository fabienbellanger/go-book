# 51 — `database/sql`

> **Objectif** — Interroger une base de données relationnelle avec l'API standard
> `database/sql` : comprendre que `*sql.DB` est un **pool de connexions** (et non une
> connexion), lire une ou plusieurs lignes, écrire avec des requêtes **paramétrées**
> (anti-injection), gérer les transactions, propager un `context`, et régler le pool
> pour la production.
>
> **Prérequis** — [Ch. 10 — Erreurs](10-erreurs.md) (`errors.Is`, `%w`),
> [Ch. 22 — `context`](22-context.md), [Ch. 02 — Structure d'un programme](02-structure-programme.md)
> (import blanc, `init`).

---

## Introduction

Go n'embarque **aucun** moteur de base de données. Ce qu'il fournit, dans `database/sql`,
c'est une **API abstraite** : un contrat commun (`Query`, `Exec`, `Begin`…) que tout moteur
peut implémenter via un **driver**. Le code applicatif parle à `database/sql` ; le driver
(SQLite, PostgreSQL, MySQL…) parle au moteur. On change de base sans réécrire la logique.

```
  votre code        database/sql          driver              moteur
  +----------+     +-------------+     +-----------+     +--------------+
  | Query    | --> | pool, Scan, | --> | traduit   | --> | PostgreSQL,  |
  | Exec, Tx |     | conversions |     | le SQL    |     | SQLite, ...  |
  +----------+     +-------------+     +-----------+     +--------------+
       ne connaît QUE database/sql          import blanc « _ »
```

L'exemple complet est dans [`code/ch51-database-sql/`](../code/ch51-database-sql/). Pour rester
**exécutable hors ligne et sans dépendance**, il embarque un **driver factice en mémoire**
(`memdb.go`) : il ne reconnaît qu'un jeu fixe de requêtes, mais suffit à montrer que le code
appelant ne dépend d'aucun moteur particulier. En production, on importe un vrai driver
(voir plus bas).

## Le driver : un import « blanc »

Un driver s'enregistre lui-même auprès de `database/sql` dans sa fonction `init`
([Ch. 02](02-structure-programme.md)). On le déclenche par un **import blanc** — importé
pour son seul effet de bord :

```go
import (
	"database/sql"

	_ "modernc.org/sqlite" // son init() appelle sql.Register("sqlite", ...)
)
```

Côté driver, l'enregistrement tient en une ligne (c'est ce que fait notre `memdb.go`) :

```go
// code/ch51-database-sql/memdb.go
func init() {
	sql.Register("memdb", &memDriver{stores: map[string]*store{}})
}
```

> 💡 Le `_` est **obligatoire** : sans lui, le compilateur refuserait un import « inutilisé ».
> Avec lui, on importe uniquement pour exécuter `init` ([Ch. 12](12-packages-modules.md)).

Quelques drivers courants, tous compatibles `database/sql` :

| Base       | Driver recommandé          | cgo ? |
| ---------- | -------------------------- | ----- |
| PostgreSQL | `github.com/jackc/pgx`     | non   |
| SQLite     | `modernc.org/sqlite`       | non   |
| SQLite     | `github.com/mattn/go-sqlite3` | oui   |
| MySQL      | `github.com/go-sql-driver/mysql` | non |

## `*sql.DB` est un **pool**, pas une connexion

L'erreur mentale la plus fréquente : croire que `sql.Open` ouvre une connexion. **Non.**
`sql.Open` valide seulement le nom du driver et **prépare un pool paresseux** : aucune
connexion n'est établie tant qu'on n'exécute pas une requête.

```go
// code/ch51-database-sql/main.go
db, err := sql.Open("memdb", dsn) // NE SE CONNECTE PAS
if err != nil {
	return nil, err // erreur seulement si le driver est inconnu
}
if err := db.PingContext(ctx); err != nil { // force une vraie connexion
	db.Close()
	return nil, err
}
```

Conséquences pratiques :

- **`*sql.DB` est sûr en concurrence** et conçu pour être **partagé** par toute
  l'application. On l'ouvre **une fois** au démarrage, on le passe aux dépendances — jamais
  un `sql.Open` par requête HTTP (🔁 [Ch. 54 — Architecture](54-architecture.md)).
- Pour **vérifier** que la base répond au démarrage, utilisez `Ping`/`PingContext`.
- `db.Close()` ferme le pool entier : à faire au shutdown, pas après chaque requête.

### Régler le pool

Un pool mal réglé est une cause classique d'incident en production (connexions épuisées,
saturation du serveur SQL). Quatre réglages :

```go
// code/ch51-database-sql/main.go
db.SetMaxOpenConns(10)                  // plafond de connexions simultanées
db.SetMaxIdleConns(5)                   // connexions gardées au chaud
db.SetConnMaxLifetime(30 * time.Minute) // recyclage périodique
db.SetConnMaxIdleTime(5 * time.Minute)  // fermeture des connexions oisives
```

| Réglage               | Rôle                                       | Piège si mal réglé                          |
| --------------------- | ------------------------------------------ | ------------------------------------------- |
| `SetMaxOpenConns`     | borne le nombre de connexions ouvertes     | `0` = illimité → peut saturer le serveur    |
| `SetMaxIdleConns`     | connexions réutilisables sans reconnexion  | trop bas → reconnexions coûteuses           |
| `SetConnMaxLifetime`  | âge max d'une connexion avant recyclage    | trop long → connexions « mortes » côté SQL  |
| `SetConnMaxIdleTime`  | durée max d'oisiveté avant fermeture       | libère les ressources côté serveur          |

> 💡 Règle de départ : `MaxOpenConns` aligné sur la capacité du serveur SQL et le nombre de
> workers ; `MaxIdleConns` ≤ `MaxOpenConns` ; une `ConnMaxLifetime` de quelques minutes à
> quelques dizaines de minutes pour tolérer les redémarrages du serveur et les bascules.

## Lire une ligne : `QueryRow` + `Scan`

Pour un résultat à **une seule ligne**, `QueryRowContext(...).Scan(...)` est direct. `Scan`
remplit les variables passées **par pointeur**, dans l'ordre des colonnes du `SELECT` :

```go
// code/ch51-database-sql/main.go
var u User
err := db.QueryRowContext(ctx,
	"select id, name, email from users where id = ?", id).
	Scan(&u.ID, &u.Name, &u.Email)
if errors.Is(err, sql.ErrNoRows) {
	return User{}, fmt.Errorf("utilisateur %d introuvable : %w", id, err)
}
if err != nil {
	return User{}, err
}
```

L'**absence** de ligne n'est pas une erreur d'exécution : `Scan` renvoie la sentinelle
**`sql.ErrNoRows`**, qu'on distingue avec `errors.Is` ([Ch. 10](10-erreurs.md)). La confondre
avec une vraie panne, ou l'ignorer, sont deux fautes fréquentes.

## Lire plusieurs lignes : `Query` + boucle

Pour N lignes, `QueryContext` renvoie un **curseur** `*sql.Rows` qu'on avance ligne par ligne.
Trois gestes vont **toujours** ensemble :

```go
// code/ch51-database-sql/main.go
rows, err := db.QueryContext(ctx, "select id, name, email from users")
if err != nil {
	return nil, err
}
defer rows.Close() // (1) libère la connexion au pool, même en cas d'erreur

var users []User
for rows.Next() { // (2) avance ; false à la fin OU à la première erreur
	var u User
	if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
		return nil, err
	}
	users = append(users, u)
}
return users, rows.Err() // (3) erreur SURVENUE PENDANT l'itération
```

```
  QueryContext           rows.Next()==true          rows.Next()==false
  +----------+  Scan     +----------+  Scan          +--------------+
  | ligne 1  | -------> | ligne 2   | -------> ... -> | fin (ou err) |
  +----------+           +----------+                 +--------------+
       defer rows.Close()  tout au long          rows.Err() APRÈS la boucle
```

- **(1) `defer rows.Close()`** : tant que `Rows` est ouvert, il **retient une connexion** du
  pool. L'oublier fuit des connexions jusqu'à épuisement.
- **(2)** `rows.Next()` renvoie `false` à la fin **et** à la première erreur — impossible de
  les distinguer ici.
- **(3) `rows.Err()`** est donc **obligatoire** après la boucle : c'est le seul endroit où une
  erreur de lecture (réseau coupé en plein curseur) remonte.

## Écrire : `Exec` + `sql.Result`

Les requêtes sans résultat (`INSERT`, `UPDATE`, `DELETE`) passent par `ExecContext`, qui
renvoie un `sql.Result` :

```go
// code/ch51-database-sql/main.go
res, err := db.ExecContext(ctx,
	"insert into users(name, email) values (?, ?)", name, email)
if err != nil {
	return 0, err
}
id, err := res.LastInsertId() // dernier id auto-incrémenté (si le driver le sait)
```

`res.RowsAffected()` donne le nombre de lignes touchées (utile pour un `UPDATE`/`DELETE`).

> ⚠️ `LastInsertId` et `RowsAffected` **dépendent du driver** : PostgreSQL, par exemple, ne
> renvoie pas de `LastInsertId` (on utilise `INSERT ... RETURNING id` + `QueryRow`). Vérifiez
> le comportement de votre driver.

## Requêtes **paramétrées** : jamais de concaténation

Les `?` (ou `$1`, `$2`… selon le driver) sont des **placeholders**. Les valeurs voyagent
**séparément** de la requête : le moteur ne les interprète jamais comme du SQL. C'est la
protection **native et suffisante** contre l'injection SQL.

```go
// CORRECT — la valeur est un paramètre, jamais du code
db.QueryRowContext(ctx, "select id, name, email from users where id = ?", id)

// DANGER — concaténation : injection SQL possible
q := "select ... where name = '" + name + "'" // ⚠️ NE JAMAIS FAIRE
```

> ⚠️ Le style de placeholder est **propre au driver** : `?` (SQLite, MySQL), `$1` (PostgreSQL/pgx),
> `@p1` (SQL Server). C'est l'un des rares endroits où le SQL n'est pas portable tel quel.

## Requêtes préparées (`Prepare`)

`Prepare` compile une requête **une fois** côté serveur ; on la réexécute ensuite avec des
paramètres différents. Utile sur un **chemin chaud** qui répète la même requête :

```go
stmt, err := db.PrepareContext(ctx, "select id, name, email from users where id = ?")
if err != nil { /* ... */ }
defer stmt.Close() // ⚠️ un Stmt retient des ressources : toujours le fermer
for _, id := range ids {
	_ = stmt.QueryRowContext(ctx, id) // réutilise la requête compilée
}
```

> 💡 Pour une requête **ponctuelle**, `QueryContext`/`ExecContext` préparent et libèrent en
> interne : inutile de préparer soi-même. Réservez `Prepare` aux requêtes **répétées**.

## Transactions

Une transaction regroupe plusieurs requêtes en une unité **atomique** : tout est validé
(`Commit`) ou rien (`Rollback`). `BeginTx` fixe une connexion et renvoie un `*sql.Tx` dont les
méthodes (`ExecContext`, `QueryContext`…) s'exécutent **dans** la transaction.

```go
// code/ch51-database-sql/main.go
tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
if err != nil {
	return err
}
defer tx.Rollback() // annule si on n'atteint pas le Commit ; no-op sinon

if _, err := tx.ExecContext(ctx, "update users set name = ? where id = ?", name1, id1); err != nil {
	return err
}
if _, err := tx.ExecContext(ctx, "update users set name = ? where id = ?", name2, id2); err != nil {
	return err
}
return tx.Commit()
```

Le patron **`defer tx.Rollback()`** est l'idiome clé : si une erreur interrompt la fonction
avant le `Commit`, le `Rollback` différé annule tout. Après un `Commit` réussi, ce même
`Rollback` est un **no-op** (il renvoie `sql.ErrTxDone`, qu'on ignore volontairement).

`sql.TxOptions` permet de choisir le **niveau d'isolation** (`sql.LevelReadCommitted`,
`sql.LevelSerializable`…) et un mode lecture seule (`ReadOnly: true`) — leur effet réel dépend
du moteur. Le [Projet 2 — API REST](../projets/2-api-rest/) applique ce patron à un cas concret.

## `context` : annuler et borner une requête

Toutes les méthodes `*Context` propagent un `context.Context` ([Ch. 22](22-context.md)). Une
requête lente est **interrompue** dès que le contexte est annulé (client HTTP parti, timeout) :

```go
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()
rows, err := db.QueryContext(ctx, "select id, name, email from users")
```

> 💡 Préférez **systématiquement** les variantes `QueryContext`/`ExecContext`/`BeginTx` à leurs
> versions sans contexte : c'est ce qui rend un service annulable et résistant aux requêtes qui
> traînent. Les versions sans `ctx` (`db.Query`, `db.Exec`) équivalent à passer
> `context.Background()`.

## Valeurs `NULL`

En SQL, une colonne peut valoir `NULL`. `Scan` **ne peut pas** écrire `NULL` dans un `string`
ou un `int` : il faut un type qui distingue « absent » de « vide ».

```go
var name sql.NullString            // { String string; Valid bool }
_ = row.Scan(&name)
if name.Valid { use(name.String) } // sinon : la valeur était NULL

var age sql.Null[int] // 🆕 Go 1.22 : version générique, pour n'importe quel type
```

| Approche             | Quand l'utiliser                                      |
| -------------------- | ----------------------------------------------------- |
| `sql.NullString`, `sql.NullInt64`… | colonnes nullables, types courants        |
| `sql.Null[T]` (1.22) | 🆕 générique, tout type scannable                     |
| `*string`, `*int`    | pointeur `nil` si `NULL` — pratique en JSON           |

## 🆕 Go récent

- **`sql.Null[T]`** (Go 1.22) généralise `sql.NullString`/`NullInt64` à n'importe quel type
  scannable — une seule forme au lieu d'une par type.
- Les drivers modernes **purs Go** (`modernc.org/sqlite`, `jackc/pgx`) suppriment le besoin de
  cgo : compilation croisée simple et binaire autonome (🔁 [Ch. 46](46-embed-build-deploiement.md)).

## ⚠️ Pièges

- **Oublier `rows.Close()`** : chaque `*sql.Rows` ouvert retient une connexion du pool.
  L'oublier fuit des connexions jusqu'à l'épuisement (`SetMaxOpenConns` atteint → tout se
  bloque). Réflexe : `defer rows.Close()` juste après un `QueryContext` réussi.
- **Oublier `rows.Err()`** : `rows.Next()` renvoie `false` aussi bien à la fin qu'à la première
  erreur. Sans `rows.Err()` après la boucle, une lecture interrompue passe pour une fin normale.
- **Concaténer le SQL** au lieu de paramétrer : injection SQL. Utilisez **toujours** les
  placeholders (`?`, `$1`…).
- **Confondre `sql.Open` et une connexion** : `sql.Open` ne se connecte pas ; utilisez `Ping`
  pour valider, et **partagez** le `*sql.DB` (ne pas en créer un par requête).
- **Garder une transaction ouverte trop longtemps** : elle retient une connexion **et** des
  verrous côté SQL. Faites-la courte ; n'y mettez pas d'appels réseau lents.
- **Scanner un `NULL` dans un `string`** : `Scan` échoue. Utilisez `sql.NullString` /
  `sql.Null[T]` / un pointeur.
- **Réutiliser un `*sql.Rows` après `Close`** : le curseur est mort. De même, ne gardez pas les
  slices renvoyés par un `Scan` de `[]byte` au-delà du prochain `Next` sans les copier.

## ⚡ Performance

- **Régler le pool** (`SetMaxOpenConns`, `SetMaxIdleConns`) est le premier levier : trop peu de
  connexions sérialise les requêtes, trop en sature le serveur.
- **Préparer les requêtes répétées** sur les chemins chauds évite de recompiler la requête à
  chaque appel — mais un `Stmt` retient des ressources, refermez-le.
- **Éviter le N+1** : une requête par élément d'une liste tue les performances. Préférez un
  `IN (...)` ou une jointure, en une seule requête.
- **`QueryRow` plutôt que `Query`** quand on attend une seule ligne : pas de curseur à gérer ni
  à fermer.
- **Scanner dans `[]byte`** plutôt que `string` évite une allocation quand on peut réutiliser le
  tampon (🔁 [Ch. 31](31-strings-profondeur.md)).

## 🧪 À tester soi-même

Dans [`code/ch51-database-sql/`](../code/ch51-database-sql/) :

```bash
cd code && go test ./ch51-database-sql/
```

Les tests exercent `QueryRow` (+ `sql.ErrNoRows`), une lecture multi-lignes, `RowsAffected`,
une transaction validée puis une annulée, et un contexte annulé. Ajoutez un test qui insère 3
utilisateurs **dans une transaction** puis fait un `Rollback` : `listUsers` doit alors n'en
retourner aucun.

---

## 📌 À retenir

- `database/sql` est une **API abstraite** ; un **driver** (importé « blanc ») la relie à un
  moteur. Le code applicatif ne dépend d'aucune base en particulier.
- **`*sql.DB` est un pool** de connexions, sûr en concurrence, à **ouvrir une fois** et à
  partager. `sql.Open` ne connecte pas — utilisez `Ping` ; réglez le pool
  (`SetMaxOpenConns`…).
- **Une ligne** : `QueryRowContext(...).Scan(...)`, avec `errors.Is(err, sql.ErrNoRows)`.
  **N lignes** : `QueryContext` + `defer rows.Close()` + boucle + **`rows.Err()`**.
- **Écrire** : `ExecContext` → `sql.Result`. **Toujours paramétrer** (`?`/`$1`), jamais
  concaténer (injection).
- **Transactions** : `BeginTx` + `defer tx.Rollback()` + `Commit`. Passer un `context` partout
  rend le service annulable.
- **`NULL`** : `sql.NullString` / `sql.Null[T]` (🆕 1.22) / pointeur.

## 🔁 Pour aller plus loin

- [Ch. 22 — `context`](22-context.md) : annulation et deadlines propagées aux requêtes.
- [Ch. 10 — Erreurs](10-erreurs.md) : `errors.Is` pour `sql.ErrNoRows`, `%w`.
- [Ch. 02 — Structure d'un programme](02-structure-programme.md) : `init` et import blanc du driver.
- [Ch. 54 — Architecture](54-architecture.md) : où placer le `*sql.DB`, injection de dépendances.
- [Projet 2 — API REST](../projets/2-api-rest/) : un `SQLStore` réel, transactions comprises.
- Référence : [`pkg.go.dev/database/sql`](https://pkg.go.dev/database/sql),
  [`go.dev/doc/database`](https://go.dev/doc/database) (guide officiel).
