package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/enumgen/internal/generator"
)

// writePkg crée un répertoire temporaire avec un unique fichier source.
func writePkg(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "src.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestGenerateIota(t *testing.T) {
	dir := writePkg(t, `package fruit

//enumgen:stringer trimprefix=Fruit
type Fruit int

const (
	FruitApple Fruit = iota
	FruitPear
	FruitPlum
)
`)
	out, err := generator.Generate(dir, "enumgen")
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	got := string(out)
	for _, want := range []string{
		"package fruit",
		"func (v Fruit) String() string",
		`FruitApple: "Apple"`,
		`FruitPlum:  "Plum"`,
		"NE PAS MODIFIER",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sortie ne contient pas %q\n--- sortie ---\n%s", want, got)
		}
	}
}

func TestGenerateExplicitNonContiguous(t *testing.T) {
	dir := writePkg(t, `package level

//enumgen:stringer
type Level int

const (
	Debug Level = 10
	Warn  Level = 30
	Error Level = 50
)
`)
	out, err := generator.Generate(dir, "enumgen")
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	got := string(out)
	// Sans trimprefix, les libellés sont les identifiants tels quels.
	for _, want := range []string{`Debug: "Debug"`, `Warn:  "Warn"`, `Error: "Error"`} {
		if !strings.Contains(got, want) {
			t.Errorf("manque %q dans :\n%s", want, got)
		}
	}
	// L'ordre de déclaration est préservé par tri sur la valeur entière.
	if i, j := strings.Index(got, "Debug:"), strings.Index(got, "Error:"); i > j {
		t.Error("les constantes devraient être triées par valeur croissante")
	}
}

func TestGenerateNoAnnotation(t *testing.T) {
	dir := writePkg(t, `package plain

type Color int

const (
	Red Color = iota
	Green
)
`)
	out, err := generator.Generate(dir, "enumgen")
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	if out != nil {
		t.Errorf("sans annotation, la sortie devrait être nil, obtenu :\n%s", out)
	}
}

func TestGenerateDuplicateValue(t *testing.T) {
	dir := writePkg(t, `package dup

//enumgen:stringer
type State int

const (
	On  State = 1
	Yes State = 1
)
`)
	if _, err := generator.Generate(dir, "enumgen"); err == nil {
		t.Fatal("deux constantes de même valeur devraient être rejetées")
	}
}

func TestGenerateMalformedValue(t *testing.T) {
	dir := writePkg(t, `package bad

//enumgen:stringer
type Tag int

const (
	A Tag = "oops"
)
`)
	_, err := generator.Generate(dir, "enumgen")
	if err == nil {
		t.Fatal("un littéral non entier devrait être rejeté")
	}
	// Le diagnostic doit pointer la portée du littéral (ligne:col-col, via ValueEnd).
	if !strings.Contains(err.Error(), "src.go:") {
		t.Errorf("le message d'erreur devrait localiser le littéral : %v", err)
	}
}

// TestGenerateMatchesExample garantit que le fichier example_enum.go versionné
// est bien le reflet exact de ce que produit le générateur — c'est le « golden ».
func TestGenerateMatchesExample(t *testing.T) {
	out, err := generator.Generate("../../example", "enumgen")
	if err != nil {
		t.Fatalf("Generate sur example : %v", err)
	}
	golden, err := os.ReadFile("../../example/example_enum.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(golden) {
		t.Errorf("example_enum.go est périmé ; relancer « go generate ./... » dans projets/6-codegen/example")
	}
}

func TestPackageName(t *testing.T) {
	src := []byte("// en-tête\n\npackage widget\n\nimport \"strconv\"\n")
	if got := generator.PackageName(src); got != "widget" {
		t.Errorf("PackageName = %q, voulu \"widget\"", got)
	}
}
