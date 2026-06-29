# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Nature du dépôt

Ce dépôt est un **livre** en français, « Comprendre et maîtriser Go 1.26 » — pas une application. Le contenu rédactionnel (Markdown) est la livraison principale ; le code Go existe pour être **compilable et vérifiable**, en appui du texte. Trois familles de contenu coexistent :

- `chapitres/` + `annexes/` — le texte du livre, un fichier Markdown par chapitre (`NN-slug.md`) ou annexe (`X-slug.md`).
- `code/` — **un seul module Go** (`example.com/gobook`) regroupant tous les exemples des chapitres, un sous-dossier `chNN-slug/` par chapitre, chacun `package main`.
- `projets/` et `tools/site/` — des modules Go **séparés**, autonomes.

`PLAN.md` (plan de production détaillé) et `PLAN_SITE.md` (conception du générateur de site) sont la source de vérité éditoriale ; `SOMMAIRE.md` est la table des matières et **pilote la navigation du site généré**.

## Commandes

### Valider les exemples du livre (le plus fréquent)

```bash
cd code
go test ./...                  # tous les exemples + tests
go vet ./...                   # analyse statique
go test ./ch06-slices/         # un seul chapitre
go test -run TestReverseInts ./ch06-slices/   # un seul test
go run ./ch06-slices           # exécuter la démo d'un chapitre
```

Prérequis : **Go 1.26+**.

### Projets (`projets/N-*`) — modules indépendants

Chaque projet a son propre `go.mod` et `Makefile`. Travailler **depuis le dossier du projet** (`cd projets/2-api-rest`), pas depuis la racine ni depuis `code/`.

### Générateur de site (`tools/site/`) — module `example.com/gobook-site`

```bash
cd tools/site
make build              # génère le site dans ../../public
make serve              # build + sert sur http://localhost:8080 (ADDR=:9000 pour changer)
make check              # fmt + vet + test (porte de qualité avant commit)
make chroma             # régénère assets/css/chroma.css (après bump de chroma)
```

`public/` est **gitignoré** et reconstruit à la volée ; ne jamais committer de HTML généré. Un workflow (`.github/workflows/site.yml`) publie le site sur GitHub Pages à chaque push sur `main`.

## Conventions de rédaction (à respecter strictement)

- **Langue** : prose en français ; **identifiants de code en anglais, commentaires en français**.
- **Schémas** : ASCII pur uniquement (`+ - | / \ < > v ^`) — **jamais** de caractères box-drawing Unicode (alignement garanti partout).
- **Émojis repères** (sémantiques, repris par le site) : 🆕 nouveauté · ⚠️ piège · 💡 astuce · 🔁 renvoi · ⚡ perf · 🧪 test · 📌 synthèse.
- Un nouveau chapitre **part de `chapitres/_gabarit.md`** (en-tête commenté à supprimer dans la version finale).
- Les blocs de code des chapitres correspondent à un fichier réel et sont annotés par un commentaire de chemin, ex. `// code/ch06-slices/main.go`. Toute modification du texte d'un exemple doit rester synchronisée avec le fichier sous `code/`.
- Densité : précis sans verbosité — privilégier schémas, listes et tableaux.

## Architecture du générateur de site (`tools/site/`)

Pipeline statique en Go pur, qui sert aussi de vitrine des sujets du livre (`//go:embed`, `io/fs`, `html/template`, `flag`, `log/slog`, plus `goldmark`/`chroma`). Pour comprendre un changement, lire ces paquets dans l'ordre du flux :

- `internal/model/` — types partagés (`Book`, `Part`, `Page`, `Heading`, `SearchDoc`).
- `internal/sommaire/` — parse `SOMMAIRE.md` → arbre de navigation (l'ordre des pages vient de là).
- `internal/render/` — Markdown → HTML (goldmark + chroma), réécriture des liens internes `.md`→`.html` et **détection des liens cassés** au build. `gen_chroma/` est un générateur jetable du CSS de coloration.
- `internal/search/` — construit l'index de recherche JSON (recherche plein-texte côté client, insensible aux accents).
- `internal/site/` — assemblage final : applique les gabarits `assets/templates/`, écrit `public/`, copie les assets embarqués.

Les assets (`assets/css`, `assets/js`, `assets/templates`) sont **embarqués via `//go:embed`** dans `main.go` ; le binaire est autonome.

## Git

Les messages de commit de ce dépôt **ne portent pas de trailer `Co-Authored-By`** — ne pas en ajouter.
