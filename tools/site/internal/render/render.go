// Package render convertit le Markdown du livre en HTML : configuration de
// goldmark (GFM + footnotes + ancres), coloration syntaxique via chroma,
// réécriture des liens internes .md → .html, et extraction des métadonnées
// (titre H1, ToC H2/H3, texte brut pour l'index de recherche).
package render

import (
	"bytes"
	"html/template"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"example.com/gobook-site/internal/model"
)

// Result regroupe les sorties du rendu d'une page.
type Result struct {
	HTML      template.HTML
	Title     string
	Headings  []model.Heading
	PlainText string
}

// Renderer encapsule un moteur goldmark configuré, réutilisable pour toutes les
// pages (la configuration est coûteuse, l'instance est sûre en réutilisation).
type Renderer struct {
	md goldmark.Markdown
}

// New construit un Renderer configuré pour le livre.
func New() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,      // tables, autolinks, strikethrough, task lists
			extension.Footnote, // notes de bas de page
			extension.DefinitionList,
			highlighting.NewHighlighting(
				// Deux thèmes (clair/sombre) sont gérés par CSS côté front ;
				// on émet des classes chroma plutôt que du style inline.
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // ancres automatiques sur les titres
			parser.WithASTTransformers(
				util.Prioritized(&linkRewriter{}, 100),
				util.Prioritized(&calloutTagger{}, 110),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // le contenu du livre est de confiance (emojis, HTML inline)
		),
	)
	return &Renderer{md: md}
}

// Render convertit la source Markdown d'une page en HTML et métadonnées.
func (r *Renderer) Render(src []byte) (*Result, error) {
	ctx := parser.NewContext()
	doc := r.md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))

	res := &Result{}
	res.Title = extractTitle(doc, src)
	res.Headings = extractHeadings(doc, src)
	res.PlainText = extractPlainText(doc, src)

	var buf bytes.Buffer
	if err := r.md.Renderer().Render(&buf, src, doc); err != nil {
		return nil, err
	}
	res.HTML = template.HTML(buf.Bytes())
	return res, nil
}

// extractTitle renvoie le texte du premier titre H1.
func extractTitle(doc ast.Node, src []byte) string {
	var title string
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok && h.Level == 1 && title == "" {
			title = string(h.Text(src))
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return title
}

// extractHeadings collecte les titres H2/H3 avec leurs ancres pour la ToC locale.
func extractHeadings(doc ast.Node, src []byte) []model.Heading {
	var hs []model.Heading
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok || (h.Level != 2 && h.Level != 3) {
			return ast.WalkContinue, nil
		}
		id, _ := h.AttributeString("id")
		hs = append(hs, model.Heading{
			Level: h.Level,
			Text:  string(h.Text(src)),
			ID:    attrString(id),
		})
		return ast.WalkContinue, nil
	})
	return hs
}

// extractPlainText concatène le texte des nœuds (titres, paragraphes, listes et
// blocs de code inclus : chercher `io.Copy` doit fonctionner).
func extractPlainText(doc ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case ast.KindText:
			b.Write(n.(*ast.Text).Segment.Value(src))
			b.WriteByte(' ')
		case ast.KindString:
			b.Write(n.(*ast.String).Value)
			b.WriteByte(' ')
		case ast.KindCodeSpan:
			b.Write(n.Text(src))
			b.WriteByte(' ')
		case ast.KindFencedCodeBlock, ast.KindCodeBlock:
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				b.Write(seg.Value(src))
				b.WriteByte(' ')
			}
		}
		return ast.WalkContinue, nil
	})
	return strings.Join(strings.Fields(b.String()), " ")
}

func attrString(v any) string {
	switch s := v.(type) {
	case []byte:
		return string(s)
	case string:
		return s
	default:
		return ""
	}
}

// linkRewriter est un transformer AST qui réécrit les liens internes terminant
// en .md (avant une éventuelle ancre) vers .html.
type linkRewriter struct{}

func (t *linkRewriter) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}
		link.Destination = rewriteDest(link.Destination)
		return ast.WalkContinue, nil
	})
}

// rewriteDest transforme « foo.md » et « foo.md#bar » en « foo.html »/«foo.html#bar ».
// Les liens externes (avec ://) et les liens vers dossiers sont laissés intacts.
func rewriteDest(dest []byte) []byte {
	s := string(dest)
	if strings.Contains(s, "://") {
		return dest
	}
	path, anchor := s, ""
	if i := strings.IndexByte(s, '#'); i >= 0 {
		path, anchor = s[:i], s[i:]
	}
	if !strings.HasSuffix(path, ".md") {
		return dest
	}
	return []byte(strings.TrimSuffix(path, ".md") + ".html" + anchor)
}

// Marqueurs emoji du livre → classe de callout appliquée au blockquote.
var calloutClass = map[string]string{
	"🆕": "callout--new",
	"⚠": "callout--warn", // ⚠️ se décompose en U+26A0 (+ VS16)
	"💡": "callout--tip",
	"🔁": "callout--ref",
	"⚡": "callout--perf",
	"🧪": "callout--test",
	"📌": "callout--note",
}

// calloutTagger repère le premier emoji d'un blockquote et pose une classe CSS
// sur le nœud (rendue en attribut class grâce à WithUnsafe + AST attributes).
type calloutTagger struct{}

func (t *calloutTagger) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	src := reader.Source()
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		bq, ok := n.(*ast.Blockquote)
		if !ok {
			return ast.WalkContinue, nil
		}
		lead := leadingRunes(bq.Text(src))
		for prefix, class := range calloutClass {
			if strings.HasPrefix(lead, prefix) {
				bq.SetAttributeString("class", []byte("callout "+class))
				break
			}
		}
		return ast.WalkSkipChildren, nil
	})
}

// leadingRunes renvoie les premiers octets utiles (sans espaces) pour tester un
// préfixe emoji.
func leadingRunes(b []byte) string {
	return strings.TrimLeft(string(b), " \t")
}
