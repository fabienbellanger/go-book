# Plan détaillé — Générateur de site HTML du livre

> Document de conception du **générateur de site statique maison**, écrit en Go,
> qui transforme les fichiers Markdown du livre en un site HTML lisible et
> recherchable. **À valider/amender avant implémentation** — aucune écriture de
> code n'est lancée tant que ce plan n'est pas approuvé.

---

## 1. Objectif & cahier des charges

Produire, à partir des **57 fichiers Markdown** existants (`chapitres/`, `annexes/`,
`SOMMAIRE.md`, `README.md`, projets), un **site HTML statique** offrant :

| Exigence (demande utilisateur)      | Traduction technique                                                            |
| ----------------------------------- | ------------------------------------------------------------------------------- |
| **Design sympa et moderne**         | Layout 2 colonnes (sidebar + contenu), typographie soignée, responsive mobile.  |
| **Dark / light mode**               | Variables CSS + bascule JS mémorisée dans `localStorage`, respect de `prefers-color-scheme`. |
| **Recherche dans chapitres/annexes**| Index JSON généré au build + recherche plein-texte côté client (zéro dépendance JS). |
| **Rendu « propre type GitHub »**    | Feuille de style inspirée de `github-markdown-css` (light + dark) + coloration syntaxique type GitHub. |

### Principes directeurs

- **Stdlib-first**, fidèle à l'esprit du livre : tout ce qui peut être fait en
  bibliothèque standard l'est (`html/template`, `io/fs`, `embed`, `encoding/json`,
  `flag`, `path/filepath`). **Deux dépendances externes seulement** (voir §4).
- **Site 100 % statique** : aucun serveur requis pour lire (ouvrable en `file://`),
  publiable tel quel sur **GitHub Pages** ou tout hébergement de fichiers.
- **Zéro modification du contenu** : le générateur **lit** les `.md`, il ne les
  réécrit jamais. La source de vérité reste le Markdown.
- **Idempotent & reproductible** : `go run ./tools/site` régénère intégralement
  `public/` à chaque fois.
- **Le générateur comme vitrine** : ce programme illustre `goldmark`, `html/template`,
  `//go:embed`, `io/fs`, `flag` — il pourra être référencé comme exemple avancé.

### Hors périmètre (pour cette première version)

- Pas de PDF/EPUB (le Markdown reste la source ; un export pourra venir plus tard).
- Pas de moteur de recherche serveur (fuzzy avancé) : plein-texte client suffit
  largement pour ~57 documents.
- Pas de versionnage multi-langues du site.

---

## 2. Emplacement & arborescence

Le générateur vit dans `tools/site/` **en tant que module Go séparé** (comme les
projets), pour ne pas alourdir le module `example.com/gobook` du dossier `code/`
avec les dépendances `goldmark`/`chroma`.

```
go-book/
  tools/
    site/
      go.mod                 module example.com/gobook-site (go 1.26)
      go.sum
      main.go                point d'entrée + flags (flag)
      internal/
        model/               types: Book, Part, Page, Section, SearchDoc
          model.go
        sommaire/            parse SOMMAIRE.md -> arbre de navigation
          sommaire.go
          sommaire_test.go
        render/              Markdown -> HTML (goldmark + chroma), réécriture liens
          render.go
          render_test.go
        site/                assemblage: gabarits, écriture public/, copie assets
          builder.go
          builder_test.go
        search/              construction de l'index de recherche (JSON)
          index.go
          index_test.go
      assets/                fichiers statiques EMBARQUÉS via //go:embed
        css/
          github-markdown.css    rendu type GitHub (light + dark)
          layout.css             sidebar, header, responsive, callouts emoji
        js/
          theme.js               bascule dark/light + localStorage
          search.js              recherche client (fetch index + rendu résultats)
        templates/
          base.html.tmpl         squelette commun (head, header, sidebar, footer)
          page.html.tmpl         page de contenu (chapitre/annexe)
          index.html.tmpl        page d'accueil / sommaire
          search.html.tmpl       page de résultats de recherche
      README.md              comment générer & publier le site
  public/                    SORTIE générée (ajoutée au .gitignore)
```

> 📌 **Décision** : module séparé `tools/site/` + sortie `public/` ignorée par git.
> Le site se régénère à la demande, on ne versionne pas le HTML produit (sauf si on
> publie via une branche `gh-pages` dédiée — voir §11).

---

## 3. Pipeline de génération (vue d'ensemble)

```
   SOMMAIRE.md ──► [sommaire] ──► arbre Book{Parts[],Pages[]}
                                        │
   chapitres/*.md  ──┐                  │
   annexes/*.md   ───┼─► [render] ──► HTML par page (+ titres/ancres + texte brut)
   README.md      ──┘     goldmark        │
                          + chroma        │
                                          ▼
                                    [search] ──► public/search-index.json
                                          │
                                          ▼
                                    [site/builder]
                                    html/template
                                          │
                          ┌───────────────┼───────────────────┐
                          ▼               ▼                    ▼
                 public/*.html    public/assets/*    public/search-index.json
```

Étapes :

1. **Parse navigation** — lire `SOMMAIRE.md`, en extraire l'arbre Parties → Pages
   (titre, lien `.md`, partie d'appartenance). C'est l'ordre canonique du livre.
2. **Rendu des pages** — pour chaque `.md` référencé : convertir en HTML, extraire
   les titres (pour la table des matières par page + ancres), récupérer le texte
   brut (pour l'index de recherche), réécrire les liens internes `.md → .html`.
3. **Index de recherche** — agréger `{page, titre, section, texte}` en un
   `search-index.json` léger.
4. **Assemblage** — injecter chaque page dans le gabarit (avec sidebar globale +
   ToC locale), écrire les `.html`, copier/écrire les assets embarqués.

---

## 4. Dépendances

| Rôle                         | Paquet                                   | Justification                                         |
| ---------------------------- | ---------------------------------------- | ----------------------------------------------------- |
| Markdown → HTML              | `github.com/yuin/goldmark`               | Standard de l'écosystème (moteur de Hugo), conforme CommonMark + GFM (tables, autolinks, strikethrough, task lists). |
| Coloration syntaxique        | `github.com/alecthomas/chroma/v2`        | Coloration Go/bash/json… ; intégrable à goldmark via `goldmark-highlighting`. |
| (liant) highlighting goldmark| `github.com/yuin/goldmark-highlighting/v2`| Connecte chroma à goldmark proprement.               |

Tout le reste = **stdlib** : `html/template`, `text/template` (au besoin), `io/fs`,
`embed`, `encoding/json`, `flag`, `path`, `path/filepath`, `os`, `strings`, `bufio`,
`regexp` (parse léger du SOMMAIRE), `log/slog` (logs de build).

> 💡 Alternative « zéro dépendance » envisagée puis écartée : écrire notre propre
> parseur Markdown serait disproportionné et fragile. `goldmark` est le bon
> compromis (sûr, maintenu, GFM complet). Choix assumé.

---

## 5. Modèle de données (`internal/model`)

```go
// Book = livre complet, prêt à rendre.
type Book struct {
    Title string
    Parts []Part   // ordre du SOMMAIRE
    Pages []*Page  // toutes les pages, à plat (pour recherche & build)
}

// Part = section du sommaire (« Partie I — Fondamentaux… »).
type Part struct {
    Title string
    Pages []*Page
}

// Page = un fichier Markdown rendu.
type Page struct {
    SrcPath   string     // ex: chapitres/41-io-flux.md
    OutPath   string     // ex: chapitres/41-io-flux.html
    Title     string     // titre H1
    PartTitle string     // partie d'appartenance (sidebar)
    HTML      template.HTML
    Headings  []Heading  // ToC locale (H2/H3 + ancres)
    PlainText string     // texte brut pour l'index de recherche
    Prev, Next *Page     // navigation séquentielle (préc./suiv.)
}

type Heading struct {
    Level int    // 2 ou 3
    Text  string
    ID    string // ancre slugifiée
}

// SearchDoc = entrée de l'index JSON.
type SearchDoc struct {
    URL     string `json:"url"`
    Title   string `json:"title"`
    Part    string `json:"part"`
    Content string `json:"content"` // texte brut tronqué/normalisé
}
```

---

## 6. Composants détaillés

### 6.1 `sommaire` — parsing de la navigation

- Entrée : `SOMMAIRE.md`. Format réel : titres `## Partie …`, listes `- Ch. N — [titre](lien.md)`.
- Stratégie : lecture ligne à ligne (`bufio.Scanner`), un `regexp` pour repérer les
  liens `[texte](chemin.md)` et un autre pour les en-têtes de partie.
- Sortie : `[]Part` ordonné, chaque `Page` avec `SrcPath` + `Title` provisoire (le
  titre définitif vient du H1 du fichier au rendu).
- ⚠️ **Pièges à gérer** :
  - Liens vers des **dossiers** (`projets/1-cli/`) → page d'index listée mais pas
    rendue comme chapitre (ou lien externe vers le repo — à décider, voir §12).
  - Notes en blockquote sous un titre de partie (« > Lisible dès la fin… ») → ignorées
    pour la nav, ou conservées comme description de partie (option).
- Tests : un `SOMMAIRE` minimal en entrée → vérifier nombre de parties/pages, ordre,
  chemins.

### 6.2 `render` — Markdown → HTML

- Configurer `goldmark` avec extensions **GFM** (tables, autolinks, strikethrough,
  task lists), **footnotes**, et **IDs de titres automatiques** (ancres).
- Brancher **chroma** via `goldmark-highlighting` : thème adapté light/dark (deux
  passes CSS, ou classes CSS + feuille chroma générée).
- **Réécriture des liens internes** : transformer les `href` se terminant par `.md`
  en `.html` (en préservant l'ancre `#...`). Implémenté via un *AST transformer*
  goldmark (propre) plutôt que par regex sur le HTML.
- **Extraction** pendant le rendu :
  - `Title` = premier H1.
  - `Headings` = H2/H3 avec leurs IDs (ToC locale).
  - `PlainText` = concaténation des nœuds texte (pour l'index ; on retire le code ?
    → à décider : garder le code aide à chercher des identifiants comme `io.Copy`).
- **Marqueurs emoji du livre** (🆕 ⚠️ 💡 🔁 ⚡ 🧪 📌) : ils vivent souvent dans des
  blockquotes. Option de style : détecter le premier emoji d'un blockquote et lui
  appliquer une **classe de callout** colorée (ex. `callout--new`, `callout--warn`).
  Implémenté par un transformer AST ou en post-traitement CSS (`:has()`/attribut).
- Tests : Markdown d'entrée connu → vérifier présence d'ancres, réécriture `.md→.html`,
  blocs `<pre>` colorés, texte brut extrait.

### 6.3 `search` — index de recherche

- Construire `[]SearchDoc` (une entrée par page, voire par section H2 pour des
  résultats plus fins).
- **Normalisation** : minuscules, suppression des accents (table simple ou
  `golang.org/x/text`… → on **évite** la dépendance, on fait une table maison de
  remplacement des diacritiques français : é→e, à→a, ç→c, …).
- Sérialiser en `public/search-index.json` (compact). Taille estimée : quelques
  centaines de Ko pour tout le livre — acceptable pour un fetch unique au chargement.
- Côté client (`search.js`, voir §7) : `fetch` de l'index, filtrage par sous-chaîne
  + scoring (titre > section > corps), surlignage des occurrences, résultats cliquables.
- Tests : index non vide, URLs valides, contenu normalisé sans accents.

### 6.4 `site/builder` — assemblage & écriture

- Charger les gabarits embarqués (`//go:embed assets/templates/*.tmpl`) via
  `template.ParseFS`.
- Pour chaque `Page` : exécuter `page.html.tmpl` avec `{Book, Page}` → écrire le
  `.html` (création des dossiers `public/chapitres`, `public/annexes`).
- Générer `public/index.html` (sommaire) et `public/search.html` (page recherche).
- **Copier les assets statiques** embarqués (`css/`, `js/`) vers `public/assets/` via
  `fs.WalkDir` sur le `embed.FS`.
- Calculer les liens **préc./suivant** (`Prev`/`Next`) à partir de l'ordre du SOMMAIRE.
- ⚠️ Sécurité d'écriture : tout reste sous `public/` ; pas d'écriture hors dossier de
  sortie (chemins nettoyés via `path/filepath`).

---

## 7. Front-end (CSS / JS)

### Design

- **Layout** : sidebar gauche fixe (arbre du SOMMAIRE, chapitre courant surligné) +
  colonne de contenu centrée (largeur de lecture ~ 80 caractères) + ToC locale à
  droite sur grand écran. **Responsive** : sidebar repliable en menu burger sur mobile.
- **Typographie** : système (`-apple-system`, `Segoe UI`, …) pour le texte ;
  police mono (`ui-monospace`, `SFMono`, `Menlo`) pour le code et **les schémas ASCII**
  (essentiel : `white-space: pre`, pas de retour à la ligne, pas de ligatures).
- **Rendu type GitHub** : reprise des conventions `github-markdown-css` (titres avec
  filet, tableaux zébrés, `blockquote` à barre latérale, `code` inline à fond grisé).

### Dark / light

- Palette via **variables CSS** (`--bg`, `--fg`, `--accent`, `--code-bg`, …), deux
  thèmes `:root` / `[data-theme="dark"]`.
- Respect initial de `prefers-color-scheme`, **bascule manuelle** (bouton dans le
  header) persistée en `localStorage`. Coloration chroma : deux jeux de classes ou
  filtres adaptés au thème.

### Callouts emoji

Mapper chaque marqueur du livre à un style visuel :

| Emoji | Sens          | Style callout        |
| ----- | ------------- | -------------------- |
| 🆕    | nouveauté     | bandeau bleu         |
| ⚠️    | piège         | bandeau orange/rouge |
| 💡    | astuce        | bandeau vert         |
| 🔁    | renvoi        | bandeau gris         |
| ⚡    | perf          | bandeau violet       |
| 🧪    | à tester      | bandeau cyan         |
| 📌    | à retenir     | bandeau ambre        |

### JavaScript (vanilla, sans framework)

- `theme.js` : lit/écrit le thème, applique `data-theme`, écoute le bouton.
- `search.js` : champ de recherche dans le header (raccourci `/` pour focus),
  `fetch('search-index.json')` au premier usage, filtrage + scoring + surlignage,
  liste de résultats en surimpression, navigation clavier (↑/↓/Entrée).
- **Aucune dépendance JS externe** : tout est écrit à la main (< 150 lignes au total).

---

## 8. Interface en ligne de commande (`main.go`)

```
go run ./tools/site [flags]

Flags :
  -src    string   racine du livre à lire           (défaut ".")
  -out    string   dossier de sortie                (défaut "public")
  -serve           sert le résultat en HTTP local + (option) live reload
  -addr   string   adresse du serveur de prévisualisation (défaut ":8080")
  -clean           vide le dossier de sortie avant génération
  -v               logs verbeux (slog niveau debug)
```

- Mode par défaut : **build** unique → `public/`.
- Mode `-serve` : build puis `http.FileServer` sur `-addr` pour prévisualiser
  (pratique pendant la rédaction). Le live-reload est optionnel (v2).
- Logs via `log/slog` (compte des pages rendues, durée, avertissements liens cassés).

> 🧪 **Vérification liens** : à la fin du build, signaler les liens internes qui
> pointent vers un fichier inexistant (utile vu les renvois 🔁 nombreux du livre).

---

## 9. Makefile — lancer, tester, builder le site

Un `Makefile` dédié dans `tools/site/` (même style que `projets/1-cli/Makefile`)
offre des raccourcis pour les tâches courantes. Il enrobe simplement `go run` /
`go build` / `go test`.

### Cibles

| Cible            | Effet                                                                 |
| ---------------- | --------------------------------------------------------------------- |
| `make build`     | Génère le site statique dans `public/` (`go run . -clean`).           |
| `make serve`     | Génère puis sert le site en local (`go run . -serve -addr :8080`).    |
| `make test`      | `go test -race ./...` sur le module du générateur.                    |
| `make vet`       | `go vet ./...`.                                                       |
| `make fmt`       | `gofmt -l .` (échoue si du code n'est pas formaté).                   |
| `make check`     | Enchaîne `fmt` + `vet` + `test` (porte de qualité avant commit/CI).   |
| `make dist`      | Compile le générateur en binaire statique dans `bin/gobook-site`.     |
| `make clean`     | Supprime `public/` et `bin/`.                                         |

### Esquisse du Makefile (`tools/site/Makefile`)

```make
# Makefile du générateur de site du livre
#
# Cibles principales :
#   make build   génère le site HTML dans ../../public
#   make serve   génère puis sert le site en local (http://localhost:8080)
#   make test    go test -race ./...
#   make check   fmt + vet + test (porte de qualité)
#   make dist    compile le générateur (binaire statique)
#   make clean   supprime public/ et bin/

BINARY  := gobook-site
SRC     := ../..                       # racine du livre (où sont chapitres/, annexes/)
OUT     := $(SRC)/public               # dossier de sortie du site
ADDR    ?= :8080
LDFLAGS := -s -w

.PHONY: build serve test vet fmt check dist clean

build:
	go run . -clean -src $(SRC) -out $(OUT)

serve:
	go run . -src $(SRC) -out $(OUT) -serve -addr $(ADDR)

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

check: fmt vet test

# Binaire statique du générateur (pratique en CI : build une fois, exécute partout).
dist:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

clean:
	rm -rf bin $(OUT)
```

> 💡 Variables surchargeables : `make serve ADDR=:9000` change le port,
> `make build OUT=/tmp/site` change la destination. Cohérent avec les flags de §8.

> 📌 **À retenir** : le Makefile ne fait qu'appeler la CLI du générateur ; toute la
> logique reste en Go. On peut tout faire « à la main » avec `go run . …`, le
> Makefile n'est qu'un confort.

---

## 10. Tests & qualité

- Tests unitaires par composant (`sommaire`, `render`, `search`, `builder`) avec
  entrées Markdown minimales en *table-driven tests*.
- Test d'intégration : générer le site dans `t.TempDir()` à partir d'un mini-livre
  fixture (2 parties, 3 pages) → vérifier l'existence des `.html`, de l'index JSON,
  des assets, et la réécriture des liens.
- `go test -race ./...`, `go vet ./...`, `gofmt -l` (vide) sur le module `tools/site`.
- Pas de réseau dans les tests (tout en local / fixtures).

---

## 11. Publication (GitHub Pages)

Deux options, à décider (§12) :

1. **Workflow GitHub Actions** : à chaque push sur `main`, installer Go, lancer
   `go run ./tools/site -out public`, publier `public/` sur l'environnement Pages.
   → Le HTML n'est **jamais commité** ; il est construit à la volée. **Recommandé.**
2. **Branche `gh-pages`** : générer localement et pousser le contenu de `public/`.
   Plus manuel, versionne le HTML.

`.gitignore` : ajouter `/public/` (sortie générée) ; ajouter `/tools/site/site/`
n'est pas nécessaire (pas de binaire si on reste en `go run`).

---

## 12. Points à trancher avant implémentation

| # | Question                                                                                  | Proposition par défaut                                           |
| - | ----------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| 1 | Les **projets** (`projets/N-*/`) : pages du site ou simples liens vers le repo GitHub ?   | Lister dans la sidebar, lien vers le code (pas de rendu Markdown du code). Rendre leur `README.md` si présent. |
| 2 | **Granularité de la recherche** : 1 entrée/page ou 1 entrée/section H2 ?                   | Par section H2 (résultats plus précis, ancres directes).         |
| 3 | Inclure le **code des blocs** dans l'index de recherche ?                                  | Oui (chercher `io.Copy`, `slog.Info`… est utile).                |
| 4 | **Publication** : GitHub Actions (build CI) ou branche `gh-pages` ?                        | GitHub Actions, HTML non commité.                                |
| 5 | **Live-reload** en mode `-serve` dès la v1 ?                                               | Non (v2) ; `-serve` simple suffit d'abord.                       |
| 6 | Rendre aussi `README.md`, `PLAN.md`, ce `PLAN_SITE.md` ?                                   | `README.md` comme page d'accueil possible ; PLAN* exclus du site.|
| 7 | **Thème de coloration** chroma (ex. `github` / `github-dark`) ?                            | `github` (light) + `github-dark` (dark), pour coller au style demandé. |

---

## 13. Découpage en lots (ordre d'implémentation proposé)

> Aucun de ces lots n'est lancé tant que le plan n'est pas validé.

1. **Lot 0 — squelette** : module `tools/site`, `model`, `main.go` avec flags, `embed`
   des assets vides, build qui ne fait rien d'utile mais compile + teste.
2. **Lot 1 — navigation** : parser `SOMMAIRE.md` → arbre `Book` (+ tests).
3. **Lot 2 — rendu** : goldmark + chroma, réécriture liens, extraction titres/texte
   (+ tests). Sortie HTML brute par page.
4. **Lot 3 — assemblage** : gabarits `html/template`, sidebar, préc./suiv., écriture
   `public/` + copie assets (+ test d'intégration).
5. **Lot 4 — style** : `github-markdown.css` + `layout.css` + callouts emoji + dark/light.
6. **Lot 5 — recherche** : index JSON (Go) + `search.js` (client) + page de résultats.
7. **Lot 6 — finitions** : `-serve`, vérification des liens cassés, responsive mobile,
   `README.md` du générateur.
8. **Lot 7 (option) — publication** : workflow GitHub Actions Pages.

**Première version « fonctionnelle » = lots 0 à 5** (génération + navigation + style +
recherche). Les lots 6-7 sont du raffinement.

---

## 14. Estimation

- **Go** : ~400-600 lignes (hors gabarits/CSS/JS), réparties sur 4 petits packages.
- **CSS** : ~300-400 lignes (markdown GitHub + layout + thèmes + callouts).
- **JS** : ~150 lignes (thème + recherche), sans dépendance.
- **Gabarits** : 4 fichiers `.tmpl` courts.

Effort modéré, entièrement en Go + front-end vanilla, cohérent avec le livre.
