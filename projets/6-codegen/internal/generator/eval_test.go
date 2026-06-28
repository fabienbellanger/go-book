package generator

import (
	"go/parser"
	"testing"
)

// parseExpr est un raccourci pour évaluer une expression de constante isolée.
func mustEval(t *testing.T, src string, iota int) int64 {
	t.Helper()
	expr, err := parser.ParseExpr(src)
	if err != nil {
		t.Fatalf("ParseExpr(%q) : %v", src, err)
	}
	v, err := evalConst(expr, iota)
	if err != nil {
		t.Fatalf("evalConst(%q, iota=%d) : %v", src, iota, err)
	}
	return v
}

func TestEvalConst(t *testing.T) {
	cases := []struct {
		src  string
		iota int
		want int64
	}{
		{"0", 0, 0},
		{"42", 0, 42},
		{"0x1f", 0, 31},
		{"iota", 3, 3},
		{"iota + 1", 2, 3},
		{"1 << iota", 4, 16},
		{"-5", 0, -5},
		{"(iota + 1) * 10", 2, 30},
		{"100 >> 2", 0, 25},
	}
	for _, c := range cases {
		if got := mustEval(t, c.src, c.iota); got != c.want {
			t.Errorf("evalConst(%q, iota=%d) = %d, voulu %d", c.src, c.iota, got, c.want)
		}
	}
}

func TestEvalConstErrors(t *testing.T) {
	for _, src := range []string{`"texte"`, "3.14", "foo", "iota / 2 % 1"} {
		expr, err := parser.ParseExpr(src)
		if err != nil {
			t.Fatalf("ParseExpr(%q) : %v", src, err)
		}
		if _, err := evalConst(expr, 0); err == nil {
			t.Errorf("evalConst(%q) aurait dû échouer", src)
		}
	}
}
