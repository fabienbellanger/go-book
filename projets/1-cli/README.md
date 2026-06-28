# Projet 1 — Outil CLI : `txtkit`

> **Objectif** — Construire un véritable outil en ligne de commande, idiomatique et
> distribuable : sous-commandes, lecture de fichiers **ou** de stdin, configuration en
> couches, **concurrence bornée**, sortie formatée, codes de retour Unix, tests, et
> **cross-compilation**.
>
> **Réinvestit** — [Ch. 5 Fonctions](../../chapitres/05-fonctions.md),
> [Ch. 7 Maps & strings](../../chapitres/07-maps-strings.md),
> [Ch. 11 Généricité](../../chapitres/11-genericite.md),
> [Ch. 13 Tests](../../chapitres/13-tests-outillage.md),
> [Ch. 19 Goroutines](../../chapitres/19-goroutines.md),
> [Ch. 21 Synchronisation](../../chapitres/21-synchronisation.md).

---

## 1. Cahier des charges

`txtkit` est une boîte à outils de traitement de fichiers texte, à la manière de `wc`,
mais avec **deux sous-commandes** et du **parallélisme** :

| Sous-commande | Rôle                                                              |
| ------------- | ---------------------------------------------------------------- |
| `count`       | Compte **lignes, mots, runes, octets** de chaque entrée.         |
| `freq`        | Affiche les **mots les plus fréquents**, toutes entrées confondues. |
| `version`     | Affiche la version (injectée à la compilation).                  |
| `help`        | Affiche l'aide.                                                  |

Contraintes :

- **Entrées** : une liste de fichiers en arguments, ou **stdin** si la liste est vide
  (`txtkit count < article.txt`).
- **Concurrence** : les fichiers sont traités par un **pool de workers borné** (`-j N`).
- **Configuration en couches** : valeur par défaut < variable d'environnement < flag.
- **Codes de retour** : `0` succès · `1` erreur de traitement (fichier illisible) ·
  `2` erreur d'usage (mauvais flag, sous-commande inconnue).
- **Distribution** : binaire statique cross-compilable (Linux/macOS/Windows, amd64/arm64).

```
$ printf 'Go go GO\nrust rust python\n' | txtkit count
  2  6  26  26 -

$ txtkit freq -n 3 *.md
     128  the
      97  go
      54  func
```

---

## 2. Architecture

```
                 main.go
                   |  os.Exit(cli.Run(os.Args[1:], stdin, stdout, stderr))
                   v
            +--------------+        Run renvoie le CODE DE RETOUR ; il reçoit
            |  cli.Run     |        ses flux en paramètres => 100 % testable
            +--------------+
                   | dispatch sur args[0]
        +----------+----------+------------+
        v          v          v            v
     count       freq      version       help
        |          |
        |  sourcesFrom(args, stdin)  -> []source   (fichiers OU stdin)
        |          |
        v          v
   +-----------------------------+
   |  mapBounded(srcs, j, work)  |   pool borné : au plus j goroutines,
   +-----------------------------+   résultats dans l'ordre des entrées
        |          |
   countSource  freqSource         un worker par source
        |          |
        v          v
    tabwriter   fusion (fan-in) + tri        sortie formatée
```

Le cœur réutilisable est `mapBounded`, un **worker pool générique** ([Ch. 11](../../chapitres/11-genericite.md),
[Ch. 19](../../chapitres/19-goroutines.md)) :

```go
// au plus n goroutines simultanées, résultats préservés dans l'ordre
func mapBounded[T, R any](items []T, n int, f func(T) R) []R {
	out := make([]R, len(items))
	sem := make(chan struct{}, n) // sémaphore comptant
	var wg sync.WaitGroup
	for i, it := range items {
		sem <- struct{}{}       // bloque si n workers tournent déjà
		wg.Go(func() {          // WaitGroup.Go (Go 1.25)
			defer func() { <-sem }()
			out[i] = f(it)
		})
	}
	wg.Wait()
	return out
}
```

> 💡 **Le patron testable** : `Run(args, stdin, stdout, stderr) int`. En injectant les
> flux et en **renvoyant** le code de retour (au lieu d'appeler `os.Exit` partout), on
> teste l'outil entier sans lancer de processus ni toucher au disque. `main` se réduit à
> `os.Exit(cli.Run(...))`.

---

## 3. Construit par étapes

1. **Squelette & dispatch** — `main.go` minimal + `cli.Run` qui aiguille sur `args[0]`
   via un `switch`. Sous-commande inconnue ⇒ usage + code `2`.
2. **Sources** — `sourcesFrom` : liste de fichiers ⇒ `os.Open` paresseux par source ;
   liste vide ⇒ stdin (nom affiché `-`). Chaque `source` porte une fonction `open()`.
3. **`count`** — un `flag.FlagSet` dédié (`-j`, `-total`), parcours **rune par rune**
   (`bufio.Reader.ReadRune`) pour compter octets/runes/mots/lignes en un seul passage,
   sortie alignée par `text/tabwriter`.
4. **Concurrence** — on remplace la boucle séquentielle par `mapBounded` : le `-j`
   contrôle le nombre de fichiers lus en parallèle.
5. **`freq`** — chaque source produit sa table `map[string]int` (fan-out), puis fusion
   en une table globale (fan-in), tri par fréquence décroissante (alphabétique à
   égalité, pour un résultat **déterministe**), top `-n`.
6. **Configuration** — `defaultWorkers()` : `GOMAXPROCS` par défaut, surchargé par
   `TXTKIT_WORKERS`, lui-même surchargé par `-j`.
7. **Tests & distribution** — tests table-driven, `-race`, puis `Makefile` de
   cross-compilation.

---

## 4. Sous-commandes : les flags

Chaque sous-commande possède son propre `flag.NewFlagSet(..., flag.ContinueOnError)`,
avec `SetOutput(stderr)` et une `Usage` dédiée. `ContinueOnError` fait **renvoyer**
l'erreur de parsing (au lieu d'`os.Exit`), ce qui laisse `Run` choisir le code `2`.

```
count :  -j N       fichiers traités en parallèle (défaut : GOMAXPROCS)
         -total     ligne de total si plusieurs sources (défaut : true)

freq  :  -j N       idem
         -n N       nombre de mots affichés (0 = tous, défaut : 10)
         -min L     longueur minimale d'un mot (défaut : 1)
```

⚠️ **Piège** : `flag` ne lit **pas** les flags placés après les arguments positionnels.
`txtkit count file.txt -j 4` ignore `-j 4` (il devient un « fichier »). Mettre les flags
**avant** les fichiers : `txtkit count -j 4 file.txt`.

---

## 5. Tests

```bash
cd projets/1-cli
go test -race ./...
```

Trois familles, toutes en **table-driven** ([Ch. 13](../../chapitres/13-tests-outillage.md)) :

- **Dispatch** (`cli_test.go`) — codes de retour et messages d'usage de `Run`, via un
  helper `run(t, stdin, args...)` qui capture stdout/stderr/code.
- **Comptage** (`count_test.go`) — `countSource` sur des cas UTF-8 (« café 🚀 » = 7 runes
  mais 11 octets), absence de total pour une source unique, fichier manquant ⇒ code `1`.
- **Fréquence** (`freq_test.go`) — `normalize` (casse, ponctuation), `topWords` (tri
  stable à égalité), bout-en-bout via `run`.

> 🧪 **À tester soi-même** : ajouter une sous-commande `lines -n N` (les N premières
> lignes, façon `head`) et son test. Le squelette `flag.FlagSet` + `sourcesFrom` se
> réutilise tel quel.

---

## 6. Build, cross-compilation & distribution

```bash
make build          # bin/txtkit (version = git describe)
make dist           # dist/txtkit-<os>-<arch> pour 5 plateformes
```

La cross-compilation est native à Go : aucune toolchain externe, on fixe juste
`GOOS`/`GOARCH`. `CGO_ENABLED=0` garantit un **binaire statique** sans dépendance libc.

```bash
CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -o txtkit-linux-arm64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o txtkit.exe .
```

La **version** est injectée sans modifier le code source, via `-ldflags -X` :

```bash
go build -ldflags "-s -w -X example.com/txtkit/internal/cli.version=v1.0.0" .
#                        ^^   ^^ : -s -w retirent la table des symboles (binaire plus petit)
```

`txtkit version` affiche alors `txtkit v1.0.0`.

---

## 7. Points de vigilance

- **`os.Exit` court-circuite les `defer`** ([Ch. 16](../../chapitres/16-defer.md)). On le
  confine donc à `main`, jamais au cœur logique — d'où le retour d'un `int` par `Run`.
- **Ordre des résultats** : `mapBounded` écrit `out[i]` à un index **distinct** par
  worker — aucune course (`-race` propre, [Ch. 19](../../chapitres/19-goroutines.md)),
  et la sortie reste dans l'ordre des arguments malgré le parallélisme.
- **Déterminisme** : l'itération de map est **randomisée**
  ([Ch. 32](../../chapitres/32-maps-hachage.md)) ; `freq` doit donc trier explicitement,
  avec un critère secondaire alphabétique pour départager les ex æquo.
- **stdin lu une seule fois** : un `io.Reader` n'est pas rejouable. `txtkit count`
  sans fichier consomme stdin intégralement, ce qui est correct ici (une seule source).

---

## 8. Pour aller plus loin

- Ajouter un flag `-w/-l/-c` à la `wc` pour ne montrer que certaines colonnes.
- Détecter le binaire (octets nuls) et l'ignorer dans `freq`.
- Lire la configuration depuis un fichier (`~/.txtkit.toml`) **sous** les variables
  d'environnement dans l'ordre de priorité.
- Remplacer le sémaphore par
  [`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup) avec
  `SetLimit` pour propager la première erreur — voir
  [Ch. 23](../../chapitres/23-patterns-concurrence.md).
- Empaqueter les binaires `dist/` en archives `.tar.gz`/`.zip` + checksums pour une
  release GitHub.

---

## 📌 À retenir

- Un CLI testable = `Run(args, in, out, err) int` ; `main` ne fait qu'`os.Exit(Run(...))`.
- Une sous-commande = un `flag.FlagSet` dédié en `ContinueOnError`.
- Le parallélisme **borné** (sémaphore à canal + `WaitGroup`) accélère les I/O sans
  saturer la machine ; écrire à des index distincts évite toute course.
- Configuration en **couches** : défaut < environnement < flag.
- Cross-compiler du Go = fixer `GOOS`/`GOARCH` (+ `CGO_ENABLED=0` pour un statique).
