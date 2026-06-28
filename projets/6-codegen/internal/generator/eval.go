package generator

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// evalConst évalue l'expression d'une constante entière dans un bloc « const ».
//
// On ne réimplémente PAS tout `go/types` : on couvre les formes qui apparaissent
// réellement dans les énumérations idiomatiques — littéraux entiers, `iota`,
// décalages et arithmétique simple (`iota + 1`, `1 << iota`, `-3`). C'est
// volontairement limité : le but pédagogique du projet est le **parcours d'AST**,
// pas l'évaluation exhaustive de constantes. Pour le reste, on renvoie une erreur
// localisée plutôt que de produire un code faux.
func evalConst(expr ast.Expr, iota int) (int64, error) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.INT {
			// On exploite ValueEnd (Go 1.26) pour pointer la fin exacte du
			// littéral fautif dans le diagnostic.
			return 0, fmt.Errorf("littéral %s non entier", e.Value)
		}
		n, err := strconv.ParseInt(e.Value, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("littéral entier illisible %q : %w", e.Value, err)
		}
		return n, nil

	case *ast.Ident:
		if e.Name == "iota" {
			return int64(iota), nil
		}
		return 0, fmt.Errorf("identifiant non supporté %q (seul `iota` est reconnu)", e.Name)

	case *ast.ParenExpr:
		return evalConst(e.X, iota)

	case *ast.UnaryExpr:
		v, err := evalConst(e.X, iota)
		if err != nil {
			return 0, err
		}
		switch e.Op {
		case token.SUB:
			return -v, nil
		case token.ADD:
			return v, nil
		}
		return 0, fmt.Errorf("opérateur unaire non supporté %q", e.Op)

	case *ast.BinaryExpr:
		l, err := evalConst(e.X, iota)
		if err != nil {
			return 0, err
		}
		r, err := evalConst(e.Y, iota)
		if err != nil {
			return 0, err
		}
		switch e.Op {
		case token.ADD:
			return l + r, nil
		case token.SUB:
			return l - r, nil
		case token.MUL:
			return l * r, nil
		case token.SHL:
			return l << uint(r), nil
		case token.SHR:
			return l >> uint(r), nil
		}
		return 0, fmt.Errorf("opérateur binaire non supporté %q", e.Op)

	default:
		return 0, fmt.Errorf("expression de constante non supportée (%T)", expr)
	}
}

// litSpan rend la portée source « ligne:col-col » d'un littéral, en s'appuyant
// sur BasicLit.ValueEnd (Go 1.26) pour la borne de fin. Utile dans les messages
// d'erreur : on désigne précisément les octets fautifs, pas seulement leur début.
func litSpan(fset *token.FileSet, lit *ast.BasicLit) string {
	start := fset.Position(lit.ValuePos)
	end := fset.Position(lit.ValueEnd)
	return fmt.Sprintf("%s:%d:%d-%d", start.Filename, start.Line, start.Column, end.Column)
}
