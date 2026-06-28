package store

import "testing"

// Le SQLStore n'est pas testé de bout en bout ici : il exigerait un driver
// enregistré (voir README). On valide en revanche la logique pure du
// découpage des migrations, qui ne dépend d'aucune base.
func TestSplitStatements(t *testing.T) {
	script := `-- commentaire d'en-tête
CREATE TABLE t (id INTEGER);

-- second bloc
CREATE INDEX idx ON t (id);
`
	got := splitStatements(script)
	if len(got) != 2 {
		t.Fatalf("got %d instructions, voulu 2 : %q", len(got), got)
	}
	if got[0] != "CREATE TABLE t (id INTEGER)" {
		t.Errorf("instruction[0] = %q", got[0])
	}
	if got[1] != "CREATE INDEX idx ON t (id)" {
		t.Errorf("instruction[1] = %q", got[1])
	}
}

func TestSplitStatementsEmpty(t *testing.T) {
	if got := splitStatements("  \n-- rien que des commentaires\n  "); len(got) != 0 {
		t.Errorf("got %q, voulu aucune instruction", got)
	}
}
