# 50 — Fichiers & systèmes de fichiers

> **Objectif** — Manipuler fichiers et répertoires « à la façon Go » : lire et écrire
> avec `os`, composer des chemins **portables** avec `path/filepath`, écrire de façon
> **atomique**, parcourir une arborescence avec `WalkDir`, consommer n'importe quelle
> source via l'abstraction `io/fs`, et **confiner** les accès avec `os.Root` (1.24)
> pour se prémunir de la traversée de chemin.
>
> **Prérequis** — [Ch. 41 — Entrées/sorties & flux](41-io-flux.md) (`io.Reader`/`Writer`),
> [Ch. 10 — Erreurs](10-erreurs.md) (`errors.Is`), [Ch. 16 — `defer`](16-defer.md).

---

## Introduction

Le chapitre 41 a montré comment **streamer** des octets à travers `io.Reader`/`io.Writer`.
Ici on descend d'un cran : **d'où** viennent ces octets sur un disque, **où** ils vont, et
comment nommer, créer, parcourir et protéger des fichiers sans se rendre dépendant d'un
système d'exploitation particulier.

Trois idées structurent le chapitre :

- **`os`** est la porte vers le système de fichiers concret (ouvrir, écrire, `Stat`, `Rename`).
- **`path/filepath`** compose des chemins **portables** — jamais de `"/"` ou `"\"` en dur.
- **`io/fs`** abstrait « un système de fichiers » en une interface : le même code lit un
  dossier réel, une archive embarquée (🔁 [Ch. 46](46-embed-build-deploiement.md)) ou un FS
  de test en mémoire.

L'exemple complet est dans [`code/ch50-fichiers-fs/`](../code/ch50-fichiers-fs/).

---

## Lire et écrire : les raccourcis et les ouvertures explicites

Pour un fichier **entier** qui tient en mémoire, deux fonctions suffisent :

```go
data, err := os.ReadFile("config.txt")          // lit tout, ferme tout seul
err = os.WriteFile("out.txt", data, 0o644)       // crée/tronque, écrit, ferme
```

`os.ReadFile`/`os.WriteFile` ouvrent, lisent/écrivent et **ferment** en un appel. Pratique,
mais `ReadFile` charge **tout** en RAM (⚠️ entrée non bornée → 🔁 [Ch. 41](41-io-flux.md),
`io.LimitReader`).

Dès qu'on veut **streamer** ou **contrôler le mode d'ouverture**, on ouvre explicitement :

| Appel                                   | Fait quoi                                                       |
| --------------------------------------- | --------------------------------------------------------------- |
| `os.Open(name)`                         | ouverture **lecture seule** (`O_RDONLY`)                        |
| `os.Create(name)`                       | crée ou **tronque**, écriture (`O_CREATE\|O_WRONLY\|O_TRUNC`)   |
| `os.OpenFile(name, flag, perm)`         | tout contrôler : flags + permissions                            |

```go
// Ajouter en fin de fichier (journal), en le créant au besoin :
f, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
if err != nil { /* ... */ }
defer f.Close() // ⚠️ un *os.File retient un descripteur tant qu'on ne ferme pas
```

Le troisième argument est un `os.FileMode` en **octal** (`0o644` = `rw-r--r--`, `0o755` =
`rwxr-xr-x`). Il n'est consulté qu'à la **création** ; sur un fichier existant, seul `os.Chmod`
change les permissions.

> 💡 **Toujours `defer f.Close()` juste après un `Open`/`Create` réussi.** C'est le même
> réflexe qu'au chapitre 41 : un descripteur non rendu au système finit par épuiser le quota
> (`too many open files`).

### `Stat` : interroger sans ouvrir

`os.Stat` renvoie un `os.FileInfo` (taille, mode, date de modification, `IsDir`) :

```go
info, err := os.Stat("data.txt")
if err == nil {
	fmt.Println(info.Size(), info.Mode(), info.IsDir())
}
```

### Répertoires : créer, supprimer, renommer

| Appel                    | Effet                                                     |
| ------------------------ | --------------------------------------------------------- |
| `os.Mkdir(dir, perm)`    | crée **un** répertoire (échoue si le parent manque)       |
| `os.MkdirAll(dir, perm)` | crée le chemin **complet** (comme `mkdir -p`)             |
| `os.Remove(name)`        | supprime un fichier **ou** un répertoire **vide**         |
| `os.RemoveAll(name)`     | supprime récursivement (comme `rm -rf`) ; nul → pas d'erreur |
| `os.Rename(old, new)`    | renomme/déplace ; **atomique** sur un même volume         |

## Écriture atomique : temporaire + `Rename`

Écrire directement sur le fichier cible (`os.WriteFile`) le **tronque d'abord** : si le
programme meurt en plein milieu, on laisse un fichier **corrompu**. Le patron sûr est
universel : écrire dans un temporaire du **même répertoire**, puis `Rename` sur la cible.
`Rename` est **atomique** — un lecteur voit soit l'ancien fichier, soit le nouveau, jamais un
état intermédiaire.

```go
// code/ch50-fichiers-fs/main.go
func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	// Le temporaire DOIT être dans le même dossier que la cible : Rename n'est
	// atomique qu'à l'intérieur d'un même système de fichiers.
	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName) // nettoyer le temporaire en cas d'échec
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil { // forcer sur le disque avant Rename
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path) // basculement instantané
}
```

> ⚠️ **Le temporaire doit être sur le même volume que la cible.** Un `Rename` entre deux
> systèmes de fichiers (ex. `/tmp` et `/home` sur des partitions distinctes) échoue avec
> `EXDEV` : il faudrait alors copier puis supprimer, ce qui **n'est plus atomique**. D'où
> `os.CreateTemp(dir, …)` dans le **dossier de destination**, pas dans `os.TempDir()`.

## Chemins portables : `path/filepath` vs `path`

Deux packages, une confusion classique :

| Package         | Séparateur                     | Pour…                                                     |
| --------------- | ------------------------------ | --------------------------------------------------------- |
| `path/filepath` | **celui de l'OS** (`\` / `/`)  | **chemins du disque** — le seul à utiliser pour `os.*`    |
| `path`          | **toujours `/`**               | chemins « slash » : URL, clés `io/fs`, `embed.FS`         |

Sur le disque, **toujours `filepath`** — il gère `\` sous Windows et `/` ailleurs :

```go
filepath.Join("etc", "app", "config.txt") // "etc/app/config.txt" (ou "etc\app\..." sous Windows)
filepath.Base("/var/log/app.log")         // "app.log"
filepath.Dir("/var/log/app.log")          // "/var/log"
filepath.Ext("app.log")                   // ".log"
filepath.Clean("a/b/../c")                // "a/c"
```

> ⚠️ **`filepath.Join` nettoie mais ne protège pas.** `Join` appelle `Clean`, donc
> `filepath.Join("safe", "../../etc/passwd")` produit `../etc/passwd` : le `..` **sort** du
> répertoire prévu. Nettoyer un chemin ne suffit **jamais** à confiner un accès — c'est le rôle
> d'`os.Root` (plus bas).

## Parcourir un répertoire

Pour **un** niveau, `os.ReadDir` renvoie des `fs.DirEntry` — plus légers qu'un `FileInfo` :
le type (fichier/répertoire) se lit **sans** appel `Stat` supplémentaire.

```go
// code/ch50-fichiers-fs/main.go
func listDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
```

Pour un parcours **récursif**, `filepath.WalkDir` (préférez-le à `filepath.Walk`) :

```go
filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err // TOUJOURS traiter l'erreur passée au callback
	}
	if !d.IsDir() && filepath.Ext(path) == ".go" {
		process(path)
	}
	return nil // renvoyer filepath.SkipDir pour élaguer un sous-arbre
})
```

## `io/fs` : le système de fichiers comme interface

Plutôt que d'écrire une fonction qui parle directement au disque, écrivez-la contre l'interface
`fs.FS`. Elle devient testable **sans disque** et acceptera aussi bien un dossier réel
(`os.DirFS`) qu'une archive embarquée (`embed.FS`, 🔁 [Ch. 46](46-embed-build-deploiement.md)) :

```go
// code/ch50-fichiers-fs/main.go
func countGoFiles(fsys fs.FS) (int, error) {
	count := 0
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".go" {
			count++
		}
		return nil
	})
	return count, err
}
```

Le package `fs` fournit les mêmes helpers que `os`, mais sur une `fs.FS` :
`fs.ReadFile`, `fs.ReadDir`, `fs.WalkDir`, `fs.Glob`, `fs.Sub` (vue sur un sous-dossier). Les
clés utilisent **toujours `/`** (package `path`), jamais le séparateur de l'OS.

> 💡 **`os.DirFS(dir)`** expose un dossier réel en `fs.FS` — le pont entre le disque et l'API
> abstraite. Dans l'exemple, `countGoFiles(os.DirFS(dir))` parcourt un vrai répertoire.

### Tester sans toucher au disque : `fstest.MapFS`

`testing/fstest.MapFS` est un `fs.FS` **en mémoire** : idéal pour tester la logique de parcours
sans créer de fichiers temporaires.

```go
// code/ch50-fichiers-fs/fs_test.go
fsys := fstest.MapFS{
	"main.go":            {Data: []byte("package main")},
	"pkg/a.go":           {Data: []byte("package pkg")},
	"pkg/data.txt":       {Data: []byte("non-go")},
	"internal/util/u.go": {Data: []byte("package util")},
}
got, err := countGoFiles(fsys) // aucune I/O disque
```

## 🆕 Go 1.24 — `os.Root` : confiner les accès

Quand un nom de fichier vient d'une **entrée utilisateur** (paramètre HTTP, nom d'archive), le
nettoyer ne suffit pas : un lien symbolique ou un `..` bien placé peut faire fuir la lecture
vers `/etc/passwd`. `os.OpenRoot(dir)` ouvre un **`*os.Root`** dont **toutes** les opérations
sont confinées à `dir` — impossible d'en sortir, y compris via un symlink.

```go
// code/ch50-fichiers-fs/main.go
func safeReadUnder(root *os.Root, name string) ([]byte, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f) // un *os.File confiné reste un io.Reader (🔁 ch41)
}
```

```go
root, _ := os.OpenRoot(baseDir)
defer root.Close()
safeReadUnder(root, "config.txt")        // OK
safeReadUnder(root, "../../etc/passwd")  // erreur : la traversée est refusée
```

`*os.Root` propose `Open`, `Create`, `OpenFile`, `Mkdir`, `Stat`, `Remove`, `Rename`… tous
relatifs à la racine. C'est **la** réponse standard à la traversée de chemin, à préférer à toute
validation manuelle de `..`.

> 💡 **`os.CopyFS(dir, fsys)`** (1.23) écrit tout un `fs.FS` sur le disque sous `dir` — pratique
> pour matérialiser une arborescence embarquée à l'installation.

## Détecter les erreurs : les sentinelles de `fs`

Testez la **cause**, pas le message. Les sentinelles vivent dans `io/fs` et se comparent avec
`errors.Is` (🔁 [Ch. 10](10-erreurs.md)) :

```go
// code/ch50-fichiers-fs/fs_test.go
_, err := os.Open(filepath.Join(t.TempDir(), "absent"))
if !errors.Is(err, fs.ErrNotExist) { /* ... */ }
```

| Sentinelle        | Signification                    |
| ----------------- | -------------------------------- |
| `fs.ErrNotExist`  | le fichier n'existe pas          |
| `fs.ErrExist`     | le fichier existe déjà           |
| `fs.ErrPermission`| droits insuffisants              |

> 💡 Préférez `errors.Is(err, fs.ErrNotExist)` aux anciennes `os.IsNotExist(err)` : `errors.Is`
> traverse les erreurs **enveloppées** (`%w`), pas les helpers historiques.

## ⚠️ Pièges

- **Oublier `Close()`** sur un `*os.File` : fuite de descripteurs (🔁 [Ch. 41](41-io-flux.md),
  [Ch. 16](16-defer.md)). Réflexe `defer f.Close()` après tout `Open`/`Create` réussi.
- **TOCTOU** (*time-of-check to time-of-use*) : tester l'existence avec `Stat` **puis** ouvrir
  laisse une fenêtre où le fichier change. Ouvrez directement et gérez l'erreur — une seule
  opération atomique.
- **Traversée de chemin** sur une entrée utilisateur : `filepath.Join`/`Clean` ne confinent
  **pas**. Utilisez `os.Root`.
- **Ignorer l'erreur du callback `WalkDir`** : si `err != nil` y est passé (dossier illisible),
  le déréférencer sur `d` panique. Testez `err` en **première** ligne.
- **Mélanger `path` et `filepath`** : `path` pour les clés `fs.FS`/URL (toujours `/`),
  `filepath` pour le disque. Les inverser casse sous Windows.
- **`os.WriteFile` n'est pas atomique** : il tronque avant d'écrire. Pour une config ou un état
  critique, passez par `writeFileAtomic`.

## ⚡ Performance

- **`WalkDir` plutôt que `Walk`** : `WalkDir` s'appuie sur `ReadDir` et fournit un `fs.DirEntry`
  — il **évite un `Lstat` par fichier**, gain net sur les grandes arborescences.
- **Streamer plutôt que `ReadFile`** : pour un gros fichier, `os.Open` + `io.Copy`/`bufio` évite
  de charger des centaines de Mio en RAM (🔁 [Ch. 41](41-io-flux.md)).
- **Tamponner les écritures nombreuses** : enrober le `*os.File` dans un `bufio.Writer` amortit
  les syscalls (et ne pas oublier `Flush`).
- **`os.ReadDir` renvoie déjà des `DirEntry` triés par nom** : inutile de re-`Stat` chaque entrée
  pour connaître son type.

## 🧪 À tester soi-même

Dans [`code/ch50-fichiers-fs/`](../code/ch50-fichiers-fs/) :

```bash
cd code && go test ./ch50-fichiers-fs/
```

Ajoutez un test qui tente `safeReadUnder(root, "../secret")` et vérifie que l'accès est
**refusé** ; comparez avec un `os.ReadFile` naïf sur le même chemin construit par
`filepath.Join`, qui, lui, sortirait du répertoire.

---

## 📌 À retenir

- **`os`** : `ReadFile`/`WriteFile` pour le tout-en-un, `Open`/`Create`/`OpenFile` (+ flags) pour
  streamer ; `defer Close()` systématique ; `MkdirAll`, `RemoveAll`, `Rename`.
- **Écriture atomique** = temporaire dans le **même dossier** + `Sync` + `Rename`. Jamais écrire
  directement sur un fichier de config critique.
- **`filepath`** pour le disque (séparateur OS), **`path`** pour les clés `fs.FS`/URL (`/`).
  `Clean`/`Join` **n'empêchent pas** la traversée.
- **`io/fs`** abstrait le système de fichiers : écrivez contre `fs.FS`, testez avec
  `fstest.MapFS`, réutilisez avec `os.DirFS` et `embed.FS`. `WalkDir` > `Walk`.
- 🆕 **`os.Root`** (1.24) confine tous les accès à un répertoire : la parade standard à la
  traversée de chemin.
- Détectez les erreurs avec `errors.Is(err, fs.ErrNotExist/ErrExist/ErrPermission)`.

## 🔁 Pour aller plus loin

- [Ch. 41 — Entrées/sorties & flux](41-io-flux.md) : `io.Reader`/`Writer`, streaming, `bufio`.
- [Ch. 46 — Embarquer & déployer](46-embed-build-deploiement.md) : `embed.FS` **est** un `fs.FS`.
- [Ch. 10 — Gestion des erreurs](10-erreurs.md) : `errors.Is` et les sentinelles.
- [Ch. 47 — Sécurité & chaîne d'approvisionnement](47-securite-supply-chain.md) : la traversée
  de chemin comme classe de vulnérabilité.
- Référence : [`pkg.go.dev/os`](https://pkg.go.dev/os), [`pkg.go.dev/io/fs`](https://pkg.go.dev/io/fs),
  [`pkg.go.dev/path/filepath`](https://pkg.go.dev/path/filepath).
