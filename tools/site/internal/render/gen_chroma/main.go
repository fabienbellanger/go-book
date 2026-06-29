//go:build ignore

// Générateur jetable : produit assets/css/chroma.css à partir des styles chroma
// github (clair) et github-dark (sombre), ce dernier scopé sous [data-theme="dark"].
//
//	go run ./internal/render/gen_chroma > assets/css/chroma.css
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

func main() {
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	fmt.Fprintln(out, "/* Coloration syntaxique chroma — généré par internal/render/gen_chroma. */")
	fmt.Fprintln(out, "/* Thème clair : github. Thème sombre : github-dark (scopé [data-theme=\"dark\"]). */")
	fmt.Fprintln(out)

	light := styles.Get("github")
	dark := styles.Get("github-dark")
	fmtter := chromahtml.New(chromahtml.WithClasses(true))

	var lb strings.Builder
	if err := fmtter.WriteCSS(&lb, light); err != nil {
		panic(err)
	}
	fmt.Fprintln(out, "/* --- clair (github) --- */")
	fmt.Fprint(out, lb.String())
	fmt.Fprintln(out)

	var db strings.Builder
	if err := fmtter.WriteCSS(&db, dark); err != nil {
		panic(err)
	}
	fmt.Fprintln(out, "/* --- sombre (github-dark) --- */")
	const scope = `[data-theme="dark"] `
	for _, line := range strings.Split(strings.TrimRight(db.String(), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			fmt.Fprintln(out, line)
			continue
		}
		// chroma écrit « /* Commentaire */ SELECTEUR { … } ». On insère le scope
		// juste avant le sélecteur (après le commentaire), sinon en tête de ligne.
		if i := strings.Index(line, "*/ "); i >= 0 {
			head := line[:i+len("*/ ")]
			rest := line[i+len("*/ "):]
			fmt.Fprintln(out, head+scope+rest)
		} else {
			fmt.Fprintln(out, scope+line)
		}
	}
}
