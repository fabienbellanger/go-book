# Annexe L — Solutions des exercices — Partie I

> **Objectif** — Fournir des **solutions compilables et testées** aux exercices
> « 🧪 À tester soi-même » des chapitres **fondamentaux (2 à 13)**. Chaque
> solution reformule l'énoncé, propose une implémentation idiomatique et explique
> le *pourquoi*. Tout le code vit dans
> [`code/annexe-L-solutions/`](../code/annexe-L-solutions/) et passe `go test`.
>
> Cette annexe couvre la **Partie I**. Les parties suivantes (mécanismes avancés,
> concurrence, runtime, internals, performance, stdlib) viendront l'étendre.

---

Beaucoup d'exercices demandent d'**observer un échec** (erreur de compilation,
troncature, aliasing). Ces cas ne peuvent pas « passer » dans un test : on donne
alors la **version correcte** et l'on explique le comportement fautif dans le
texte. Les solutions sont volontairement **courtes** : un exercice réussi tient
souvent en quelques lignes.

```bash
cd code
go test ./annexe-L-solutions/
go test -run xxx -fuzz=FuzzCh13Slugify -fuzztime=5s ./annexe-L-solutions/
```

---

## Chapitre 2 — [Structure d'un programme](../chapitres/02-structure-programme.md)

**Énoncé.** Ajouter une langue (`"de": "Hallo"`), observer qu'un symbole passé en
minuscule devient invisible hors de son paquet, et confirmer que `init()` tourne
avant `main`.

**Solution.** Une salutation multilingue par map, avec repli sur l'anglais
([`ch02.go`](../code/annexe-L-solutions/ch02.go)) :

```go
// code/annexe-L-solutions/ch02.go
var ch02Greetings = map[string]string{"fr": "Bonjour", "en": "Hello", "de": "Hallo"}

func ch02Greet(lang, name string) string {
	msg, ok := ch02Greetings[lang]
	if !ok {
		msg = ch02Greetings["en"] // repli
	}
	return msg + ", " + name + " !"
}
```

**Pourquoi.** L'export en Go est **lexical** : seule une **majuscule initiale**
rend un identifiant visible depuis un autre paquet. Renommer `Greet` en `greet`
casse donc la compilation de tout appelant externe — c'est une propriété du
compilateur, pas une convention. Quant à `init()`, il s'exécute **après** les
initialisations de variables du paquet et **avant** `main` ; plusieurs `init()`
sont autorisés et s'enchaînent dans l'ordre des fichiers.

## Chapitre 3 — [Variables, constantes & types de base](../chapitres/03-variables-constantes-types.md)

**Énoncé.** Observer la troncature de `int8(200)` sans garde, puis constater que
`var x int8 = 200` et `1<<70` dans un `iota` sont attrapés **à la compilation**.

**Solution.** Deux conversions, l'une brute, l'autre gardée
([`ch03.go`](../code/annexe-L-solutions/ch03.go)) :

```go
// code/annexe-L-solutions/ch03.go
func ch03ToInt8Unchecked(n int) int8 { return int8(n) } // 200 -> -56 (troncature)

func ch03ToInt8Checked(n int) (int8, error) {
	if n < -128 || n > 127 {
		return 0, fmt.Errorf("%d hors de la plage int8 [-128, 127]", n)
	}
	return int8(n), nil
}
```

**Pourquoi.** Une conversion `int8(200)` **tronque** sur 8 bits sans prévenir :
`200` devient `-56` (soit `200 - 256`). C'est un piège classique — d'où la
version gardée. En revanche, `var x int8 = 200` (littéral **constant** hors
plage) et `1<<70` (dépassement d'un type sous-jacent) sont refusés **dès la
compilation** : les constantes non typées sont vérifiées avant l'exécution.

## Chapitre 4 — [Flux de contrôle](../chapitres/04-flux-controle.md)

**Énoncé.** Réécrire `classify` avec un `switch score/10`, et montrer qu'un
`break` **étiqueté** quitte deux boucles imbriquées d'un coup.

**Solution** ([`ch04.go`](../code/annexe-L-solutions/ch04.go)) :

```go
// code/annexe-L-solutions/ch04.go
func ch04Classify(score int) string {
	switch score / 10 {
	case 10, 9:
		return "excellent"
	case 8, 7:
		return "bien"
	case 6, 5:
		return "passable"
	default:
		return "insuffisant"
	}
}
```

```go
search:
	for _, x := range xs {
		for _, y := range ys {
			if x+y == target {
				break search // quitte LES DEUX boucles
			}
		}
	}
```

**Pourquoi.** Le `switch` d'expression regroupe les dizaines équivalentes sans
`fallthrough` : chaque `case` **rompt** implicitement. Sur les boucles
imbriquées, un `break` nu ne quitte que la boucle **interne** — on continuerait
à chercher à tort. L'**étiquette** (`break search`) est le seul moyen de sortir
des deux d'un coup ; un `fallthrough` ajouté à `classify`, lui, ferait tomber en
cascade dans le `case` suivant.

## Chapitre 5 — [Fonctions](../chapitres/05-fonctions.md)

**Énoncé.** Ajouter une option `WithMaxConns(n)`, écrire `compose(f, g)`, et
expliquer pourquoi `sum(divmod(17, 5))` compile mais pas `q := divmod(17, 5)`.

**Solution** ([`ch05.go`](../code/annexe-L-solutions/ch05.go)) :

```go
// code/annexe-L-solutions/ch05.go
func ch05Compose(f, g func(int) int) func(int) int {
	return func(x int) int { return f(g(x)) } // x -> f(g(x))
}

func ch05WithMaxConns(n int) ch05Option { return func(s *ch05Server) { s.maxConns = n } }
```

**Pourquoi.** `compose` renvoie une **closure** qui capture `f` et `g` : les
fonctions sont des valeurs de première classe. `sum(divmod(17, 5))` compile car
les **deux** valeurs de retour de `divmod` alimentent directement les **deux**
paramètres de `sum` — Go autorise ce « déballage » uniquement quand l'appel est
le **seul** argument. À l'inverse, `q := divmod(17, 5)` échoue : on ne peut pas
affecter deux valeurs à une seule variable. Enfin, une nouvelle option comme
`WithMaxConns` n'impacte **aucun** appel existant : c'est l'atout du patron des
options fonctionnelles.

## Chapitre 6 — [Arrays & slices](../chapitres/06-arrays-slices.md)

**Énoncé.** Comprendre pourquoi `s[i:end:end]` évite l'aliasing dans `chunk`, et
écrire `removeAt` **sans fuite**.

**Solution** ([`ch06.go`](../code/annexe-L-solutions/ch06.go)) :

```go
// code/annexe-L-solutions/ch06.go
out = append(out, s[i:end:end]) // capacité figée : pas de fuite dans le morceau suivant

func ch06RemoveAt[T any](s []T, i int) []T {
	copy(s[i:], s[i+1:])
	var zero T
	s[len(s)-1] = zero // libère la dernière référence (anti-fuite)
	return s[:len(s)-1]
}
```

**Pourquoi.** L'expression à **trois indices** `s[i:end:end]` fige la
**capacité** du morceau à sa longueur : un `append` ultérieur sur ce morceau
réallouera au lieu d'écraser la mémoire du morceau suivant (`s[i:end]`
laisserait la capacité déborder — c'est l'aliasing que le test détecte). Dans
`removeAt`, décaler les éléments laisse une **copie du dernier** en fin de
slice ; pour un slice de pointeurs, cette référence morte empêcherait le GC de
collecter l'objet. La remettre à sa valeur zéro avant de tronquer évite la fuite.

## Chapitre 7 — [Maps & strings](../chapitres/07-maps-strings.md)

**Énoncé.** Observer l'ordre aléatoire d'itération d'une map, inverser une chaîne
**par runes**, et écrire `wordFrequencies` trié par fréquence décroissante.

**Solution** ([`ch07.go`](../code/annexe-L-solutions/ch07.go)) :

```go
// code/annexe-L-solutions/ch07.go
func ch07ReverseString(s string) string {
	r := []rune(s) // et non []byte : préserve les caractères multi-octets
	slices.Reverse(r)
	return string(r)
}

slices.SortFunc(out, func(a, b ch07WordFreq) int {
	if a.N != b.N {
		return b.N - a.N // fréquence décroissante
	}
	return strings.Compare(a.Word, b.Word) // départage déterministe
})
```

**Pourquoi.** L'ordre d'itération d'une map est **volontairement randomisé** par
le runtime : trier explicitement est la seule façon d'obtenir une sortie stable.
Convertir en `[]rune` (et non `[]byte`) traite les caractères Unicode entiers :
inverser `"café"` par octets couperait l'octet médian du « é » et produirait de
l'**UTF-8 invalide**. Le tri secondaire par mot rend `wordFrequencies`
**déterministe** malgré l'itération aléatoire de la map de comptage.

## Chapitre 8 — [Structs, méthodes & composition](../chapitres/08-structs-methodes.md)

**Énoncé.** Montrer qu'un récepteur **valeur** empêche `Deposit` de muter le
solde, et réduire la taille d'un struct en **réordonnant** ses champs.

**Solution** ([`ch08.go`](../code/annexe-L-solutions/ch08.go)) :

```go
// code/annexe-L-solutions/ch08.go
func (a *ch08Account) Deposit(n int) { a.balance += n } // récepteur POINTEUR : mute en place

type ch08Padded struct { a bool; b int64; c bool } // 24 octets (padding)
type ch08Packed struct { b int64; a bool; c bool } // 16 octets (compact)
```

**Pourquoi.** Un récepteur **pointeur** (`*ch08Account`) opère sur la valeur
d'origine ; un récepteur **valeur** travaillerait sur une **copie** et le solde
appelant ne bougerait pas. Côté taille, l'**alignement mémoire** insère du
*padding* : en groupant les champs du plus grand au plus petit, on supprime les
trous et l'on passe de 24 à 16 octets (`unsafe.Sizeof` le mesure). Ajouter un
champ non comparable (`tags []string`) à un `Point` rendrait par ailleurs
`Point{} == Point{}` illégal **à la compilation**.

## Chapitre 9 — [Interfaces](../chapitres/09-interfaces.md)

**Énoncé.** Ajouter un `Triangle` satisfaisant `Shape` sans rien changer
d'autre, et écrire `describe(x any)` distinguant un `fmt.Stringer`.

**Solution** ([`ch09.go`](../code/annexe-L-solutions/ch09.go)) :

```go
// code/annexe-L-solutions/ch09.go
func (t ch09Triangle) Area() float64 { return t.Base * t.Height / 2 } // satisfait Shape implicitement

func ch09Describe(x any) string {
	if s, ok := x.(fmt.Stringer); ok {
		return "Stringer: " + s.String()
	}
	return fmt.Sprintf("%T: %v", x, x)
}
```

**Pourquoi.** La satisfaction d'interface est **implicite** : dès que `Triangle`
possède `Area()`, il devient une `Shape` utilisable par `totalArea` — sans
déclaration ni modification du code existant. `describe` exploite l'**assertion
de type** `x.(fmt.Stringer)` pour brancher sur la méthode `String()` quand elle
existe. Comparer deux `any` contenant des `[]int` avec `==` **panique**
(`comparing uncomparable type`) : les slices ne sont pas comparables.

## Chapitre 10 — [Gestion des erreurs](../chapitres/10-erreurs.md)

**Énoncé.** Montrer que `%v` (au lieu de `%w`) rompt `errors.Is`, et ajouter une
méthode `Is` qui compare deux `*ParseError` sur la **ligne**.

**Solution** ([`ch10.go`](../code/annexe-L-solutions/ch10.go)) :

```go
// code/annexe-L-solutions/ch10.go
func (e *ch10ParseError) Is(target error) bool {
	var pe *ch10ParseError
	if !errors.As(target, &pe) {
		return false
	}
	return e.Line == pe.Line // égales si même ligne
}

return &ch10ParseError{Line: line, Err: fmt.Errorf("%q : %w", s, ch10ErrEmptyKey)} // %w préserve la chaîne
```

**Pourquoi.** Le verbe `%w` d'`fmt.Errorf` **emballe** l'erreur en conservant un
lien inspectable : `errors.Is(err, ErrEmptyKey)` remonte alors la chaîne
jusqu'à la sentinelle. Remplacer `%w` par `%v` **aplatit** l'erreur en simple
texte et rompt ce lien (`Is` renvoie `false`). La méthode `Is` personnalisée
permet une **égalité métier** (ici : même ligne), qu'`errors.Is` consulte en
priorité. Pour extraire une erreur d'un type donné, `errors.As` (ou son
équivalent typé `errors.AsType` sur les versions récentes) reste l'outil.

## Chapitre 11 — [Généricité](../chapitres/11-genericite.md)

**Énoncé.** Comprendre le rôle des `~` dans une contrainte, réécrire `Index` avec
`slices.Index`, et écrire `Zero[T]()`.

**Solution** ([`ch11.go`](../code/annexe-L-solutions/ch11.go)) :

```go
// code/annexe-L-solutions/ch11.go
type ch11Number interface{ ~int | ~int64 | ~float64 } // ~ : accepte les types DÉFINIS

func ch11Index[T comparable](xs []T, target T) int { return slices.Index(xs, target) }

func ch11Zero[T any]() T { var z T; return z }
```

**Pourquoi.** Le `~` (tilde) élargit la contrainte aux types **définis** sur ces
bases : sans lui, `type Celsius float64` serait refusé alors même que son type
sous-jacent est `float64`. `slices.Index` remplace avantageusement une recherche
linéaire écrite à la main. Enfin, `ch11Zero()` **sans argument** échoue (`cannot
infer T`) : rien ne permet de déduire `T` ; il faut l'expliciter,
`ch11Zero[int]()`. À noter : les **méthodes** ne peuvent pas être paramétrées
(`func (s *Stack[T]) Map[R any]` est refusé) — seules les fonctions et les types
le sont.

## Chapitre 12 — [Packages, modules & organisation](../chapitres/12-packages-modules.md)

**Énoncé.** Vérifier qu'`internal/` bloque un import externe, et qu'un
`Example` **testable** verrouille le format de sortie.

**Solution** ([`ch12.go`](../code/annexe-L-solutions/ch12.go),
[`ch12_test.go`](../code/annexe-L-solutions/ch12_test.go)) :

```go
// code/annexe-L-solutions/ch12_test.go
func Example_ch12Amount() {
	fmt.Println(ch12Amount(1250))
	fmt.Println(ch12Amount(-99))
	// Output:
	// 12.50 €
	// -0.99 €
}
```

**Pourquoi.** Un paquet placé sous `internal/` n'est importable que par du code
**partageant sa racine** : le compilateur rejette tout import externe (« use of
internal package not allowed »). C'est de l'encapsulation garantie, pas une
simple convention. Un `Example` dont le commentaire `// Output:` correspond à la
sortie est un **test exécutable** : casser le format (`12.50` → `12.5`) fait
échouer `go test`. `go mod tidy` reste par ailleurs le réflexe avant commit,
même quand rien ne change.

## Chapitre 13 — [Tests & outillage](../chapitres/13-tests-outillage.md)

**Énoncé.** Ajouter un cas accentué à la table de `TestSlugify`, et **fuzzer**
l'idempotence de `Slugify`.

**Solution** ([`ch13.go`](../code/annexe-L-solutions/ch13.go),
[`ch13_test.go`](../code/annexe-L-solutions/ch13_test.go)) :

```go
// code/annexe-L-solutions/ch13_test.go
func FuzzCh13Slugify(f *testing.F) {
	f.Add("Hello World")
	f.Add("café *** 42")
	f.Fuzz(func(t *testing.T, s string) {
		once := ch13Slugify(s)
		if twice := ch13Slugify(once); once != twice {
			t.Errorf("non idempotent : %q -> %q -> %q", s, once, twice)
		}
	})
}
```

**Pourquoi.** Pour que le repli d'accents (« Élevé » → `eleve`) reste cohérent,
`Slugify` doit produire une sortie **déjà slugifiée** : d'où l'**idempotence**
`Slugify(Slugify(s)) == Slugify(s)`. C'est une **propriété** idéale à fuzzer :
le moteur génère des entrées arbitraires et cherche un contre-exemple. Sur
plusieurs secondes de fuzzing, aucune entrée ne casse l'invariance — bon signe.
Retirer `t.Helper()` d'un assistant de test ferait par ailleurs pointer l'échec
sur la **ligne interne** de l'assistant plutôt que sur l'appel du test.

---

## 📌 À retenir

- Beaucoup d'exercices fondamentaux illustrent un **échec** volontaire
  (troncature, aliasing, chaîne d'erreur rompue, panique) : la solution montre la
  **version correcte** et nomme la cause.
- Les invariants (idempotence, absence d'aliasing, préservation UTF-8) se
  vérifient bien par **test de table** et **fuzzing**.
- Tout le code est regroupé dans
  [`code/annexe-L-solutions/`](../code/annexe-L-solutions/), un fichier par
  chapitre, préfixé `chNN` pour éviter les collisions.

## 🔁 Pour aller plus loin

- Les chapitres eux-mêmes, dont chaque section « 🧪 À tester soi-même » énonce
  l'exercice d'origine.
- [Annexe F — Idiomes & style](F-idiomes-style.md) pour le « pourquoi » des choix
  idiomatiques.
- [Ch. 13 — Tests & outillage](../chapitres/13-tests-outillage.md) et
  [Ch. 36 — Tests avancés, benchmarks & fuzzing](../chapitres/36-tests-benchmarks-fuzzing.md).
