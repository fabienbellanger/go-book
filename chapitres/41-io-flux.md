# 41 — Entrées/sorties & flux

> **Objectif** — Maîtriser le **modèle de flux** de Go : les interfaces `io.Reader`/`io.Writer`,
> leur composition, le tamponnage avec `bufio`, et les tampons en mémoire de `bytes`. Savoir
> **streamer** plutôt que tout charger en mémoire.
>
> **Prérequis** — [Ch. 9 — Interfaces](09-interfaces.md), [Ch. 7 — Maps & strings](07-maps-strings.md)

---

## Introduction

En Go, presque tout ce qui produit ou consomme des octets — un fichier, une connexion réseau,
une réponse HTTP, un champ de formulaire, un buffer en mémoire, `os.Stdin` — parle **deux
interfaces minuscules** : `io.Reader` et `io.Writer`. Les apprendre une fois, c'est savoir
brancher n'importe quelle source sur n'importe quelle destination.

L'autre idée centrale est le **streaming** : on traite les données **par blocs**, sans jamais
charger un fichier de 2 Gio en RAM. C'est ce qui rend les programmes Go économes et scalables.

L'exemple complet est dans [`code/ch41-io/`](../code/ch41-io/).

---

## Les deux interfaces qui gouvernent tout

```go
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
```

- **`Read(p)`** remplit `p` avec **au plus** `len(p)` octets, renvoie le nombre lu `n` et une
  erreur. À la fin du flux : `io.EOF`.
- **`Write(p)`** écrit `p` ; le contrat impose `n == len(p)` dès que `err == nil`.

Leur force vient de leur petitesse : **une seule méthode** suffit à fabriquer un nouveau maillon
qui s'insère dans tout l'écosystème.

> 💡 **Pourquoi une seule méthode ?** En Go, la satisfaction d'une interface est **implicite**
> ([Ch. 9](09-interfaces.md)) : un type n'a rien à déclarer pour devenir un `io.Reader`, il lui
> suffit d'avoir une méthode `Read` de la bonne signature. Plus un contrat a de méthodes, plus il
> est dur à satisfaire — et plus il exclut de types qui auraient pu s'y prêter. En réduisant
> `Reader`/`Writer` à un seul verbe (lire, ou écrire), Go maximise le nombre de types compatibles :
> fichiers, sockets, buffers mémoire, compresseurs (`gzip`), chiffreurs (`crypto/cipher`),
> `os.Stdin`… tous deviennent interchangeables derrière la même interface, sans rien connaître les
> uns des autres.

Voici un `Writer` qui met en majuscules à la volée :

```go
type upperWriter struct{ dst io.Writer }

func (w upperWriter) Write(p []byte) (int, error) {
	if _, err := w.dst.Write(bytes.ToUpper(p)); err != nil {
		return 0, err
	}
	return len(p), nil // contrat : n == len(p) quand err == nil
}
```

> 💡 **« Accepter des interfaces, renvoyer des structs »** ([Ch. 9](09-interfaces.md)). Une fonction
> qui prend un `io.Reader` accepte aussi bien un fichier qu'une chaîne (`strings.NewReader`),
> un `bytes.Buffer`, une socket… et devient **triviale à tester**.

### Composition : interfaces enrichies et décorateurs

La bibliothèque standard compose des contrats plus riches en **embarquant** `Reader`/`Writer` dans
d'autres interfaces — jamais en ajoutant de méthodes à `Reader`/`Writer` eux-mêmes :

| Interface            | Définition                     | Exemple de type qui l'implémente       |
| -------------------- | ------------------------------ | -------------------------------------- |
| `io.Closer`          | `Close() error`                | `*os.File`, `net.Conn`, `*gzip.Reader` |
| `io.ReadCloser`      | `Reader` + `Closer`            | `http.Response.Body`                   |
| `io.WriteCloser`     | `Writer` + `Closer`            | `*gzip.Writer`, `net.Conn`             |
| `io.ReadWriter`      | `Reader` + `Writer`            | `net.Conn`, `*os.File`                 |
| `io.ReadWriteCloser` | `Reader` + `Writer` + `Closer` | `net.Conn`, `*os.File`                 |

`io.Closer` est la pièce qui manquait aux exemples précédents : un `bytes.Buffer` n'a rien à
fermer (pas de ressource OS), mais un `*os.File` ou un `net.Conn` **retiennent** un descripteur
tant qu'on n'a pas appelé `Close` (voir ⚠️ Pièges).

C'est aussi ce qui rend possible la **composition par décoration** : envelopper un `Reader` dans
un autre qui ajoute un comportement — déchiffrement, décompression, tamponnage — sans connaître
ni modifier le type d'origine :

```
  os.Open(path)         gzip.NewReader(f)         bufio.NewReader(gz)
  *os.File         -->  *gzip.Reader         -->  *bufio.Reader
  (io.Reader)            (décompresse f)           (tamponne gz)
       |                       |                          |
       v                       v                          v
  octets bruts du disque  octets décompressés       octets servis par blocs

  chaque étage ne connaît QUE l'interface io.Reader, jamais le type concret précédent
```

```go
f, _ := os.Open("archive.log.gz")
defer f.Close()
gz, _ := gzip.NewReader(f) // *gzip.Reader est lui-même un io.Reader
defer gz.Close()
br := bufio.NewReader(gz)  // un troisième Reader, qui enveloppe le deuxième
line, _ := br.ReadString('\n')
```

Chaque maillon **ignore tout** des maillons précédents : c'est le patron décorateur, rendu naturel
par une interface à une seule méthode — n'importe quel `Reader` peut en envelopper un autre.

### Le streaming en image

```
  source                                    destination
  (io.Reader)        io.Copy                 (io.Writer)
  +--------+      +-------------+            +----------+
  | fichier| ---> | buffer 32Ko | ---------> | réseau   |
  | 2 Gio  | <--- | (réutilisé) | <--------- | / disque |
  +--------+      +-------------+            +----------+
       ^   on lit un bloc, on l'écrit, on recommence — jamais 2 Gio en RAM
```

## `io.Copy` & la boîte à outils `io`

`io.Copy(dst, src)` est le couteau suisse : il lit `src` bloc par bloc et écrit dans `dst`
jusqu'à `io.EOF`. Pas besoin de boucle manuelle.

| Approche                                                 | Ce qu'il faut gérer soi-même                                                    |
| -------------------------------------------------------- | ------------------------------------------------------------------------------- |
| boucle manuelle (`for { n, err := src.Read(buf); ... }`) | `Read` partiel, ordre `n`/`err` (⚠️ ci-dessous), relance jusqu'à `io.EOF`       |
| `io.Copy(dst, src)`                                      | rien — gère ces cas en interne, et peut être **zéro-copie** (⚡ Performance)    |
| `io.ReadAll(r)`                                          | rien, mais charge **tout** en mémoire avant de pouvoir traiter quoi que ce soit |

```go
func copyThrough(src io.Reader) (string, error) {
	var sink bytes.Buffer
	if _, err := io.Copy(upperWriter{dst: &sink}, src); err != nil {
		return "", err
	}
	return sink.String(), nil
}
```

Les helpers du package `io`, à connaître :

| Fonction              | Rôle                                                                                                            |
| --------------------- | --------------------------------------------------------------------------------------------------------------- |
| `io.Copy(dst, src)`   | recopie tout un flux, par blocs                                                                                 |
| `io.CopyBuffer`       | comme `Copy` mais avec un buffer fourni (réutilisable)                                                          |
| `io.ReadAll(r)`       | lit tout en mémoire (⚠️ taille non bornée)                                                                      |
| `io.ReadFull(r, buf)` | lit exactement `len(buf)` octets, ou erreur (`io.EOF` si rien lu, `io.ErrUnexpectedEOF` si lecture interrompue) |
| `io.WriteString(w,s)` | écrit une `string` sans conversion `[]byte` superflue                                                           |
| `io.MultiReader(...)` | concatène plusieurs `Reader` en un seul flux                                                                    |
| `io.MultiWriter(...)` | écrit simultanément vers plusieurs `Writer` (ex. log + écran)                                                   |
| `io.TeeReader(r, w)`  | renvoie un `Reader` qui **copie dans `w`** tout ce qu'on y lit                                                  |
| `io.LimitReader(r,n)` | borne la lecture à `n` octets (anti-DoS sur un upload)                                                          |
| `io.SectionReader`    | vue sur une **tranche** d'un `io.ReaderAt` (offset + longueur)                                                  |
| `io.Discard`          | un `Writer` puits : jette tout (utile pour mesurer/drainer)                                                     |

`io.TeeReader` permet de lire **une seule fois** tout en gardant une copie :

```go
func teeAndCount(src io.Reader) (string, int64, error) {
	var mirror bytes.Buffer
	tee := io.TeeReader(src, &mirror) // ce qu'on lit dans tee est copié dans mirror
	n, err := io.Copy(io.Discard, tee)
	return mirror.String(), n, err
}
```

### `io.Pipe` : producteur ↔ consommateur en mémoire

`io.Pipe()` renvoie un couple `(*PipeReader, *PipeWriter)` connecté : ce qu'on écrit dans l'un, on
le lit dans l'autre. **Synchrone** (l'écriture bloque tant qu'on ne lit pas), sans fichier ni
buffer géant — idéal pour brancher une API « qui veut un `Reader` » sur une API « qui écrit dans
un `Writer` ».

```go
func pipeProducerConsumer(chunks []string) (string, error) {
	pr, pw := io.Pipe()
	var wg sync.WaitGroup
	wg.Go(func() { // producteur dans sa goroutine
		defer pw.Close() // ⚠️ fermer le Writer signale l'EOF au lecteur
		for _, c := range chunks {
			io.WriteString(pw, c)
		}
	})
	out, err := io.ReadAll(pr) // lit jusqu'à la fermeture
	wg.Wait()
	return string(out), err
}
```

> ⚠️ **Toujours fermer le `PipeWriter`** (ici via `defer pw.Close()`), sinon le lecteur attend un
> EOF qui ne vient jamais → **fuite de goroutine** ([Ch. 23](23-patterns-concurrence.md)).
> `WaitGroup.Go` est l'idiome 1.25 ([Ch. 21](21-synchronisation.md)).

## `bufio` : amortir les appels système

Un `Read`/`Write` non tamponné = **un appel système par opération**. Lire un fichier octet par
octet ferait des millions de syscalls. `bufio` interpose un **tampon** : on remplit/vide la RAM,
on ne touche le noyau que lorsque le tampon est plein/vide.

```
   sans bufio :  Write('a') Write('b') Write('c')   -> 3 syscalls
   avec bufio :  [a][b][c] ... Flush()              -> 1 syscall
```

### `bufio.Scanner` : découper un flux

`Scanner` lit un flux **token par token** : par défaut une **ligne** à la fois, sans charger tout
le fichier.

```go
sc := bufio.NewScanner(r)
for sc.Scan() {        // avance d'un token ; false à EOF ou à la 1re erreur
	process(sc.Text()) // ou sc.Bytes() pour éviter une allocation
}
if err := sc.Err(); err != nil { /* erreur réelle (EOF n'en est pas une) */ }
```

On change la stratégie de découpage avec `Split` :

| `Split` fourni    | Découpe par…               |
| ----------------- | -------------------------- |
| `bufio.ScanLines` | lignes (défaut)            |
| `bufio.ScanWords` | mots (séparés par espaces) |
| `bufio.ScanRunes` | runes (caractères Unicode) |
| `bufio.ScanBytes` | octets                     |

D'autres façons de découper un flux, à choisir selon le besoin :

| Outil                             | Délimiteur                    | Quand l'utiliser                                   |
| --------------------------------- | ----------------------------- | -------------------------------------------------- |
| `bufio.Reader.ReadString('\n')`   | **conservé** dans le résultat | besoin du séparateur, pas de limite de taille fixe |
| `bufio.Scanner` (+ `Split`)       | **retiré** du résultat        | cas courant ; ⚠️ buffer plafonné (ci-dessous)      |
| `strings.Lines`/`SplitSeq` (1.24) | selon la fonction             | une `string` déjà en mémoire, pas un flux à lire   |

> ⚠️ **Lignes trop longues.** Le `Scanner` a un buffer **plafonné** (64 Kio par défaut). Une ligne
> plus longue provoque **`bufio.ErrTooLong`** (renvoyée par `sc.Err()`, pas par `Scan`). Pour de
> longues lignes : `sc.Buffer(make([]byte, 0, 64*1024), 1<<20)` augmente le plafond, ou passez à
> `bufio.Reader.ReadString('\n')`.

## `bytes` : un tampon qui lit ET écrit

`bytes.Buffer` est à la fois un `io.Reader` **et** un `io.Writer` : on y écrit, puis on le lit —
parfait comme destination de `io.Copy` ou pour assembler un message. `bytes.NewReader` expose un
`[]byte` existant comme `Reader` (sans copie).

```go
var buf bytes.Buffer
fmt.Fprintf(&buf, "x=%d", 42) // Buffer est un Writer
io.Copy(dst, &buf)            // ... et un Reader
```

Le package `bytes` reflète `strings`, mais pour des `[]byte` : `bytes.Contains`, `bytes.Split`,
`bytes.Fields`, `bytes.ToUpper`, `bytes.Equal`… Règle simple : **`[]byte` muable → `bytes` ;
`string` immuable → `strings`** ([Ch. 31](31-strings-profondeur.md)).

## 🆕 Go 1.24 — itérateurs `strings.Lines`/`SplitSeq`/`FieldsSeq`

Pour une **`string`** déjà en mémoire, les itérateurs de `strings` ([Ch. 18](18-iterateurs.md))
remplacent avantageusement un `Scanner` : ni allocation de slice, ni objet `Scanner`.

```go
for line := range strings.Lines(text) { // les lignes incluent leur '\n'
	use(strings.TrimRight(line, "\n"))
}
for word := range strings.FieldsSeq(text) { use(word) }
```

- `strings.Lines` — itère les lignes (terminateur `\n` **inclus**).
- `strings.SplitSeq(s, sep)` — comme `Split` mais sans construire le slice.
- `strings.FieldsSeq` — comme `Fields` (découpe sur les espaces) sans slice.

> 💡 **Scanner vs itérateur.** `Scanner` lit un **flux** (`io.Reader` : fichier, socket). Les
> itérateurs `strings.*` travaillent sur une **`string` en mémoire**. Choisissez selon la source.

## ⚠️ Pièges

- **Oublier `Close()`** sur un `*os.File`, une connexion réseau ou tout type `io.Closer` : la
  ressource sous-jacente (descripteur de fichier, socket) n'est jamais rendue au système
  d'exploitation. Réflexe : `defer f.Close()` juste après un `Open`/`Dial` réussi — y compris pour
  `resp.Body` en HTTP (🔁 [Ch. 45](45-net-http.md)), où il faut en plus **drainer** le corps avant
  de le fermer pour permettre la réutilisation de la connexion.
- **Oublier `Flush()`** sur un `bufio.Writer` : les derniers octets restent dans le tampon et
  n'atteignent jamais la destination. Réflexe : `defer bw.Flush()` (et vérifier son erreur).
- **`Read` partiel** : `Read` peut renvoyer `0 < n < len(p)` **sans** erreur — il ne « remplit »
  pas forcément `p`. Ne bouclez jamais à la main : utilisez `io.Copy`, `io.ReadFull` ou un `Scanner`.
- **`io.EOF` n'est pas une erreur** anormale : c'est la fin **normale** du flux. `io.Copy`/`ReadAll`
  le gèrent et ne le remontent pas.
- **`io.EOF` peut accompagner des données valides, dans le même appel.** Le contrat de `Read`
  autorise `n > 0` **et** `err == io.EOF` renvoyés ensemble — ou l'erreur seule, au tour suivant ;
  les deux comportements sont légaux. Du code qui teste `err` **avant** de traiter `p[:n]` perd
  silencieusement le dernier bloc :

  ```go
  n, err := r.Read(buf)
  if n > 0 {
      process(buf[:n]) // traiter D'ABORD les octets reçus, même si err != nil
  }
  if err != nil {
      return err // ... puis seulement vérifier l'erreur
  }
  ```

  `io.Copy`, `io.ReadAll` et `bufio.Scanner` respectent déjà cet ordre — une raison de plus de
  leur préférer une boucle manuelle.

- **`Scanner.Bytes()` est réutilisé** : le slice renvoyé pointe vers le buffer interne et change au
  prochain `Scan`. Pour le conserver, **copiez-le** (`append([]byte(nil), sc.Bytes()...)`) ou
  utilisez `sc.Text()` (qui alloue une `string`).
- **`io.ReadAll` non borné** : sur une entrée hostile, c'est un risque mémoire. Encadrez avec
  `io.LimitReader`.

## ⚡ Performance

- **Réutiliser le buffer.** `io.CopyBuffer(dst, src, buf)` évite de réallouer 32 Kio à chaque
  appel ; combinez avec un `sync.Pool` ([Ch. 21](21-synchronisation.md)) pour les chemins chauds.
- **Taille du tampon `bufio`.** `bufio.NewReader`/`NewWriter` réservent **4 Kio** par défaut.
  `bufio.NewReaderSize`/`NewWriterSize` permettent de l'ajuster à la charge réelle : plus grand
  pour amortir des lectures réseau par paquets, plus petit pour limiter l'empreinte mémoire quand
  des milliers de connexions tournent en parallèle.
- **`io.Copy` peut être zéro-copie.** S'il détecte que `src` implémente `WriterTo` ou `dst`
  implémente `ReaderFrom`, il délègue et **n'alloue aucun buffer** intermédiaire (c'est le cas
  entre fichiers, sockets, `bytes.Buffer`).
- **`io.WriteString`** évite la conversion `string`→`[]byte` quand le `Writer` sait écrire une
  chaîne directement (`io.StringWriter`).
- **Pré-dimensionner** un `bytes.Buffer` (`buf.Grow(n)`) supprime les réallocations successives
  ([Ch. 26](26-allocation-escape.md), [Ch. 30](30-slices-profondeur.md)).

## 🧪 À tester soi-même

Dans [`code/ch41-io/`](../code/ch41-io/) :

```bash
cd code && go test -race ./ch41-io/
```

Ajoutez un test qui mesure le nombre d'appels `Write` reçus par un `Writer` compteur, avec et
sans `bufio.Writer` interposé — vous verrez l'amortissement.

---

## 📌 À retenir

- **`io.Reader`/`io.Writer`** sont les deux interfaces pivots : une méthode chacune, composables à
  l'infini — en chaîne par décoration (`gzip` sur un fichier, `bufio` sur le tout) et en contrats
  enrichis par embedding (`io.Closer`, `io.ReadWriteCloser`). Acceptez-les en paramètre pour un
  code générique et testable.
- **`io.Copy`** streame par blocs ; les helpers (`Tee`, `Multi`, `Limit`, `Pipe`, `Discard`)
  couvrent l'essentiel des montages de flux.
- **`bufio`** amortit les syscalls ; `Scanner` découpe un flux (⚠️ lignes trop longues, `Bytes()`
  réutilisé) — pensez à `Flush` côté écriture.
- **`bytes.Buffer`** est Reader **et** Writer ; `bytes` ↔ `strings` selon muable/immuable.
- 🆕 Pour une `string` en mémoire, `strings.Lines`/`SplitSeq`/`FieldsSeq` (1.24) remplacent le
  `Scanner` sans allouer.

## 🔁 Pour aller plus loin

- [Ch. 31 — Strings en profondeur](31-strings-profondeur.md) : conversions `string`↔`[]byte`.
- [Ch. 26 — Allocation & escape analysis](26-allocation-escape.md) : réutilisation de buffers.
- [Ch. 18 — Itérateurs](18-iterateurs.md) : les `iter.Seq` derrière `strings.Lines`.
- [Ch. 45 — `net/http`](45-net-http.md) : `Request.Body`/`ResponseWriter` sont des flux `io`.
- Référence : [`pkg.go.dev/io`](https://pkg.go.dev/io), [`pkg.go.dev/bufio`](https://pkg.go.dev/bufio).
