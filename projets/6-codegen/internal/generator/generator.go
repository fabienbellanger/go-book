// Package generator implémente le cœur d'enumgen : analyser un paquet Go,
// repérer les types « enum » annotés, et produire pour chacun une méthode
// String() à partir de leurs constantes.
//
// Le pipeline est classique pour un outil de méta-programmation Go :
//
//	source .go ──parser──▶ AST ──parcours──▶ modèle ──text/template──▶ source .go ──go/format──▶ fichier
//
// On reste 100 % bibliothèque standard : go/parser, go/ast, go/token,
// go/format, text/template.
package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// Directive reconnue sur un type, par exemple :
//
//	//enumgen:stringer trimprefix=Color
//	type Color int
//
// `go/ast.ParseDirective` (Go 1.26) décompose la ligne en {Tool, Name, Args}
// sans bricolage de chaînes : ici Tool="enumgen", Name="stringer".
const (
	directiveTool = "enumgen"
	directiveName = "stringer"
)

// enumValue est une constante d'une énumération : son identifiant Go et le
// libellé qu'en rendra String() (après éventuel rognage de préfixe).
type enumValue struct {
	Ident string // nom de la constante, p. ex. ColorRed
	Label string // libellé rendu, p. ex. "Red" si trimprefix=Color
}

// enumType regroupe tout ce qu'il faut pour générer la méthode d'un type.
type enumType struct {
	Name   string      // nom du type, p. ex. Color
	Values []enumValue // constantes, dans l'ordre de déclaration
}

// File est le modèle complet passé au gabarit : un paquet et ses énumérations.
type File struct {
	Package string
	Command string // ligne de commande, pour l'en-tête « Code généré »
	Enums   []enumType
}

// Generate analyse le répertoire dir et renvoie le source Go des méthodes
// String() pour tous les types annotés. Le résultat est déjà formaté
// (go/format) ; il est vide (nil) si aucun type n'est annoté.
func Generate(dir, command string) ([]byte, error) {
	fset := token.NewFileSet()

	pkgName, files, err := parseDir(fset, dir)
	if err != nil {
		return nil, err
	}

	// 1. Repérer les types annotés //enumgen:stringer et leur éventuel trimprefix.
	annotated, err := findAnnotatedTypes(files)
	if err != nil {
		return nil, err
	}
	if len(annotated) == 0 {
		return nil, nil
	}

	// 2. Collecter les constantes de chaque type annoté.
	enums, err := collectEnums(fset, files, annotated)
	if err != nil {
		return nil, err
	}

	// 3. Rendre le gabarit, puis reformater avec go/format (gofmt programmatique).
	model := File{Package: pkgName, Command: command, Enums: enums}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, model); err != nil {
		return nil, fmt.Errorf("exécution du gabarit : %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatage du code généré : %w\n--- source brut ---\n%s", err, buf.String())
	}
	return formatted, nil
}

// parseDir lit tous les .go (hors tests et hors fichiers générés) d'un
// répertoire et renvoie le nom du paquet et les AST de fichiers.
func parseDir(fset *token.FileSet, dir string) (string, []*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, fmt.Errorf("lecture du répertoire %s : %w", dir, err)
	}

	var files []*ast.File
	var pkgName string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_enum.go") {
			continue // on ignore les tests et notre propre sortie
		}
		path := filepath.Join(dir, name)
		// On a besoin des commentaires (les directives en sont) : ParseComments.
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return "", nil, fmt.Errorf("analyse de %s : %w", path, err)
		}
		if pkgName == "" {
			pkgName = f.Name.Name
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return "", nil, fmt.Errorf("aucun fichier .go à analyser dans %s", dir)
	}
	return pkgName, files, nil
}

// findAnnotatedTypes parcourt les déclarations de types et renvoie, pour chaque
// type portant //enumgen:stringer, le préfixe à rogner (trimprefix=…, ou "").
func findAnnotatedTypes(files []*ast.File) (map[string]string, error) {
	out := map[string]string{}
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				// La doc peut être portée par le TypeSpec (bloc « type (...) »)
				// ou par le GenDecl (déclaration simple « type X int »).
				doc := ts.Doc
				if doc == nil {
					doc = gd.Doc
				}
				args, ok, err := stringerDirective(doc)
				if err != nil {
					return nil, fmt.Errorf("type %s : %w", ts.Name.Name, err)
				}
				if ok {
					out[ts.Name.Name] = parseTrimPrefix(args)
				}
			}
		}
	}
	return out, nil
}

// stringerDirective cherche //enumgen:stringer dans un groupe de commentaires
// via ast.ParseDirective (Go 1.26) et renvoie ses arguments bruts.
func stringerDirective(doc *ast.CommentGroup) (args string, found bool, err error) {
	if doc == nil {
		return "", false, nil
	}
	for _, c := range doc.List {
		d, ok := ast.ParseDirective(c.Slash, c.Text)
		if !ok {
			continue // commentaire ordinaire, pas une directive //tool:name
		}
		if d.Tool == directiveTool && d.Name == directiveName {
			return d.Args, true, nil
		}
	}
	return "", false, nil
}

// parseTrimPrefix extrait la valeur de « trimprefix=… » des arguments d'une
// directive (le seul argument supporté). Les autres sont ignorés silencieusement.
func parseTrimPrefix(args string) string {
	for field := range strings.FieldsSeq(args) {
		if v, ok := strings.CutPrefix(field, "trimprefix="); ok {
			return v
		}
	}
	return ""
}

// PackageName extrait le nom du paquet d'un source Go déjà formaté (la clause
// « package … »). enumgen s'en sert pour nommer le fichier de sortie par défaut.
func PackageName(src []byte) string {
	for line := range strings.SplitSeq(string(src), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "package "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return "generated"
}

// collectEnums parcourt les blocs « const » et rattache chaque constante typée à
// son enum annotée. La valeur entière de chaque constante est évaluée (evalConst)
// pour trier les constantes et détecter les doublons de valeur.
func collectEnums(fset *token.FileSet, files []*ast.File, annotated map[string]string) ([]enumType, error) {
	// type -> liste de (valeur, constante)
	type valued struct {
		v   int64
		val enumValue
	}
	collected := map[string][]valued{}

	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			// Dans un bloc const, iota s'incrémente à chaque ValueSpec, et une
			// spec sans valeur réutilise l'expression de la précédente.
			var lastType string
			var lastValues []ast.Expr
			for iota, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)

				typeName := lastType
				if vs.Type != nil {
					if id, ok := vs.Type.(*ast.Ident); ok {
						typeName = id.Name
					} else {
						typeName = "" // type composite : pas une enum simple
					}
				}
				values := vs.Values
				if len(values) == 0 {
					values = lastValues // héritée de la spec précédente (iota implicite)
				}
				lastType, lastValues = typeName, values

				if _, want := annotated[typeName]; !want {
					continue
				}
				if len(values) == 0 {
					continue
				}
				n, err := evalConst(values[0], iota)
				if err != nil {
					if lit, ok := values[0].(*ast.BasicLit); ok {
						return nil, fmt.Errorf("%s : %w", litSpan(fset, lit), err)
					}
					return nil, fmt.Errorf("%s : %w", fset.Position(values[0].Pos()), err)
				}
				// Une enum peut déclarer plusieurs noms ; on prend le premier.
				ident := vs.Names[0].Name
				if ident == "_" {
					continue // constante anonyme : ignorée
				}
				label := strings.TrimPrefix(ident, annotated[typeName])
				collected[typeName] = append(collected[typeName], valued{n, enumValue{Ident: ident, Label: label}})
			}
		}
	}

	// Mise en forme déterministe : types triés, constantes triées par valeur,
	// doublons de valeur rejetés (String() serait ambigu).
	out := make([]enumType, 0, len(collected))
	for name, vals := range collected {
		sort.SliceStable(vals, func(i, j int) bool { return vals[i].v < vals[j].v })
		seen := map[int64]string{}
		et := enumType{Name: name}
		for _, vv := range vals {
			if prev, dup := seen[vv.v]; dup {
				return nil, fmt.Errorf("type %s : constantes %s et %s partagent la valeur %d", name, prev, vv.val.Ident, vv.v)
			}
			seen[vv.v] = vv.val.Ident
			et.Values = append(et.Values, vv.val)
		}
		out = append(out, et)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// tmpl est le gabarit du fichier généré. Une map id->libellé gère proprement les
// valeurs non contiguës ou négatives ; le repli affiche « Type(n) » pour une
// valeur hors énumération (comportement attendu d'un String() robuste).
var tmpl = template.Must(template.New("enum").Parse(`// Code généré par {{.Command}} ; NE PAS MODIFIER.

package {{.Package}}

import "strconv"
{{range .Enums}}
var _{{.Name}}_names = map[{{.Name}}]string{
{{- range .Values}}
	{{.Ident}}: {{printf "%q" .Label}},
{{- end}}
}

// String implémente fmt.Stringer pour {{.Name}}.
func (v {{.Name}}) String() string {
	if s, ok := _{{.Name}}_names[v]; ok {
		return s
	}
	return "{{.Name}}(" + strconv.FormatInt(int64(v), 10) + ")"
}
{{end}}`))
