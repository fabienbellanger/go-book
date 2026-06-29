# `gobook-site` — générateur de site HTML du livre

Petit générateur de site statique, écrit en Go, qui transforme les fichiers
Markdown du livre (`chapitres/`, `annexes/`, `SOMMAIRE.md`, READMEs des projets)
en un site HTML lisible, navigable et recherchable.

C'est aussi une **vitrine** des sujets du livre : `//go:embed`, `io/fs`,
`html/template`, `flag`, `log/slog`, plus `goldmark` et `chroma` pour le rendu.

## Caractéristiques

- **Site 100 % statique** : ouvrable en `file://`, publiable sur GitHub Pages.
- **Rendu type GitHub** : `github-markdown-css` maison + coloration `chroma`.
- **Dark / light** : variables CSS, bascule mémorisée (`localStorage`), respect de
  `prefers-color-scheme`.
- **Recherche plein-texte côté client** : index JSON généré au build, zéro
  dépendance JS, insensible aux accents.
- **Navigation** : sidebar (arbre du SOMMAIRE), ToC locale par page, liens
  précédent/suivant, callouts emoji (🆕 ⚠️ 💡 🔁 ⚡ 🧪 📌).
- **Vérification des liens** : les liens internes cassés sont signalés au build.

## Utilisation

Depuis ce dossier (`tools/site/`) :

```bash
make build              # génère le site dans ../../public
make serve              # génère puis sert sur http://localhost:8080
make serve ADDR=:9000   # autre port
make check              # fmt + vet + test (porte de qualité)
```

Ou directement via la CLI :

```bash
go run . -clean -src ../.. -out ../../public
go run . -src ../.. -serve -addr :8080
```

### Flags

| Flag     | Défaut                            | Effet                                     |
| -------- | --------------------------------- | ----------------------------------------- |
| `-src`   | `.`                               | racine du livre à lire                    |
| `-out`   | `public`                          | dossier de sortie                         |
| `-clean` | `false`                           | vide la sortie avant génération           |
| `-serve` | `false`                           | sert le résultat en HTTP après génération |
| `-addr`  | `:8080`                           | adresse du serveur de prévisualisation    |
| `-title` | `Comprendre et maîtriser Go 1.26` | titre du livre                            |
| `-v`     | `false`                           | logs verbeux (niveau debug)               |

## Structure

```
tools/site/
  main.go                  CLI + flags + //go:embed assets
  internal/
    model/                 types : Book, Part, Page, Heading, SearchDoc
    sommaire/              parse SOMMAIRE.md → arbre de navigation
    render/                Markdown → HTML (goldmark + chroma), réécriture liens
      gen_chroma/          générateur jetable de assets/css/chroma.css
    search/                construction de l'index de recherche (JSON)
    site/                  assemblage : gabarits, écriture public/, copie assets
  assets/                  embarqués (//go:embed) : css/, js/, templates/
```

## Coloration syntaxique

`assets/css/chroma.css` est **généré** depuis les thèmes `github` (clair) et
`github-dark` (sombre) de chroma. Pour le régénérer après une mise à jour de
chroma :

```bash
make chroma
```

## Publication

Un workflow GitHub Actions (`.github/workflows/site.yml`) construit le site à
chaque push sur `main` et le publie sur GitHub Pages. Le HTML n'est **jamais
commité** : `public/` est dans `.gitignore` et reconstruit à la volée.

### Prérequis : activer GitHub Pages (à faire une fois)

Le workflow échoue tant que Pages n'est pas activé avec la **source « GitHub
Actions »**. Le symptôme est une erreur 404 à l'étape `actions/deploy-pages` :

```
Error: Creating Pages deployment failed
Error: HttpError: Not Found
Ensure GitHub Pages has been enabled: https://github.com/<owner>/<repo>/settings/pages
```

Pour corriger, activer Pages une fois pour toutes :

- **Via l'interface** : _Settings → Pages → Build and deployment → Source =
  « GitHub Actions »_.
- **Via la CLI `gh`** :

  ```bash
  gh api --method POST repos/<owner>/<repo>/pages -f build_type=workflow
  # puis relancer le déploiement échoué :
  gh run rerun <run-id>
  ```

### Note : avertissement « Node.js 20 is deprecated »

Les logs affichent un avertissement indiquant que les actions ciblant Node 20
(`actions/checkout`, `setup-go`, `upload-artifact`, `deploy-pages`) sont
forcées sur Node 24. C'est un simple *warning*, **sans impact** sur le
déploiement : rien à corriger. Il disparaîtra en bumpant les tags de ces
actions quand elles publieront des versions ciblant Node 24.
