# Chapitre 47 — Sécurité & chaîne d'approvisionnement

> **Objectif** : adopter les réflexes de sécurité qui comptent vraiment en Go, des
> deux côtés de la barrière : **la chaîne d'approvisionnement** (d'où vient le code
> que vous compilez, comment garantir qu'il n'a pas changé) et **le code lui-même**
> (aléa, secrets, templates, fichiers, TLS). On reste au plus près de la
> bibliothèque standard et de l'outillage `go`.

La sécurité n'est pas un module à brancher en fin de projet : c'est une suite de
petits choix par défaut. Go aide beaucoup (binaire statique, pas d'interpréteur à
durcir, `html/template` qui échappe tout seul, sommes de modules vérifiées), mais
quelques pièges classiques restent à votre charge. Ce chapitre les catalogue.

```
   CHAÎNE D'APPROVISIONNEMENT                 CODE EXÉCUTÉ
   +-----------------------------+            +--------------------------+
   |  modules tiers (GOPROXY)    |            |  aléa  : crypto/rand     |
   |  go.sum + GOSUMDB           |  --build-> |  secrets: ne pas logguer |
   |  govulncheck (OSV)          |            |  HTML  : html/template   |
   |  toolchain épinglée         |            |  SQL   : requête liée    |
   |  build reproductible        |            |  fichiers: os.Root       |
   +-----------------------------+            |  réseau : TLS >= 1.2     |
                                              +--------------------------+
```

---

## 1. La chaîne d'approvisionnement des modules

### 1.1 `go.sum` et la base de sommes

Chaque dépendance est figée dans `go.mod` (version) **et** dans `go.sum` (empreinte
cryptographique du contenu). À chaque build, le toolchain recalcule l'empreinte des
modules téléchargés et la compare à `go.sum` : toute altération (proxy compromis,
réécriture d'un tag) fait **échouer** la compilation.

```
  go.mod : example.com/lib v1.4.0
  go.sum : example.com/lib v1.4.0       h1:Ab3...=   (somme du contenu)
           example.com/lib v1.4.0/go.mod h1:Zz9...=   (somme du go.mod)
```

Au premier ajout d'une dépendance, le toolchain consulte la **base de sommes
publique** (`GOSUMDB`, par défaut `sum.golang.org`), un journal transparent et
infalsifiable. La somme y est vérifiée **une fois**, puis figée dans votre `go.sum`.

| Variable       | Rôle                                                                          |
| -------------- | ----------------------------------------------------------------------------- |
| `GOPROXY`      | miroir des modules (`proxy.golang.org` par défaut, puis `direct`)             |
| `GOSUMDB`      | base de sommes (`sum.golang.org`) ; `off` la désactive (⚠️ déconseillé)       |
| `GOPRIVATE`    | motifs de modules privés : court-circuite proxy **et** GOSUMDB                |
| `GONOSUMCHECK` | (hérité) ne pas vérifier les sommes — à éviter                                |
| `GOFLAGS`      | drapeaux par défaut, ex. `-mod=readonly` pour interdire les modifs implicites |

💡 Pour du code d'entreprise : `GOPRIVATE=*.mon-entreprise.com` évite que des noms
de modules internes ne fuitent vers le proxy public.

### 1.2 Vérifier, verrouiller, auditer

```
  go mod verify     # recalcule et compare toutes les sommes du cache
  go mod tidy        # synchronise go.mod/go.sum avec les imports réels
  go mod download    # pré-télécharge (utile en CI, cache reproductible)
  go mod vendor      # copie les dépendances dans vendor/ (build hors-ligne auditable)
```

En CI, compilez avec **`-mod=readonly`** (le défaut depuis Go 1.16) pour qu'un build
ne modifie jamais `go.mod`/`go.sum` en douce : une dépendance manquante devient une
erreur, pas un ajout silencieux.

### 1.3 La sélection minimale de versions, propriété de sécurité

Go choisit les versions par **MVS** (_Minimal Version Selection_) : la version
retenue est la **plus basse** qui satisfait toutes les contraintes, jamais « la plus
récente disponible ». Conséquence directe : vos builds sont **reproductibles et
prévisibles** — personne ne peut vous pousser une mise à jour surprise en publiant
un nouveau tag. La mise à jour est un acte explicite (`go get module@version`).

### 1.4 Épingler le toolchain

🆕 Depuis Go 1.21, `go.mod` peut figer la version du compilateur :

```
  go 1.26
  toolchain go1.26.4
```

Combiné à `GOTOOLCHAIN` (`auto`, `local`, ou une version exacte), cela garantit que
toute l'équipe et la CI compilent avec **le même** toolchain — un maillon de la
chaîne d'approvisionnement souvent oublié (renvoi 🔁 ch. 1 et 12).

### 1.5 Builds reproductibles et auditables

```
  go build -trimpath -buildvcs=true -ldflags="-s -w" ./...
```

- `-trimpath` retire les chemins absolus du binaire → deux machines produisent le
  même binaire (reproductibilité, pas de fuite d'arborescence locale).
- `-buildvcs=true` embarque la révision VCS, lisible via `runtime/debug.ReadBuildInfo`
  (`vcs.revision`, `vcs.modified`) — traçabilité « quel commit a produit ce binaire ? »
  (renvoi 🔁 ch. 46).

🧪 **À tester soi-même** : lancez `go version -m ./votre-binaire` : il affiche le
module, ses dépendances **avec leurs sommes**, et les réglages de build. Un audit
complet de provenance en une commande.

---

## 2. `govulncheck` : ne corriger que ce qui vous concerne

`govulncheck` (paquet `golang.org/x/vuln`) croise vos dépendances avec la base de
vulnérabilités **OSV** de l'écosystème Go. Sa force : l'analyse est guidée par
l'**appel de symboles**. Une CVE dans une fonction que vous n'appelez jamais n'est
**pas** signalée → très peu de faux positifs, contrairement aux scanners qui ne
regardent que les numéros de version.

```
  go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

```
  Vulnerability #1: GO-2024-XXXX
    ... appelée depuis votre code via :
      main.handler -> lib.Parse -> lib.unsafeDecode
    Correction : mettre lib à jour vers v1.4.1
```

💡 Intégrez-le en CI comme une étape bloquante. Couplé à `go vet ./...` et aux
analyzers (renvoi 🔁 ch. 13), c'est votre filet de sécurité statique.

---

## 3. Aléa et comparaisons de secrets

### 3.1 `crypto/rand`, jamais `math/rand` pour un secret

⚠️ `math/rand`/`math/rand/v2` est **prévisible** (générateur déterministe) : parfait
pour des simulations, **catastrophique** pour un jeton de session, un mot de passe
temporaire ou un identifiant non devinable. Pour tout ce qui doit résister à un
attaquant, utilisez **`crypto/rand`**.

🆕 Go 1.24 ajoute `crypto/rand.Text()`, qui renvoie une chaîne base32 avec au moins
128 bits d'entropie — l'outil idéal pour un jeton :

```go
func newToken() string {
    return rand.Text() // crypto/rand : sûr par construction
}
```

### 3.2 Comparer en temps constant

⚠️ Comparer un secret avec `==` (ou `bytes.Equal`) **s'arrête au premier octet
différent**. La durée de la comparaison fuit alors la position de la différence —
un attaquant peut reconstituer le secret octet par octet (_timing attack_).
Utilisez `crypto/subtle.ConstantTimeCompare` :

```go
func equalTokens(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
```

### 3.3 Mots de passe

Ne stockez **jamais** un mot de passe en clair, et ne le hachez **pas** avec un
hachage rapide nu (`sha256`) : un GPU en teste des milliards par seconde. Utilisez
une fonction de dérivation **lente et salée** — `bcrypt`, `scrypt` ou `argon2`, via
`golang.org/x/crypto` (hors stdlib, mais maintenu par l'équipe Go).

⚡ **Performance** : la lenteur est ici une **fonctionnalité**. On règle le coût
(`bcrypt.DefaultCost`, paramètres argon2) pour qu'un hachage prenne ~100 ms : assez
rapide pour un login légitime, ruineux pour une attaque par force brute.

---

## 4. Templates : `html/template` contre le XSS

C'est sans doute le piège le plus fréquent côté web. `text/template` et
`html/template` ont la **même API** mais des comportements opposés :

- `html/template` **échappe automatiquement** selon le contexte (HTML, attribut,
  URL, JavaScript, CSS). Du contenu utilisateur devient inoffensif.
- `text/template` **n'échappe rien**. Rendu dans une page HTML, c'est une faille XSS.

```
  entrée utilisateur : <script>alert(1)</script>

  html/template  ->  &lt;script&gt;alert(1)&lt;/script&gt;     (inerte)
  text/template  ->  <script>alert(1)</script>                 (exécuté !)
```

```go
// html/template échappe selon le contexte d'insertion.
t := htmltemplate.Must(htmltemplate.New("greet").Parse(`<p>Bonjour {{.}}</p>`))
```

⚠️ Forcer du contenu via `template.HTML`, `template.JS`, etc. **désactive**
l'échappement pour cette valeur : ne le faites que sur du contenu que **vous**
produisez, jamais sur une entrée externe.

---

## 5. Injection SQL : toujours des requêtes paramétrées

`database/sql` rend la défense triviale : passez les valeurs comme **paramètres**
(`?` ou `$1` selon le driver), jamais par concaténation de chaîne. Le driver les
transmet séparément de la requête, donc une valeur ne peut pas devenir du SQL.

```go
// ❌ Injection : l'entrée devient du code SQL.
db.Query("SELECT * FROM users WHERE name = '" + name + "'")

// ✅ Paramétré : name est une donnée, jamais du SQL.
db.QueryContext(ctx, "SELECT * FROM users WHERE name = $1", name)
```

💡 La même règle vaut pour les commandes système : préférez `exec.Command("git",
"clone", url)` (arguments séparés) à un appel via un shell qui interprète la chaîne.

---

## 6. Accès fichiers : barrer la traversée de répertoire

Quand un nom de fichier vient de l'extérieur (téléchargement, paramètre d'URL), un
`../../etc/passwd` peut sortir du dossier prévu. ⚠️ `filepath.Clean` **seul ne
suffit pas** (il normalise mais n'empêche pas la remontée).

Deux outils stdlib :

- `io/fs.ValidPath(name)` : valide un chemin **relatif, propre, sans `..` ni `/`
  initial**. Idéal pour filtrer une entrée avant usage.
- 🆕 `os.OpenRoot(dir)` (Go 1.24) renvoie un `*os.Root` qui **confine** tous les
  accès sous `dir`, y compris à travers les liens symboliques. Une traversée échoue,
  même si la cible existe.

```go
func readWithinRoot(dir, name string) ([]byte, error) {
    root, err := os.OpenRoot(dir) // tout accès reste sous dir
    if err != nil {
        return nil, err
    }
    defer root.Close()
    f, err := root.Open(name) // "../secret" -> erreur
    if err != nil {
        return nil, err
    }
    defer f.Close()
    return io.ReadAll(f)
}
```

---

## 7. Réseau : HTTP et TLS durcis

### 7.1 Limiter et expirer

- ⚠️ Un serveur **sans timeout** est vulnérable au déni de service (connexions lentes
  qui s'accumulent). Renseignez `ReadTimeout`, `WriteTimeout`, `IdleTimeout`,
  `ReadHeaderTimeout` sur `http.Server` (renvoi 🔁 ch. 45).
- `http.MaxBytesReader(w, r.Body, n)` plafonne la taille d'un corps de requête : au
  delà, la lecture renvoie une erreur. Indispensable pour les uploads et le JSON.
- 🆕 Go 1.25 : `http.CrossOriginProtection` rejette les requêtes _cross-origin_ non
  sûres (anti-CSRF), via l'en-tête `Sec-Fetch-Site`.

### 7.2 TLS

```go
func hardenedTLSConfig() *tls.Config {
    return &tls.Config{
        MinVersion: tls.VersionTLS12, // refuser les versions obsolètes
        // InsecureSkipVerify reste false : la vérif. de certificat est active.
    }
}
```

⚠️ `InsecureSkipVerify: true` **désactive la vérification de certificat** : la
connexion devient interceptable (_man-in-the-middle_). C'est un réglage de débogage,
**jamais** de production. Si vous le voyez en revue de code, c'est un signal d'alarme.

🆕 **FIPS 140-3** : Go 1.24 introduit un mode de conformité cryptographique activable
par `GOFIPS140=on` (au build) ou `GODEBUG=fips140=on` (à l'exécution), qui restreint
la cryptographie aux algorithmes validés. Utile en environnement réglementé.

---

## 8. Catalogue de pièges ❌ → ✅

| Sujet           | ❌ Vulnérable                         | ✅ Correct                                      |
| --------------- | ------------------------------------- | ----------------------------------------------- |
| Jeton aléatoire | `math/rand`                           | `crypto/rand` / `rand.Text` (1.24)              |
| Comparer secret | `a == b`                              | `subtle.ConstantTimeCompare`                    |
| Mot de passe    | `sha256(pwd)`                         | `bcrypt`/`argon2` (x/crypto), salé, lent        |
| Gabarit HTML    | `text/template` dans du HTML          | `html/template` (échappement contextuel)        |
| SQL             | concaténation de chaîne               | requête paramétrée (`$1`/`?`)                   |
| Chemin fichier  | `filepath.Join(dir, input)` brut      | `fs.ValidPath` + `os.Root` (1.24)               |
| TLS             | `InsecureSkipVerify: true`            | `MinVersion`, vérif. active                     |
| Corps HTTP      | lecture non bornée                    | `http.MaxBytesReader` + timeouts serveur        |
| Dépendances     | `GOFLAGS=-mod=mod` en CI, pas d'audit | `-mod=readonly`, `go mod verify`, `govulncheck` |
| Secrets         | logger l'objet entier                 | `slog.LogValuer` qui masque (ch. 43)            |

---

## 9. Checklist pré-déploiement

- [ ] `go mod verify` OK et `go.sum` à jour ; build CI en `-mod=readonly`.
- [ ] `govulncheck ./...` sans vulnérabilité atteinte (étape CI bloquante).
- [ ] `toolchain` épinglée dans `go.mod`.
- [ ] Build `-trimpath -buildvcs=true` ; provenance vérifiable (`go version -m`).
- [ ] Aucun `math/rand` pour un secret ; comparaisons de secrets en temps constant.
- [ ] Mots de passe via `bcrypt`/`argon2`, jamais en clair ni en hachage rapide.
- [ ] Rendu HTML via `html/template` uniquement ; pas de `template.HTML` sur entrée externe.
- [ ] Requêtes SQL paramétrées partout.
- [ ] Entrées de chemins validées (`fs.ValidPath`/`os.Root`).
- [ ] Serveur HTTP avec timeouts + `MaxBytesReader` ; TLS ≥ 1.2, pas de `InsecureSkipVerify`.
- [ ] Aucun secret dans les logs, le code ou le dépôt (`.env`, clés exclus du commit).

---

## 📌 À retenir

- **Deux fronts** : la _provenance_ du code (go.sum, GOSUMDB, `govulncheck`, toolchain
  épinglée, builds reproductibles) et le _code_ lui-même (aléa, secrets, templates,
  fichiers, TLS).
- **Le défaut de Go aide** : sommes vérifiées, MVS reproductible, `html/template` qui
  échappe — mais `text/template`, `InsecureSkipVerify` et `math/rand` sont des pièges
  à connaître.
- **`crypto/rand` pour tout secret**, `subtle.ConstantTimeCompare` pour les comparer,
  `bcrypt`/`argon2` pour les mots de passe.
- **Confiner et borner** : `os.Root` contre la traversée, `MaxBytesReader` et les
  timeouts contre le déni de service.
- **Automatiser** : `go mod verify` + `govulncheck` + `-mod=readonly` en CI valent
  mieux que toute vigilance manuelle.

🔁 Renvois : ch. 1 (toolchain), ch. 12 (modules), ch. 43 (secrets & `slog`), ch. 45
(timeouts HTTP, CSRF), ch. 46 (builds reproductibles), ch. 35 (`runtime/secret`),
Projet 2 (API REST).
