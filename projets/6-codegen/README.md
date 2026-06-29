# Projet 6 — Générateur de code : `enumgen`

> **Objectif** — Écrire un **générateur de code** Go, dans l'esprit de `stringer`,
> avec la seule bibliothèque standard : on **analyse** le source d'un paquet
> (`go/parser` → AST), on **repère** les types d'énumération annotés, on **évalue**
> leurs constantes, et on **émet** une méthode `String()` via `text/template`. Le
> tout s'intègre à `go generate`. On y voit le triptyque de la méta-programmation
> Go — **parser → parcourir l'AST → générer** — et deux nouveautés **Go 1.26** :
> `ast.ParseDirective` et `BasicLit.ValueEnd`.
>
> **Réinvestit** — [Ch. 8 Structures](../../chapitres/08-structs-methodes.md),
> [Ch. 12 Packages](../../chapitres/12-packages-modules.md),
> [Ch. 13 Tests](../../chapitres/13-tests-outillage.md),
> [Ch. 34 Réflexion](../../chapitres/34-reflexion.md).

---

## 1. Cahier des charges

`enumgen` génère la méthode `String()` des types entiers d'énumération **annotés**.
On annote le type avec une **directive**, puis on déclenche la génération avec
`go generate` :

```go
//enumgen:stringer trimprefix=Color
type Color int

const (
	ColorRed Color = iota
	ColorGreen
	ColorBlue
)

//go:generate enumgen
```

```bash
$ go generate ./...
enumgen : example_enum.go écrit
$ # ColorRed.String() == "Red"  (le préfixe « Color » a été rogné)
```

Contraintes :

- **Bibliothèque standard seule** : `go/parser`, `go/ast`, `go/token`,
  `go/format`, `text/template`.
- **Robustesse** : un littéral non entier, une valeur dupliquée ⇒ erreur
  **localisée** (fichier:ligne:colonne), jamais de code faux émis.
- **Déterminisme** : même entrée ⇒ même sortie, octet pour octet (indispensable
  pour versionner le fichier généré et le vérifier en CI).

---

## 2. Le pipeline d'un générateur

```
  color.go                                              example_enum.go
     │                                                        ▲
     │ go/parser.ParseFile (avec ParseComments)               │ go/format.Source
     ▼                                                        │
   *ast.File ──┐                                       []byte (gofmt'é)
               │ parcours des Decls                           ▲
               ▼                                               │
   types annotés //enumgen:stringer  ───▶  modèle  ───▶  text/template.Execute
   + constantes évaluées (iota…)         (File{Enums})
```

Quatre étapes, une par fonction dans `internal/generator` :

1. **Parser** le répertoire (`parseDir`) — on garde les commentaires, car les
   **directives en sont**.
2. **Repérer** les types annotés (`findAnnotatedTypes`) — via `ast.ParseDirective`.
3. **Collecter** les constantes de chaque type (`collectEnums`) — avec gestion
   d'`iota` et évaluation des valeurs.
4. **Rendre** le gabarit puis **reformater** (`go/format.Source`).

---

## 3. Repérer une directive — `ast.ParseDirective` (🆕 1.26)

Une **directive** Go a la forme `//tool:name args` (sans espace après `//`).
Historiquement, on la détectait à coups de `strings.HasPrefix`. Go 1.26 fournit
un décodeur officiel :

```go
d, ok := ast.ParseDirective(c.Slash, c.Text)
// pour "//enumgen:stringer trimprefix=Color" :
//   ok == true
//   d.Tool == "enumgen"   d.Name == "stringer"   d.Args == "trimprefix=Color"
if ok && d.Tool == "enumgen" && d.Name == "stringer" {
	prefix := parseTrimPrefix(d.Args)   // "Color"
}
```

> 💡 **Pourquoi c'est mieux** : `ParseDirective` distingue une **vraie directive**
> (`//go:generate`, `//enumgen:stringer`) d'un commentaire ordinaire qui
> commencerait par `//` — la règle « pas d'espace, un `:` » est subtile, autant
> la déléguer à la bibliothèque standard.

On lit la doc d'un type via `TypeSpec.Doc` (bloc `type (...)`) **ou** `GenDecl.Doc`
(déclaration simple `type X int`) — il faut regarder les deux.

---

## 4. Évaluer les constantes (`iota` & co.)

Pour générer la map `valeur → libellé`, il faut **la valeur entière** de chaque
constante. Dans un bloc `const`, deux subtilités d'`iota` :

- `iota` vaut l'**indice de la `ValueSpec`** dans le bloc ;
- une spec **sans valeur** réutilise l'expression de la précédente (`ColorGreen`
  hérite de `= iota`).

On n'embarque **pas** tout `go/types` : un petit évaluateur (`evalConst`) couvre
les formes réelles des énumérations — littéraux entiers, `iota`, `iota + 1`,
`1 << iota`, `-3`, parenthèses :

```go
func evalConst(expr ast.Expr, iota int) (int64, error) {
	switch e := expr.(type) {
	case *ast.BasicLit:    // 42, 0x1f
	case *ast.Ident:       // iota
	case *ast.BinaryExpr:  // iota + 1, 1 << iota
	case *ast.UnaryExpr:   // -3
	case *ast.ParenExpr:   // (…)
	}
}
```

> ⚠️ **Choix de portée** : couvrir 100 % de la sémantique des constantes Go
> demanderait `go/types` (et la résolution des imports). Le **but pédagogique** ici
> est le **parcours d'AST**, pas l'évaluateur ; pour le reste, on **échoue
> proprement** plutôt que d'émettre du faux. Un vrai `stringer` s'appuie, lui, sur
> `go/packages` + `go/types`.

---

## 5. Diagnostics localisés — `BasicLit.ValueEnd` (🆕 1.26)

Quand une valeur est invalide (littéral non entier), on veut **désigner les
octets fautifs**. `BasicLit` expose désormais `ValueEnd` (position juste après le
littéral) en plus de `ValuePos` :

```go
func litSpan(fset *token.FileSet, lit *ast.BasicLit) string {
	start := fset.Position(lit.ValuePos)
	end := fset.Position(lit.ValueEnd)   // 🆕 1.26
	return fmt.Sprintf("%s:%d:%d-%d", start.Filename, start.Line, start.Column, end.Column)
}
// => "src.go:6:12-18"  (la portée exacte du littéral, pas seulement son début)
```

---

## 6. Émettre du code formaté

On rend un gabarit `text/template`, puis on passe **toujours** par
`go/format.Source` — c'est `gofmt` en bibliothèque. Avantage : le gabarit n'a pas
à soigner l'alignement, et la sortie est **canonique** (donc stable, donc
diffable en CI).

```go
formatted, err := format.Source(buf.Bytes())
```

L'en-tête `// Code généré par enumgen ; NE PAS MODIFIER.` suit la **convention**
reconnue par l'outillage Go (`go test` ignore ces fichiers pour la couverture,
les revues les replient).

---

## 7. Intégration `go generate`

```go
//go:generate enumgen
```

`go generate ./...` exécute la commande **dans le répertoire du fichier** ; c'est
pourquoi `enumgen` analyse `.` par défaut et écrit `<paquet>_enum.go` à côté.
La commande doit être **dans le `PATH`** :

```bash
make install        # go install . → $GOBIN/enumgen
go generate ./...
```

> 📌 `go generate` **ne lance jamais** la génération tout seul : ni `go build`, ni
> `go test` ne l'exécutent. C'est un geste **explicite** du développeur, et le
> fichier généré est **versionné**.

---

## 8. Tests

```bash
cd projets/6-codegen
go test -race ./...
```

- **`generator`** — génération à partir d'`iota`, de valeurs explicites non
  contiguës, rejet des valeurs **dupliquées** et des littéraux **non entiers**
  (diagnostic localisé), cas « aucune annotation ⇒ sortie nil ».
- **`eval`** — l'évaluateur de constantes (`iota`, décalages, arithmétique,
  erreurs).
- **`cli`** — `Run` de bout en bout : `-n` (dry-run, n'écrit rien), écriture du
  fichier, `-version`, drapeau inconnu ⇒ code 2.
- **Golden** — `TestGenerateMatchesExample` vérifie que `example/example_enum.go`
  **versionné** est l'exact reflet du générateur (sinon : relancer `go generate`).
- **`example`** — `Example…` testables qui exécutent les `String()` **générés**.

---

## 9. Build & cross-compilation

```bash
make install                  # indispensable pour « go generate »
make generate                 # go generate ./...
make build                    # bin/enumgen
make dist                     # binaires statiques, 5 plateformes
```

Options : `-dir`, `-out`, `-n` (dry-run), `-version`.

---

## 10. Points de vigilance

- **`ParseComments` est obligatoire** : sans lui, les directives (qui sont des
  commentaires) n'apparaissent pas dans l'AST.
- **Toujours `go/format.Source`** sur la sortie : un gabarit produit du texte
  approximatif ; le code versionné doit être canonique et déterministe.
- **Échouer plutôt que deviner** : valeur dupliquée, littéral non entier,
  expression non supportée ⇒ erreur claire et **localisée** (`ValueEnd`), pas de
  `String()` faux.
- **Ne pas relire sa propre sortie** : on **exclut** `*_enum.go` (et `*_test.go`)
  à l'analyse, sinon la deuxième génération verrait les méthodes déjà émises.
- **`go generate` est manuel** : le fichier généré est committé ; la CI le
  **régénère et compare** (le test golden) pour détecter tout oubli.

---

## 11. Pour aller plus loin

- **`go/types`** : évaluer n'importe quelle expression constante (et vérifier que
  le type sous-jacent est bien entier) via `go/packages` + `types.Info`.
- **Autres générateurs** : `//enumgen:accessors` (getters/setters), `//enumgen:json`
  (`MarshalJSON` par libellé), une variante `Parse<Type>(string)`.
- **`golang.org/x/tools/go/packages`** : charger un paquet avec ses dépendances
  résolues, comme le font `stringer` et la plupart des outils réels.

---

## 📌 À retenir

- Un générateur Go suit toujours le même pipeline : **parser → parcourir l'AST →
  rendre un gabarit → reformater** (`go/format`).
- **`ast.ParseDirective`** (1.26) décode proprement les directives `//tool:name
args` ; **`BasicLit.ValueEnd`** (1.26) borne un littéral pour des diagnostics
  précis.
- **`iota`** s'évalue par l'indice de la spec, et une spec sans valeur **hérite**
  de l'expression précédente.
- Le code généré est **versionné** et **déterministe** ; un **test golden** en CI
  garantit qu'il n'est jamais périmé.
- **`go generate` est explicite** : ni `build` ni `test` ne le déclenchent.
