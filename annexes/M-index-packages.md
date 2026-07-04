# Annexe M — Index des packages de la bibliothèque standard

> **Objectif** — Répondre d'un coup d'œil à « **où le livre parle-t-il de tel
> package ?** ». Cet index ne recense que les chapitres/annexes où le package est
> **réellement traité** (usage substantiel, pas une simple mention). Chaque
> référence est cliquable. La colonne « Rôle » résume la fonction du package en une
> ligne — pour les détails, suivez le lien.
>
> **Note de lecture** — Un package peut apparaître dans plusieurs chapitres : on
> cite d'abord son **chapitre de référence** (là où il est enseigné), puis les
> chapitres qui le réinvestissent en profondeur.

---

## Index rapide par domaine

- **E/S, fichiers & flux** — `io`, `bufio`, `bytes`, `os`, `path`, `path/filepath`, `io/fs`, `embed`
- **Texte** — `strings`, `strconv`, `unicode/utf8`, `fmt`, `regexp`, `text/template`, `html/template`
- **Collections & génériques** — `slices`, `maps`, `cmp`, `sort`, `iter`, `container/*`
- **Concurrence** — `sync`, `sync/atomic`, `context`
- **Réseau & web** — `net`, `net/http`, `crypto/tls`
- **Encodage & sérialisation** — `encoding/json`, `encoding/xml`, `encoding/csv`, `encoding/gob`
- **Base de données** — `database/sql`, `database/sql/driver`
- **Cryptographie & aléa** — `crypto/*`, `hash`, `math/rand/v2`
- **Erreurs & journalisation** — `errors`, `log`, `log/slog`
- **Temps** — `time`
- **Système & CLI** — `os/exec`, `os/signal`, `flag`
- **Runtime & bas niveau** — `runtime`, `runtime/pprof`, `runtime/trace`, `reflect`, `unsafe`
- **Tests & outillage** — `testing`, `testing/fstest`, `testing/synctest`

---

## Index alphabétique

| Package | Traité dans | Rôle |
| ------- | ----------- | ---- |
| `bufio` | [Ch. 41](../chapitres/41-io-flux.md) | Tamponnage des E/S (`Reader`/`Writer`/`Scanner`), amortit les appels système |
| `bytes` | [Ch. 41](../chapitres/41-io-flux.md), [Ch. 31](../chapitres/31-strings-profondeur.md) | Manipulation de `[]byte` ; `bytes.Buffer` est `Reader` **et** `Writer` |
| `cmp` | [Ch. 11](../chapitres/11-genericite.md) | Comparaison générique ordonnée (`cmp.Compare`, `cmp.Or`, contrainte `cmp.Ordered`) |
| `container/heap`, `container/list`, `container/ring` | [Annexe D](D-algorithmes.md) | Tas binaire, liste chaînée et anneau — structures de données classiques |
| `context` | [Ch. 22](../chapitres/22-context.md), [Ch. 20](../chapitres/20-channels-select.md), [Ch. 23](../chapitres/23-patterns-concurrence.md) | Annulation, deadlines et valeurs de requête propagées dans l'arbre d'appels |
| `crypto/aes`, `crypto/cipher` | [Ch. 53](../chapitres/53-crypto.md) | Chiffrement symétrique authentifié (AES-GCM, AEAD) |
| `crypto/hmac` | [Ch. 53](../chapitres/53-crypto.md) | Authentification de message par clé partagée (HMAC) ; `hmac.Equal` en temps constant |
| `crypto/rand` | [Ch. 53](../chapitres/53-crypto.md), [Ch. 47](../chapitres/47-securite-supply-chain.md) | Générateur d'aléa cryptographique (CSPRNG) : clés, nonces, jetons |
| `crypto/sha256`, `crypto/sha512` | [Ch. 53](../chapitres/53-crypto.md), [Ch. 41](../chapitres/41-io-flux.md) | Hachage cryptographique ; `hash.Hash` est un `io.Writer` (hachage en flux) |
| `crypto/subtle` | [Ch. 53](../chapitres/53-crypto.md) | Comparaisons en temps constant (`ConstantTimeCompare`), anti-timing |
| `crypto/tls` | [Ch. 53](../chapitres/53-crypto.md), [Ch. 52](../chapitres/52-reseau-net.md) | TLS client/serveur, `tls.Config`, mTLS |
| `database/sql` | [Ch. 51](../chapitres/51-database-sql.md) | API abstraite d'accès aux bases SQL : pool, requêtes, transactions, `context` |
| `database/sql/driver` | [Ch. 51](../chapitres/51-database-sql.md) | Interface qu'un driver implémente sous `database/sql` (`Conn`, `Stmt`, `Rows`, `Tx`) |
| `embed` | [Ch. 46](../chapitres/46-embed-build-deploiement.md), [Ch. 49](../chapitres/49-templates.md), [Ch. 50](../chapitres/50-fichiers-fs.md) | Embarquer des fichiers dans le binaire via `//go:embed` ; `embed.FS` est un `fs.FS` |
| `encoding/csv` | [Ch. 42](../chapitres/42-encodages-serialisation.md) | Lecture/écriture de fichiers CSV |
| `encoding/gob` | [Ch. 42](../chapitres/42-encodages-serialisation.md) | Sérialisation binaire propre à Go |
| `encoding/json` | [Ch. 42](../chapitres/42-encodages-serialisation.md), [Ch. 08](../chapitres/08-structs-methodes.md), [Ch. 34](../chapitres/34-reflexion.md) | (Dé)sérialisation JSON, balises de struct, `Marshal`/`Unmarshal` |
| `encoding/xml` | [Ch. 42](../chapitres/42-encodages-serialisation.md) | (Dé)sérialisation XML |
| `errors` | [Ch. 10](../chapitres/10-erreurs.md), [Ch. 17](../chapitres/17-panic-recover.md) | Erreurs comme valeurs : `errors.Is`/`As`/`Join`, enveloppement (`%w`) |
| `flag` | [Ch. 48](../chapitres/48-processus-signaux-cli.md), [Ch. 13](../chapitres/13-tests-outillage.md) | Analyse des arguments de ligne de commande |
| `fmt` | [Annexe I](I-formatage-fmt.md), [Ch. 03](../chapitres/03-variables-constantes-types.md) | Formatage et affichage ; verbes, `Stringer`, `Errorf` (`%w`) |
| `hash` | [Ch. 53](../chapitres/53-crypto.md) | Interface commune des fonctions de hachage (implémente `io.Writer`) |
| `html/template` | [Ch. 49](../chapitres/49-templates.md) | Gabarits HTML avec **échappement contextuel** (anti-XSS) |
| `io` | [Ch. 41](../chapitres/41-io-flux.md) | Les interfaces pivots `Reader`/`Writer` et la boîte à outils de flux (`Copy`, `Tee`…) |
| `io/fs` | [Ch. 50](../chapitres/50-fichiers-fs.md), [Ch. 46](../chapitres/46-embed-build-deploiement.md) | Système de fichiers abstrait en lecture (`fs.FS`, `fs.WalkDir`, `fs.Sub`) |
| `iter` | [Ch. 18](../chapitres/18-iterateurs.md), [Ch. 11](../chapitres/11-genericite.md) | Itérateurs par fonction (`iter.Seq`/`Seq2`) pour `range`-over-func |
| `log` | [Ch. 43](../chapitres/43-journalisation-slog.md) | Journalisation classique (texte) ; à comparer à `log/slog` |
| `log/slog` | [Ch. 43](../chapitres/43-journalisation-slog.md), [Ch. 45](../chapitres/45-net-http.md) | Journalisation **structurée** (handlers, attributs, niveaux) |
| `maps` | [Ch. 07](../chapitres/07-maps-strings.md), [Ch. 32](../chapitres/32-maps-hachage.md) | Fonctions génériques sur les maps (`Keys`, `Values`, `Clone`, `Equal`) |
| `math/rand/v2` | [Ch. 53](../chapitres/53-crypto.md), [Annexe C](C-nouveautes-1.21-1.26.md) | Aléa **non** cryptographique (rapide, reproductible) — jamais pour un secret |
| `net` | [Ch. 52](../chapitres/52-reseau-net.md) | Réseau bas niveau : TCP/UDP, `Dial`/`Listen`, `net.Conn`, deadlines, DNS |
| `net/http` | [Ch. 45](../chapitres/45-net-http.md) | Serveurs et clients HTTP, `Handler`, `ServeMux`, `http.Client` |
| `os` | [Ch. 50](../chapitres/50-fichiers-fs.md), [Ch. 48](../chapitres/48-processus-signaux-cli.md), [Ch. 41](../chapitres/41-io-flux.md) | Fichiers, permissions, répertoires, variables d'environnement, `os.Root` (1.24) |
| `os/exec` | [Ch. 48](../chapitres/48-processus-signaux-cli.md) | Lancement et pilotage de processus externes |
| `os/signal` | [Ch. 48](../chapitres/48-processus-signaux-cli.md) | Capture des signaux OS (`SIGINT`, `SIGTERM`) ; arrêt gracieux |
| `path` | [Ch. 50](../chapitres/50-fichiers-fs.md) | Chemins à séparateur `/` (URL, `fs.FS`) — indépendant de l'OS |
| `path/filepath` | [Ch. 50](../chapitres/50-fichiers-fs.md) | Chemins **du système de fichiers local** (`Join`, `Clean`, `WalkDir`, `Glob`) |
| `reflect` | [Ch. 34](../chapitres/34-reflexion.md), [Ch. 33](../chapitres/33-interfaces-profondeur.md) | Réflexion : inspecter et manipuler les types/valeurs à l'exécution |
| `regexp`, `regexp/syntax` | [Annexe J](J-regexp.md) | Expressions régulières RE2 (temps linéaire garanti, pas de backtracking) |
| `runtime` | [Ch. 24](../chapitres/24-runtime-bootstrap.md), [Ch. 27](../chapitres/27-garbage-collector.md), [Ch. 28](../chapitres/28-ordonnanceur-gmp.md), [Ch. 29](../chapitres/29-observabilite-runtime.md) | Interface avec le runtime : GC, ordonnanceur, `GOMAXPROCS`, `MemStats` |
| `runtime/pprof` | [Ch. 37](../chapitres/37-profiling-pprof.md) | Profils CPU/mémoire/blocage pour `pprof` |
| `runtime/trace` | [Ch. 38](../chapitres/38-traces-flight-recorder.md) | Traces d'exécution fines ; Flight Recorder (1.25) |
| `slices` | [Ch. 06](../chapitres/06-arrays-slices.md), [Ch. 30](../chapitres/30-slices-profondeur.md), [Ch. 11](../chapitres/11-genericite.md) | Fonctions génériques sur les slices (`Sort`, `Contains`, `Insert`, `Delete`…) |
| `sort` | [Annexe D](D-algorithmes.md) | Tri historique (interface `sort.Interface`) — souvent supplanté par `slices.Sort` |
| `strconv` | [Ch. 07](../chapitres/07-maps-strings.md), [Ch. 31](../chapitres/31-strings-profondeur.md) | Conversions `string` ↔ nombres/bool (`Atoi`, `Itoa`, `ParseFloat`, `Quote`) |
| `strings` | [Ch. 07](../chapitres/07-maps-strings.md), [Ch. 31](../chapitres/31-strings-profondeur.md), [Ch. 41](../chapitres/41-io-flux.md) | Manipulation de chaînes immuables ; `Builder`, itérateurs `Lines`/`SplitSeq` (1.24) |
| `sync` | [Ch. 21](../chapitres/21-synchronisation.md) | Primitives : `Mutex`, `RWMutex`, `WaitGroup`, `Once`, `Pool`, `Map` |
| `sync/atomic` | [Ch. 21](../chapitres/21-synchronisation.md), [Ch. 25](../chapitres/25-modele-memoire.md) | Opérations atomiques lock-free ; types `atomic.Int64`, `atomic.Pointer` |
| `testing` | [Ch. 13](../chapitres/13-tests-outillage.md), [Ch. 36](../chapitres/36-tests-benchmarks-fuzzing.md) | Tests, benchmarks (`testing.B`), fuzzing (`testing.F`), `t.TempDir` |
| `testing/fstest` | [Ch. 50](../chapitres/50-fichiers-fs.md) | `MapFS` : système de fichiers en mémoire pour les tests |
| `testing/synctest` | [Ch. 44](../chapitres/44-temps.md) | Test de code temporel avec horloge **virtuelle** (1.25) |
| `text/template` | [Ch. 49](../chapitres/49-templates.md) | Gabarits texte (rapports, config, code) ; langage d'actions, `FuncMap` |
| `time` | [Ch. 44](../chapitres/44-temps.md) | Instants, durées, horloge monotone, timers/tickers, fuseaux, deadlines |
| `unicode/utf8` | [Ch. 31](../chapitres/31-strings-profondeur.md), [Ch. 07](../chapitres/07-maps-strings.md) | Décodage/encodage UTF-8, runes, `RuneCountInString` |
| `unsafe` | [Ch. 35](../chapitres/35-unsafe-cgo.md) | Pointeurs bruts, `Sizeof`/`Offsetof`, interopérabilité bas niveau et cgo |

---

## 🔁 Voir aussi

- [Annexe B — Antisèche des commandes `go`](B-antiseche-go.md) : l'outillage `go` (build, test, mod…), complémentaire de cet index des packages.
- [Annexe C — Nouveautés 1.21 → 1.26](C-nouveautes-1.21-1.26.md) : ce qui a été ajouté à ces packages au fil des versions.
- [Annexe G — Ressources](G-ressources.md) : documentation officielle et `pkg.go.dev` pour la référence exhaustive de chaque package.
