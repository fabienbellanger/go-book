# 48 — Processus, signaux & ligne de commande

> **Objectif** — Piloter un programme depuis l'extérieur avec la seule
> bibliothèque standard : analyser des arguments (`flag`), lancer et contrôler des
> sous-processus (`os/exec`), et réagir proprement aux signaux du système
> (`os/signal`) pour un arrêt gracieux.

> **Prérequis** — [Ch. 5](05-fonctions.md) (fonctions), [Ch. 10](10-erreurs.md)
> (erreurs, `errors.As`), [Ch. 20](20-channels-select.md) (`select`),
> [Ch. 22](22-context.md) (`context`).

---

## Introduction

Un programme ne vit pas seul : il reçoit des **arguments** au démarrage, il
**délègue** parfois du travail à d'autres exécutables, et il doit **s'arrêter
proprement** quand le système le lui demande. Trois paquets de la bibliothèque
standard couvrent ces trois surfaces d'interaction avec le monde extérieur.

```
   monde extérieur                        programme Go
   +----------------------+               +---------------------------+
   |  arguments CLI        |  --flag--->   |  configuration            |
   |  autre exécutable     |  <-exec-->    |  sous-processus piloté    |
   |  signaux (SIGINT...)  |  --signal->   |  arrêt propre / rechargt. |
   +----------------------+               +---------------------------+
```

Aucune dépendance tierce n'est nécessaire : `flag`, `os/exec` et `os/signal`
suffisent pour un outil robuste.

---

## `flag` : analyser la ligne de commande

Le paquet `flag` décrit les options attendues, puis analyse `os.Args`. Deux styles
coexistent : le `flag` **global** (rapide pour un `main` simple) et un
`flag.FlagSet` **dédié** (indispensable dès qu'on veut tester ou gérer des
sous-commandes).

```go
fs := flag.NewFlagSet("greet", flag.ContinueOnError)
name := fs.String("name", "Go", "nom à saluer")     // *string
count := fs.Int("count", 1, "répétitions")           // *int
verbose := fs.Bool("verbose", false, "verbeux")      // *bool
timeout := fs.Duration("timeout", 5*time.Second, "délai") // *time.Duration
err := fs.Parse(os.Args[1:])                          // renvoie l'erreur, ne quitte pas
```

Le troisième argument du `FlagSet` choisit la **politique d'erreur** :

| Mode              | Comportement sur erreur de parsing | Usage                     |
| ----------------- | ---------------------------------- | ------------------------- |
| `ExitOnError`     | affiche l'usage puis `os.Exit(2)`  | défaut du `flag` global   |
| `ContinueOnError` | **renvoie** l'erreur à l'appelant  | code testable, composable |
| `PanicOnError`    | `panic`                            | rare                      |

> 💡 Pour une fonction `parseFlags` **testable**, utilisez toujours
> `ContinueOnError` : un test peut alors vérifier l'erreur au lieu de voir le
> process de test se terminer brutalement. C'est le choix du code du chapitre.

`flag.Func("nom", "aide", fn)` branche une fonction de validation/accumulation à
chaque occurrence — utile pour un flag répétable (`-header a -header b`) ou un
type maison. `fs.Var` accepte n'importe quelle valeur implémentant `flag.Value`
(méthodes `String()` et `Set(string) error`).

### Sous-commandes

`flag` n'a pas de notion native de sous-commande (comme `git commit`), mais le
motif est simple : router sur `os.Args[1]`, puis un `FlagSet` par sous-commande.

```go
switch os.Args[1] {
case "add":
    fs := flag.NewFlagSet("add", flag.ExitOnError)
    // ... flags propres à "add"
    fs.Parse(os.Args[2:])
case "list":
    // ...
}
```

⚠️ **Piège classique** : `Parse` **s'arrête au premier argument non-flag**. Les
options doivent donc **précéder** les arguments positionnels. `monoutil fichier.txt
-verbose` ignore `-verbose` (traité comme un second argument positionnel), alors
que `monoutil -verbose fichier.txt` fonctionne. Après `Parse`, `fs.Args()` donne
les positionnels restants.

> 💡 Des frameworks tiers (`spf13/cobra`, `urfave/cli`) offrent sous-commandes
> imbriquées, complétion shell et aide riche. Ils sont justifiés pour une CLI
> vaste, mais `flag` suffit très souvent — et sans dépendance. Le
> [Projet 1](../projets/1-cli/) construit une CLI complète à sous-commandes sur le
> seul `flag`.

---

## `os/exec` : lancer un sous-processus

`os/exec` exécute un autre programme. **Préférez toujours `exec.CommandContext`** à
`exec.Command` : le `context` permet d'imposer un timeout et d'annuler — sans quoi
un enfant lent bloque l'appelant indéfiniment.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
cmd := exec.CommandContext(ctx, "go", "version") // args SÉPARÉS, aucun shell
out, err := cmd.Output()                          // capture stdout
```

Trois façons de lancer, selon ce qu'on veut récupérer :

| Méthode            | Renvoie                        | Quand l'utiliser                 |
| ------------------ | ------------------------------ | -------------------------------- |
| `Run()`            | seulement l'erreur             | on ne veut que le code de sortie |
| `Output()`         | stdout (`[]byte`) + erreur     | on veut la sortie standard       |
| `CombinedOutput()` | stdout **+** stderr entremêlés | diagnostic, logs bruts           |

### Distinguer les modes d'échec

Une erreur d'`Output()` recouvre deux cas très différents. `errors.As` (🔁
[Ch. 10](10-erreurs.md)) les sépare :

```go
out, err := cmd.Output()
if err != nil {
    var ee *exec.ExitError
    if errors.As(err, &ee) {
        // la commande a TOURNÉ mais renvoyé un code != 0 ; ee.Stderr contient
        // son flux d'erreur, ee.ExitCode() son code.
        return fmt.Errorf("code %d : %s", ee.ExitCode(), ee.Stderr)
    }
    // sinon : binaire introuvable, ctx expiré, permission refusée...
    return err
}
```

Autres réglages utiles : `cmd.Env = append(os.Environ(), "KEY=val")` (environnement
explicite), `cmd.Dir` (répertoire de travail), `cmd.StdinPipe()` /
`cmd.StdoutPipe()` (streaming au lieu de tout charger en mémoire), et
`exec.LookPath("git")` pour vérifier qu'un exécutable existe dans le `PATH`.

⚠️ **Sécurité** : `os/exec` **n'appelle aucun shell**. Les arguments sont passés
tels quels au programme, donc `exec.Command("git", "clone", url)` ne peut pas subir
d'injection, même si `url` contient `;` ou `$(...)`. Ne réintroduisez jamais un
shell (`sh -c "..."`) autour d'une entrée externe (🔁
[Ch. 47](47-securite-supply-chain.md)).

---

## `os/signal` : réagir aux signaux

Un signal est une notification asynchrone du système : `SIGINT` (Ctrl-C),
`SIGTERM` (`docker stop`, Kubernetes), `SIGHUP` (rechargement de config par
convention). Par défaut, la plupart **terminent** le process. `signal.Notify`
intercepte ces signaux et les livre sur un canal, à vous d'agir.

```go
ch := make(chan os.Signal, 1)        // BUFFERISÉ (au moins 1)
signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
defer signal.Stop(ch)                 // détache le canal quand on a fini
sig := <-ch                           // bloque jusqu'au signal
```

⚠️ Le canal **doit être bufferisé** (capacité ≥ 1). Le runtime ne bloque jamais
pour livrer un signal : si personne ne lit et que le tampon est plein, le signal
est **perdu**. Une capacité de 1 suffit pour l'arrêt (on ne s'arrête qu'une fois).

### `signal.NotifyContext` : l'idiome d'arrêt propre 🆕

Depuis Go 1.16, `signal.NotifyContext` relie directement les signaux à un
`context` : à la réception d'un signal écouté, le context est **annulé**. C'est la
façon moderne d'implémenter un arrêt gracieux, car tout le code aval qui accepte
déjà un `context` (serveur HTTP, requêtes DB...) s'arrête sans plomberie
supplémentaire.

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

srv := &http.Server{Addr: ":8080", Handler: mux}
go srv.ListenAndServe()

<-ctx.Done()                          // Ctrl-C ou SIGTERM reçu
shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
defer c()
srv.Shutdown(shutdownCtx)             // draine les requêtes en cours
```

```
  démarrage --> service en cours --.
                                    |  SIGINT / SIGTERM
                                    v
                            ctx.Done() se ferme
                                    |
                                    v
                    arrêt propre (Shutdown, drain, close) --> exit 0
```

Ce motif est repris par le serveur du [Projet 2](../projets/2-api-rest/) et
détaillé côté HTTP au [Ch. 45](45-net-http.md). Pour un rechargement de config à
chaud, écoutez `syscall.SIGHUP` sur un canal séparé et rechargez sans quitter.

---

## ⚠️ Pièges

- **Flags après positionnels** : `Parse` s'arrête au premier non-flag ; placez les
  options avant les arguments.
- **`exec.Command` sans context** : un enfant qui ne rend jamais la main bloque le
  parent. Utilisez `CommandContext` + timeout.
- **Confondre les deux échecs d'`Output()`** : code de sortie non nul
  (`*exec.ExitError`, stderr disponible) vs binaire introuvable / ctx expiré.
- **Canal de signal non bufferisé** : signal potentiellement perdu.
- **Oublier `stop()` / `signal.Stop`** : le handler reste installé et peut
  perturber d'autres parties du programme (ou d'autres tests).

---

## 🧪 À tester soi-même

Le code du chapitre (`code/ch48-processus/`) expose `parseFlags` (analyse
testable), `capture` (sous-processus borné avec gestion de `*exec.ExitError`),
`notify` (écoute de signaux) et `serve` (arrêt sur annulation du context). Les
tests sont **hermétiques** : `capture` interroge `go version` (binaire garanti
présent sous `go test`) et le test de signal s'envoie `SIGUSR1` à lui-même.

```bash
cd code && go test ./ch48-processus/...
```

**À essayer :**

1. Appelez `parseFlags("greet", []string{"fichier.txt", "-verbose"})` et observez
   que `-verbose` reste à `false` : il est vu comme un second positionnel (piège
   des flags après arguments). Inversez l'ordre pour le corriger.
2. Dans `TestCapture`, remplacez `"go", "version"` par une commande inexistante
   (`"binaire-absent"`) : l'erreur n'est **pas** un `*exec.ExitError` (le binaire
   n'a jamais tourné). Vérifiez-le avec `errors.As`.
3. Ramenez le timeout de `capture` à `1*time.Nanosecond` sur une commande plus
   lente et constatez que le context annule le sous-processus.
4. Ajoutez `syscall.SIGHUP` à l'appel `notify` et envoyez-vous ce signal : montrez
   qu'on peut router « recharger » (SIGHUP) et « arrêter » (SIGTERM) différemment.

---

## 📌 À retenir

- **`flag`** couvre l'essentiel des CLI ; un `FlagSet` en `ContinueOnError` rend
  l'analyse testable, et le motif `os.Args[1]` + `FlagSet` gère les sous-commandes.
- **`os/exec`** : toujours `CommandContext` avec timeout ; `Output`/`CombinedOutput`
  selon les flux voulus ; `*exec.ExitError` distingue « code non nul » du reste ;
  arguments séparés = pas d'injection.
- **`os/signal`** : canal **bufferisé** pour `Notify` ; `signal.NotifyContext` est
  l'idiome d'arrêt propre, car il propage l'annulation à tout le code aval.
- Ces trois briques sont **100 % stdlib** — une dépendance tierce n'est justifiée
  que pour une CLI vraiment vaste.

## 🔁 Pour aller plus loin

- [Ch. 1](01-installation-toolchain.md) : variables d'environnement `GO*`, structure d'un binaire.
- [Ch. 45](45-net-http.md) : arrêt gracieux d'un serveur HTTP (`Shutdown`).
- [Ch. 47](47-securite-supply-chain.md) : pourquoi éviter le shell autour d'une entrée externe.
- [Projet 1 — Outil CLI `txtkit`](../projets/1-cli/) : une CLI complète bâtie sur `flag`.
- [Projet 2 — API REST](../projets/2-api-rest/) : arrêt propre via `signal.NotifyContext`.
- Doc : [`pkg.go.dev/flag`](https://pkg.go.dev/flag), [`os/exec`](https://pkg.go.dev/os/exec), [`os/signal`](https://pkg.go.dev/os/signal).
