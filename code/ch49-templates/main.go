// Command ch49-templates illustre les moteurs text/template et html/template :
// le langage de gabarits (actions, range, pipelines, FuncMap), la composition par
// {{define}}/{{template}}, l'option missingkey, et l'échappement contextuel de
// html/template. Les identifiants sont en anglais, les commentaires en français.
package main

import (
	"fmt"
	htmltemplate "html/template"
	"strings"
	"text/template"
)

// Invoice décrit une facture à rendre dans un gabarit.
type Invoice struct {
	Customer string
	Paid     bool
	Lines    []Line
}

// Line est une ligne de facture ; le montant est en centimes pour éviter les
// flottants (voir Ch. 3 sur les pièges des nombres à virgule).
type Line struct {
	Label string
	Cents int64
}

// funcs regroupe les fonctions personnalisées exposées aux gabarits. Une clé de
// la FuncMap devient un mot du langage de template, utilisable en appel
// (`{{ upper .X }}`) ou en aval d'un pipeline (`{{ .X | upper }}`).
var funcs = template.FuncMap{
	"upper":   strings.ToUpper,
	"frMoney": frMoney,
	"total":   totalCents,
}

// frMoney formate un montant en centimes façon "12,50 €".
func frMoney(cents int64) string {
	return fmt.Sprintf("%d,%02d €", cents/100, cents%100)
}

// totalCents additionne les montants des lignes.
func totalCents(lines []Line) int64 {
	var sum int64
	for _, l := range lines {
		sum += l.Cents
	}
	return sum
}

// invoiceTmpl est compilé UNE fois au chargement du paquet, pas à chaque rendu.
// Must panique si le gabarit est syntaxiquement invalide : l'erreur est alors
// détectée au démarrage du programme, jamais au premier appel en production.
//
// Le `-}}` après `range` et `end` supprime le saut de ligne qui suivrait
// l'action, pour que la mise en page reste nette.
var invoiceTmpl = template.Must(template.New("invoice").Funcs(funcs).Parse(invoiceText))

const invoiceText = `Facture — {{ .Customer | upper }}
{{ range .Lines -}}
- {{ .Label }}: {{ .Cents | frMoney }}
{{ end -}}
Total: {{ total .Lines | frMoney }}
Statut: {{ if .Paid }}payée{{ else }}à régler{{ end }}
`

// render exécute invoiceTmpl vers un strings.Builder. On écrit dans un io.Writer
// (ici *strings.Builder) plutôt que de concaténer des chaînes : Execute streame
// sa sortie sans construire de valeur intermédiaire (voir Ch. 41).
func render(inv Invoice) (string, error) {
	var b strings.Builder
	if err := invoiceTmpl.Execute(&b, inv); err != nil {
		return "", err
	}
	return b.String(), nil
}

// menuTmpl associe deux gabarits : "item" (défini par {{define}}) et "menu"
// (le corps restant, nommé au New). {{template "item" .}} invoque le premier
// depuis le second en lui passant l'élément courant de la boucle.
var menuTmpl = template.Must(template.New("menu").Parse(menuText))

const menuText = `{{define "item"}}- {{.}}{{end}}Menu:
{{range .}}{{template "item" .}}
{{end -}}`

// renderMenu rend une liste à puces via la composition ci-dessus.
func renderMenu(items []string) (string, error) {
	var b strings.Builder
	if err := menuTmpl.Execute(&b, items); err != nil {
		return "", err
	}
	return b.String(), nil
}

// renderStrict rend un gabarit sur une map avec l'option missingkey=error :
// une clé absente devient une ERREUR au lieu du discret "<no value>" par défaut.
func renderStrict(text string, data map[string]any) (string, error) {
	t, err := template.New("strict").Option("missingkey=error").Parse(text)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

// pageTmpl utilise html/template (et non text/template) : l'API et le langage sont
// IDENTIQUES, mais html/template échappe automatiquement selon le CONTEXTE où
// atterrit chaque action. Ici la même valeur {{.Name}} apparaît deux fois — dans le
// corps HTML puis dans une URL d'attribut href — et sera échappée différemment.
var pageTmpl = htmltemplate.Must(htmltemplate.New("page").Parse(pageText))

const pageText = `<p>Bonjour {{.Name}}</p>` + "\n" +
	`<a href="/u/{{.Name}}">profil</a>`

// renderPage rend pageTmpl. Pour une entrée hostile comme "<script>alert(1)</script>",
// html/template produit des entités HTML dans le corps (&lt;script&gt;...) et un
// encodage d'URL dans l'attribut href : aucune des deux ne devient exécutable.
func renderPage(name string) (string, error) {
	var b strings.Builder
	if err := pageTmpl.Execute(&b, map[string]any{"Name": name}); err != nil {
		return "", err
	}
	return b.String(), nil
}

func main() {
	inv := Invoice{
		Customer: "café du coin",
		Lines: []Line{
			{Label: "Expresso", Cents: 150},
			{Label: "Croissant", Cents: 120},
		},
	}
	out, err := render(inv)
	if err != nil {
		panic(err)
	}
	fmt.Print(out)

	menu, err := renderMenu([]string{"Entrée", "Plat", "Dessert"})
	if err != nil {
		panic(err)
	}
	fmt.Println("\n" + menu)

	// missingkey=error : le rendu échoue proprement sur une clé absente.
	if _, err := renderStrict(`Bonjour {{.name}}`, map[string]any{}); err != nil {
		fmt.Println("erreur attendue:", err)
	}

	// html/template : échappement contextuel d'une entrée hostile.
	page, err := renderPage("<script>alert(1)</script>")
	if err != nil {
		panic(err)
	}
	fmt.Println("\n" + page)
}
