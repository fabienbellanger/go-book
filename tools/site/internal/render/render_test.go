package render

import (
	"strings"
	"testing"
)

const sample = "# Titre principal\n\n" +
	"Intro avec un lien interne vers [le chapitre 9](09-interfaces.md) et une ancre [section](autre.md#sec).\n\n" +
	"## Première section\n\n" +
	"Du texte avec `io.Copy` en code.\n\n" +
	"```go\nfunc main() { println(\"x\") }\n```\n\n" +
	"### Sous-section\n\n" +
	"> 💡 Une astuce utile.\n\n" +
	"> ⚠️ Un piège à éviter.\n\n" +
	"## Deuxième section\n\nFin.\n"

func TestRender(t *testing.T) {
	r := New()
	res, err := r.Render([]byte(sample))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if res.Title != "Titre principal" {
		t.Errorf("Title = %q", res.Title)
	}

	html := string(res.HTML)

	// Réécriture des liens .md → .html (ancre préservée).
	if !strings.Contains(html, `href="09-interfaces.html"`) {
		t.Errorf("lien .md non réécrit en .html : %s", html)
	}
	if !strings.Contains(html, `href="autre.html#sec"`) {
		t.Errorf("ancre non préservée à la réécriture")
	}

	// Coloration : bloc <pre class="chroma">.
	if !strings.Contains(html, `class="chroma"`) {
		t.Errorf("bloc de code non colorisé (chroma absent)")
	}

	// Callouts emoji → classe CSS sur le blockquote.
	if !strings.Contains(html, "callout--tip") {
		t.Errorf("callout 💡 (tip) absent : %s", html)
	}
	if !strings.Contains(html, "callout--warn") {
		t.Errorf("callout ⚠️ (warn) absent")
	}

	// Texte brut : contient le code inline et le contenu des blocs.
	if !strings.Contains(res.PlainText, "io.Copy") {
		t.Errorf("PlainText sans le code inline io.Copy")
	}
	if !strings.Contains(res.PlainText, "func main") {
		t.Errorf("PlainText sans le contenu du bloc de code")
	}
}

func TestExtractHeadings(t *testing.T) {
	r := New()
	res, _ := r.Render([]byte(sample))
	if len(res.Headings) != 3 {
		t.Fatalf("headings = %d, attendu 3 (2×H2 + 1×H3)", len(res.Headings))
	}
	if res.Headings[0].Level != 2 || res.Headings[0].Text != "Première section" {
		t.Errorf("heading 0 = %+v", res.Headings[0])
	}
	if res.Headings[1].Level != 3 {
		t.Errorf("heading 1 devrait être H3, got %d", res.Headings[1].Level)
	}
	for _, h := range res.Headings {
		if h.ID == "" {
			t.Errorf("heading %q sans ancre", h.Text)
		}
	}
}

func TestRewriteDest(t *testing.T) {
	cases := map[string]string{
		"x.md":                "x.html",
		"x.md#a":              "x.html#a",
		"https://ex.com/y.md": "https://ex.com/y.md", // externe : inchangé
		"image.png":           "image.png",           // non-.md : inchangé
		"projets/1/":          "projets/1/",          // dossier : inchangé
	}
	for in, want := range cases {
		if got := string(rewriteDest([]byte(in))); got != want {
			t.Errorf("rewriteDest(%q) = %q, attendu %q", in, got, want)
		}
	}
}
