// Package site assemble le livre rendu en un site statique : exécution des
// gabarits html/template, écriture des pages sous le dossier de sortie, copie des
// assets embarqués et écriture de l'index de recherche.
package site

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"example.com/gobook-site/internal/model"
	"example.com/gobook-site/internal/render"
	"example.com/gobook-site/internal/search"
)

// Builder pilote la génération complète du site.
type Builder struct {
	SrcDir   string // racine du livre (où sont chapitres/, annexes/)
	OutDir   string // dossier de sortie (public/)
	Assets   fs.FS  // assets embarqués (css/, js/, templates/)
	Log      *slog.Logger
	renderer *render.Renderer
	tmpl     *template.Template
}

// New construit un Builder prêt à l'emploi. assets doit contenir css/, js/ et
// templates/ (typiquement un sous-FS de l'embed.FS).
func New(srcDir, outDir string, assets fs.FS, logger *slog.Logger) (*Builder, error) {
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(assets, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("chargement des gabarits : %w", err)
	}
	return &Builder{
		SrcDir:   srcDir,
		OutDir:   outDir,
		Assets:   assets,
		Log:      logger,
		renderer: render.New(),
		tmpl:     tmpl,
	}, nil
}

// Build effectue le rendu, l'assemblage et l'écriture du site complet.
func (b *Builder) Build(book *model.Book) error {
	if err := os.MkdirAll(b.OutDir, 0o755); err != nil {
		return err
	}

	// 1. Rendu de chaque page (Markdown → HTML + métadonnées).
	var broken []string
	for _, p := range book.Pages {
		if err := b.renderPage(p); err != nil {
			return fmt.Errorf("rendu de %s : %w", p.SrcPath, err)
		}
	}

	// 2. Écriture des pages assemblées.
	for _, p := range book.Pages {
		if err := b.writePage(book, p); err != nil {
			return fmt.Errorf("écriture de %s : %w", p.OutPath, err)
		}
	}

	// 3. Pages spéciales : accueil (sommaire) et recherche.
	if err := b.writeTemplate("index.html.tmpl", "index.html", map[string]any{"Book": book}); err != nil {
		return err
	}
	if err := b.writeTemplate("search.html.tmpl", "search.html", map[string]any{"Book": book}); err != nil {
		return err
	}

	// 4. Index de recherche.
	if err := b.writeSearchIndex(book); err != nil {
		return err
	}

	// 5. Copie des assets statiques (css/, js/).
	if err := b.copyAssets(); err != nil {
		return err
	}

	// 6. Vérification des liens internes cassés (avertissements).
	broken = b.checkLinks(book)
	for _, w := range broken {
		b.Log.Warn("lien interne cassé", "ref", w)
	}

	b.Log.Info("site généré",
		"pages", len(book.Pages),
		"sortie", b.OutDir,
		"liens_casses", len(broken),
	)
	return nil
}

func (b *Builder) renderPage(p *model.Page) error {
	src, err := os.ReadFile(filepath.Join(b.SrcDir, filepath.FromSlash(p.SrcPath)))
	if err != nil {
		return err
	}
	res, err := b.renderer.Render(src)
	if err != nil {
		return err
	}
	p.HTML = res.HTML
	if res.Title != "" {
		p.Title = res.Title // le H1 fait foi sur le titre provisoire du SOMMAIRE
	}
	p.Headings = res.Headings
	p.PlainText = res.PlainText
	return nil
}

func (b *Builder) writePage(book *model.Book, p *model.Page) error {
	data := map[string]any{
		"Book":    book,
		"Page":    p,
		"BaseURL": relativeToRoot(p.OutPath),
	}
	return b.writeTemplate("page.html.tmpl", p.OutPath, data)
}

// writeTemplate exécute un gabarit nommé et écrit le résultat sous OutDir/rel.
func (b *Builder) writeTemplate(name, rel string, data any) error {
	// La donnée de page spéciale n'a pas de BaseURL : on la calcule ici aussi.
	if m, ok := data.(map[string]any); ok {
		if _, has := m["BaseURL"]; !has {
			m["BaseURL"] = relativeToRoot(rel)
		}
	}
	var buf bytes.Buffer
	if err := b.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}
	return b.writeFile(rel, buf.Bytes())
}

func (b *Builder) writeSearchIndex(book *model.Book) error {
	docs := search.Build(book)
	data, err := search.Marshal(docs)
	if err != nil {
		return err
	}
	b.Log.Debug("index de recherche", "entrees", len(docs), "octets", len(data))
	if err := b.writeFile("search-index.json", data); err != nil {
		return err
	}
	// Variante JS chargée par balise <script> : fonctionne aussi en file://,
	// où fetch() est bloqué par la politique de sécurité du navigateur.
	js := append([]byte("window.__SEARCH_INDEX__="), data...)
	js = append(js, ';')
	return b.writeFile("search-index.js", js)
}

// copyAssets recopie css/ et js/ embarqués vers OutDir/assets/.
func (b *Builder) copyAssets() error {
	for _, dir := range []string{"css", "js"} {
		err := fs.WalkDir(b.Assets, dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := fs.ReadFile(b.Assets, p)
			if err != nil {
				return err
			}
			return b.writeFile(path.Join("assets", p), data)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// writeFile écrit data sous OutDir/rel en garantissant que la cible reste
// confinée dans OutDir (défense en profondeur contre les chemins malveillants).
func (b *Builder) writeFile(rel string, data []byte) error {
	clean := filepath.FromSlash(path.Clean("/" + rel)) // absolutise puis nettoie
	dst := filepath.Join(b.OutDir, clean)
	absOut, err := filepath.Abs(b.OutDir)
	if err != nil {
		return err
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	if absDst != absOut && !strings.HasPrefix(absDst, absOut+string(filepath.Separator)) {
		return fmt.Errorf("chemin de sortie hors du dossier cible : %s", rel)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// checkLinks signale les liens internes (.html) pointant vers une page absente
// du livre. On vérifie contre l'ensemble des pages connues.
func (b *Builder) checkLinks(book *model.Book) []string {
	known := make(map[string]bool, len(book.Pages))
	for _, p := range book.Pages {
		known[p.OutPath] = true
	}
	var broken []string
	seen := make(map[string]bool)
	for _, p := range book.Pages {
		base := path.Dir(p.OutPath)
		for _, dest := range extractHrefs(string(p.HTML)) {
			if !strings.HasSuffix(stripAnchor(dest), ".html") {
				continue
			}
			if strings.Contains(dest, "://") {
				continue
			}
			target := path.Clean(path.Join(base, stripAnchor(dest)))
			if !known[target] {
				key := p.OutPath + " -> " + dest
				if !seen[key] {
					seen[key] = true
					broken = append(broken, key)
				}
			}
		}
	}
	sort.Strings(broken)
	return broken
}

func stripAnchor(s string) string {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i]
	}
	return s
}

// extractHrefs récupère grossièrement les valeurs href="…" d'un fragment HTML.
func extractHrefs(htmlStr string) []string {
	var out []string
	const marker = `href="`
	for {
		i := strings.Index(htmlStr, marker)
		if i < 0 {
			break
		}
		htmlStr = htmlStr[i+len(marker):]
		j := strings.IndexByte(htmlStr, '"')
		if j < 0 {
			break
		}
		out = append(out, htmlStr[:j])
		htmlStr = htmlStr[j+1:]
	}
	return out
}

// relativeToRoot renvoie le préfixe « ../ » nécessaire pour remonter à la racine
// du site depuis une page de profondeur donnée (ex: chapitres/x.html → "../").
func relativeToRoot(outPath string) string {
	depth := strings.Count(path.Clean(outPath), "/")
	if depth == 0 {
		return ""
	}
	return strings.Repeat("../", depth)
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// rel construit un lien relatif à la racine du site (préfixe ../ adéquat).
		"rel": func(base, target string) string {
			return base + target
		},
		// navLabel raccourcit un titre pour la navigation : « Ch. 4 — Flux de
		// contrôle » devient « 4 — Flux de contrôle ».
		"navLabel": func(title string) string {
			return strings.TrimPrefix(title, "Ch. ")
		},
	}
}
