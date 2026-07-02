//go:build ignore

// Générateur jetable : produit assets/css/chroma.css à partir des styles chroma
// github (clair) et github-dark (sombre). Les DEUX thèmes sont scopés : le clair
// sous :root:not([data-theme="dark"]) (défaut, y compris sans attribut), le sombre
// sous [data-theme="dark"]. Le :root est essentiel : data-theme n'est posé que sur
// <html>, donc un simple :not([data-theme="dark"]) matcherait via <body> et laisserait
// fuiter le thème clair en mode sombre. Sans ce double scope, les règles claires
// (globales) écrasent en sombre les tokens que github-dark ne redéfinit pas (.nx, .p…),
// rendant identifiants et ponctuation illisibles sur fond sombre.
//
//	go run ./internal/render/gen_chroma > assets/css/chroma.css
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

func main() {
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	fmt.Fprintln(out, "/* Coloration syntaxique chroma — généré par internal/render/gen_chroma. */")
	fmt.Fprintln(out, `/* Clair : github (scopé :root:not([data-theme="dark"])). Sombre : github-dark (scopé [data-theme="dark"]). */`)
	fmt.Fprintln(out)

	fmtter := chromahtml.New(chromahtml.WithClasses(true))

	fmt.Fprintln(out, "/* --- clair (github) --- */")
	writeScoped(out, fmtter, styles.Get("github"), `:root:not([data-theme="dark"]) `)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "/* --- sombre (github-dark) --- */")
	writeScoped(out, fmtter, styles.Get("github-dark"), `[data-theme="dark"] `)
}

// writeScoped écrit le CSS d'un style chroma en préfixant chaque sélecteur par scope.
func writeScoped(out *bufio.Writer, fmtter *chromahtml.Formatter, style *chroma.Style, scope string) {
	var b strings.Builder
	if err := fmtter.WriteCSS(&b, style); err != nil {
		panic(err)
	}
	for line := range strings.SplitSeq(strings.TrimRight(b.String(), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			fmt.Fprintln(out, line)
			continue
		}
		// chroma écrit « /* Commentaire */ SELECTEUR { … } ». On insère le scope
		// juste avant le sélecteur (après le commentaire), sinon en tête de ligne.
		if i := strings.Index(line, "*/ "); i >= 0 {
			head := line[:i+len("*/ ")]
			rest := line[i+len("*/ "):]
			// Le fond de la boîte de code est piloté par le <pre> (var --code-bg),
			// cohérent avec le reste du site et distinct des citations. On retire donc
			// le background-color que chroma pose sur le conteneur (.chroma/.bg), sinon
			// il repeint la couleur de page par-dessus et la boîte devient invisible.
			if strings.Contains(head, "PreWrapper") || strings.Contains(head, "Background") {
				rest = stripBackground(rest)
			}
			fmt.Fprintln(out, head+scope+rest)
		} else {
			fmt.Fprintln(out, scope+line)
		}
	}
}

// stripBackground retire la déclaration « background-color: … ; » d'un bloc de règles CSS.
func stripBackground(s string) string {
	i := strings.Index(s, "background-color:")
	if i < 0 {
		return s
	}
	j := strings.Index(s[i:], ";")
	if j < 0 {
		return s
	}
	return strings.ReplaceAll(s[:i]+s[i+j+1:], "  ", " ")
}
