# 13 — Tests & outillage de base

> **Objectif** — Adopter la **culture du test** intégrée à Go : écrire des tests
> **table-driven**, des sous-tests, des `Example` exécutables, et exploiter `go test`/`go vet`.
>
> **Prérequis** — [Ch. 12 — Packages & modules](12-packages-modules.md), [Ch. 5 — Fonctions](05-fonctions.md)

---

## Introduction

Le test n'est pas un add-on en Go : il est **dans le langage et l'outillage**. Aucune dépendance
à installer, aucun framework à choisir. Un fichier `*_test.go`, une fonction `TestXxx`, et
`go test` — c'est tout. Cette intégration explique la **culture du test** très répandue dans
l'écosystème : pas de débat « JUnit ou TestNG », « pytest ou unittest », « Jest ou Mocha » — un
seul outil, identique dans tous les projets Go, de la bibliothèque standard au plus petit module.
Le coût d'entrée pour écrire un test est quasi nul, ce qui encourage à en écrire plus tôt et plus
souvent.

L'exemple complet est dans [`code/ch13-tests/`](../code/ch13-tests/).

---

## Un premier test

Les tests vivent dans des fichiers **`*_test.go`** (non inclus dans le binaire final). Une fonction
de test a la signature `func TestXxx(t *testing.T)` :

```go
// slugify_test.go
func TestSlugify(t *testing.T) {
	got := Slugify("Hello, World!")
	if got != "hello-world" {
		t.Errorf("Slugify = %q ; attendu %q", got, "hello-world")
	}
}
```

```bash
go test ./...          # lance tous les tests du module
go test -v ./ch13-tests/...   # mode verbeux (chaque test affiché)
go test -run Slugify   # filtre par nom (regexp)
```

> 💡 **`t.Error`/`t.Errorf`** signalent un échec mais **continuent** le test ; **`t.Fatal`/`t.Fatalf`**
> l'**arrêtent** immédiatement (utile quand la suite n'a plus de sens, ex. après une erreur
> d'ouverture). Le message décrit **`got` vs `want`** — la convention.

> ⚠️ Le nom doit suivre **exactement** le motif `TestXxx`, où `Xxx` commence par une **majuscule**
> (ou un chiffre) : `func TestFoo` est reconnu, mais `func Testfoo` **compile sans erreur** et
> n'est **jamais exécuté** par `go test`. `go vet` (que `go test` invoque en partie avant de
> lancer les tests) le signale : `Testfoo has malformed name: first letter after 'Test' must
> not be lowercase`.

## Tests **table-driven** + sous-tests

L'idiome Go par excellence : une **table** de cas, une **boucle**, et un **sous-test** `t.Run` par
cas. Chaque cas est nommé, isolable (`-run TestSlugify/vide`), et son échec n'arrête pas les autres.

```go
func TestSlugify(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Hello, World!", "hello-world"},
		{"ponctuation", "  Go 1.26 : Top!  ", "go-1-26-top"},
		{"déjà slug", "go-1-26-top", "go-1-26-top"},
		{"vide", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { // un sous-test nommé par cas
			if got := Slugify(tc.in); got != tc.want {
				t.Errorf("Slugify(%q) = %q ; attendu %q", tc.in, got, tc.want)
			}
		})
	}
}
```

```
   go test -v :
   === RUN   TestSlugify
   === RUN   TestSlugify/simple
   === RUN   TestSlugify/ponctuation
   === RUN   TestSlugify/déjà_slug      (les espaces du nom deviennent '_')
   --- PASS: TestSlugify (0.00s)
       --- PASS: TestSlugify/simple (0.00s)
       ...
```

> 💡 Ajouter un cas = ajouter **une ligne**. C'est ce qui rend ce style si productif : la logique
> de test est écrite une fois, les cas se multiplient sans effort.

> 💡 Variante courante dans la stdlib : `[]struct{...}` peut être remplacé par
> **`map[string]struct{...}`**, la clé servant de nom de cas (plus besoin du champ `name`). Le
> prix est la perte de l'**ordre d'exécution déterministe** — l'itération sur une map est
> randomisée (🔁 [Ch. 7](07-maps-strings.md)) — mais c'est sans conséquence ici puisque chaque
> sous-test est indépendant.

> ⚠️ **Sous-tests parallèles et variable de boucle.** Paralléliser chaque cas avec `t.Parallel()`
> ne pose pas le même problème que sans elle : un appel à `t.Parallel()` **met le sous-test en
> pause** et rend la main à la boucle englobante, qui passe aussitôt au tour suivant. Avant Go
> **1.22**, la variable de boucle était **unique et partagée** par toutes les itérations
> ([Ch. 4](04-flux-controle.md), [Ch. 15](15-closures.md)) : au moment où les sous-tests en pause
> reprenaient, ils lisaient tous la **même** valeur — celle laissée par le dernier tour de boucle.
>
> ```go
> for _, tc := range cases {
> 	tc := tc // nécessaire avant Go 1.22 ; sans quoi tous les sous-tests testent le DERNIER tc
> 	t.Run(tc.name, func(t *testing.T) {
> 		t.Parallel()
> 		if got := Slugify(tc.in); got != tc.want {
> 			t.Errorf("Slugify(%q) = %q ; attendu %q", tc.in, got, tc.want)
> 		}
> 	})
> }
> ```
>
> Le piège est **sournois** : le test s'exécute et rapporte un résultat — juste pas sur la bonne
> entrée, donc sans erreur de compilation ni panique pour l'indiquer. Depuis Go 1.22, la portée
> **par itération** rend `tc := tc` inutile (à condition que `go.mod` déclare `go 1.22` ou plus).

## Helpers : `t.Helper()`

Une fonction d'assertion partagée doit appeler **`t.Helper()`** : en cas d'échec, Go pointe alors
la ligne de l'**appelant** (le cas de test) au lieu de la ligne interne du helper.

```go
func mustEqual(t *testing.T, got, want string) {
	t.Helper() // sans cette ligne, l'échec pointerait ici, pas le cas de test
	if got != want {
		t.Errorf("got %q ; want %q", got, want)
	}
}
```

| Méthode              | Effet                                                                  | Quand l'utiliser                                            |
| -------------------- | ---------------------------------------------------------------------- | ----------------------------------------------------------- |
| `t.Error`/`t.Errorf` | Marque l'échec, **continue** le test                                   | Plusieurs assertions indépendantes dans le même test        |
| `t.Fatal`/`t.Fatalf` | Marque l'échec, **arrête** le test immédiatement                       | La suite n'a plus de sens (échec de setup, ex. ouverture)   |
| `t.Skip`/`t.Skipf`   | Marque le test **ignoré** (ni succès ni échec)                         | Précondition absente (réseau, OS, mode `-short`)            |
| `t.Helper()`         | N'arrête rien : exclut la fonction courante du **call frame** rapporté | Toute fonction d'assertion ou de setup partagée entre tests |

`t.Skip` se combine souvent avec **`testing.Short()`**, qui reflète le flag `-short` (utile pour
exclure les tests lents — réseau, base de données — d'un cycle de développement rapide) :

```go
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("test d'intégration ignoré en mode -short")
	}
	// ... appel réseau, base de données, etc.
}
```

> 💡 `t.Skip` sans condition est aussi pratique pour **désactiver temporairement** un test cassé
> pendant un refactoring, sans le supprimer ni le laisser échouer en CI — préférez-le à un
> commentaire qui finit par être oublié.

## `Example` : documentation exécutable

Une fonction **`ExampleXxx`** apparaît dans la doc (`go doc`, `pkg.go.dev`) **et** s'exécute comme
un test : le commentaire **`// Output:`** est comparé à la sortie réelle ([Ch. 12](12-packages-modules.md)).

```go
func ExampleSlugify() {
	fmt.Println(Slugify("Hello, World!"))
	fmt.Println(Slugify("  Go 1.26 : Top!  "))
	// Output:
	// hello-world
	// go-1-26-top
}
```

> 💡 Pour une sortie dont l'**ordre n'est pas garanti** (parcours de map, [Ch. 7](07-maps-strings.md)),
> utilisez **`// Unordered output:`** : Go compare les lignes sans tenir compte de l'ordre.

## Tester le système de fichiers : `t.TempDir` & `t.Cleanup`

**`t.TempDir()`** crée un répertoire temporaire **supprimé automatiquement** en fin de test —
chaque test est isolé, sans nettoyage manuel.

```go
func TestSaveLines(t *testing.T) {
	dir := t.TempDir() // effacé automatiquement à la fin
	path, err := SaveLines(dir, "out.txt", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("SaveLines: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "a\nb\nc\n" {
		t.Errorf("contenu = %q", got)
	}
}
```

**`t.Cleanup(fn)`** enregistre une fonction de nettoyage exécutée en fin de test, en ordre
**LIFO** (dernier enregistré, premier exécuté) — pratique quand un **helper** ouvre une ressource
et doit la libérer. L'ordre se vérifie concrètement :

```go
// slugify_test.go
func TestCleanupLIFO(t *testing.T) {
	var order []int
	t.Run("ressource", func(t *testing.T) {
		t.Cleanup(func() { order = append(order, 1) }) // enregistré en 1er -> exécuté en dernier
		t.Cleanup(func() { order = append(order, 2) }) // enregistré en 2nd -> exécuté en premier
	})
	if !slices.Equal(order, []int{2, 1}) {
		t.Errorf("ordre des cleanups = %v ; attendu [2 1] (LIFO)", order)
	}
}
```

Les cleanups du sous-test `"ressource"` s'exécutent dès qu'il se termine — avant même que
`TestCleanupLIFO` ne rende la main — d'où la possibilité de les observer via une simple variable
capturée par la closure.

> 💡 **Golden files.** Pour une sortie longue (rapport, JSON formaté, gabarit HTML), comparer
> caractère par caractère dans le code de test devient illisible. La technique consiste à
> comparer la sortie à un fichier de référence versionné sous `testdata/` — un nom de dossier que
> l'outillage `go` **ignore systématiquement** (ni compilé, ni traité comme package), donc libre
> d'accueillir n'importe quel contenu lu par le test via `os.ReadFile` — et à le régénérer via un
> flag dédié, par exemple `go test -update` (`if *update { os.WriteFile(path, got, 0o644) }`). La
> stdlib ne fournit pas cet outillage : c'est une **convention** répandue, pas une fonction du
> package `testing`.

## Couverture

```bash
go test -cover ./ch13-tests/...                    # pourcentage de lignes couvertes
go test -coverprofile=cover.out ./ch13-tests/...   # profil détaillé
go tool cover -func=cover.out                      # détail par fonction (terminal, idéal en CI)
go tool cover -html=cover.out                      # visualisation ligne à ligne (navigateur)
```

`-func` détaille le profil sans navigateur — pratique en CI. Sur `ch13-tests/`, il révèle un
détail instructif :

```
ch13-tests/main.go:11:      main          0.0%
ch13-tests/report.go:12:    SaveLines     80.0%
ch13-tests/slugify.go:11:   Slugify       100.0%
total:                      (statements)  48.4%
```

`main` est à **0 %** : `go run ./ch13-tests` l'exécute, mais aucun test ne l'appelle (et l'appeler
depuis un test relancerait tout le programme). Un total inférieur à 100 % n'est donc pas
forcément un signal d'alerte — encore faut-il savoir **quelle** ligne manque et pourquoi.

> ⚠️ La couverture **mesure l'exécution, pas la pertinence**. 100 % de lignes exécutées ne garantit
> pas que les **bons cas** sont testés. C'est un indicateur, pas un objectif.

## `go vet` : l'analyse statique

`go vet` détecte des erreurs que le compilateur laisse passer : verbe `Printf` incohérent,
copie de `sync.Mutex`, code mort… `go test` lance d'ailleurs **un sous-ensemble** de `vet`
automatiquement avant les tests. Exemple concret — le compilateur accepte ce code (`Printf`
prend des `...any`), mais l'incohérence entre verbe et argument est une vraie erreur :

```go
fmt.Printf("%d\n", "x") // compile (any) ; à l'exécution : "%!d(string=x)"
```

```
$ go vet ./...
./main.go:9:14: fmt.Printf format %d has arg "x" of wrong type string
```

```bash
go vet ./...
```

## Au-delà : benchmarks & fuzzing (teaser)

Le même package `testing` gère deux autres familles, détaillées au [Ch. 36](36-tests-benchmarks-fuzzing.md) :

```go
// Benchmark : b.Loop (🆕 1.24) cadence les itérations.
func BenchmarkSlugify(b *testing.B) {
	for b.Loop() {
		sink = Slugify("  Go 1.26 : Top!  ")
	}
}

// Fuzzing : Go génère des entrées pour casser un invariant. Lancé en test normal,
// seul le corpus de seeds (f.Add) s'exécute.
func FuzzSlugify(f *testing.F) {
	f.Add("Hello, World!")
	f.Fuzz(func(t *testing.T, s string) {
		out := Slugify(s)
		if Slugify(out) != out { // invariant : idempotence
			t.Errorf("non idempotent : %q -> %q", s, out)
		}
	})
}
```

```bash
go test -bench=. -benchmem ./ch13-tests/...   # benchmarks
go test -fuzz=FuzzSlugify ./ch13-tests/...    # fuzzing actif (génère des entrées)
```

---

## 🆕 Go 1.2x

- **1.24** — **`b.Loop()`** : nouvelle façon de cadencer un benchmark (remplace `for i := 0; i < b.N`) ;
  **`T.Chdir(dir)`** change le répertoire courant du test et le **restaure automatiquement** via
  `t.Cleanup` (incompatible avec `t.Parallel`) ; **`T.Context()`** renvoie un `context.Context`
  annulé juste avant les cleanups, pratique pour borner une opération asynchrone lancée par le
  test (🔁 [Ch. 22](22-context.md)).
- **1.25** — **`T.Attr(key, value)`** attache une métadonnée à un test (ticket, catégorie) ;
  **`T.Output()`** renvoie un `io.Writer` indenté comme `t.Log` ; nouveaux analyzers `go vet`
  **`waitgroup`** et **`hostport`**.
- **1.26** — **`T.ArtifactDir()`** + flag **`-artifacts`** : un répertoire **persistant** où un test
  dépose des fichiers de sortie (images, dumps) pour inspection post-mortem.

```go
func TestObservability(t *testing.T) {
	t.Attr("ticket", "GO-1234")             // 1.25
	fmt.Fprintln(t.Output(), "diagnostic")  // 1.25
	dir := t.ArtifactDir()                   // 1.26 : où écrire des artefacts
	_ = dir
}
```

## ⚠️ Pièges

- **Tests dépendants de l'ordre** ou d'un **état partagé** (variable globale) : fragiles. Chaque
  test doit être **indépendant** et idéalement parallélisable (`t.Parallel()` — voir le piège de
  capture de boucle pré-1.22 détaillé plus haut).
- **Asserter sur l'ordre d'itération d'une map** ([Ch. 7](07-maps-strings.md)) : il est **randomisé**.
  Triez avant de comparer, ou utilisez `// Unordered output:`.
- **Oublier `t.Helper()`** dans une assertion partagée → les échecs pointent la mauvaise ligne.
- **`==` sur slices/maps** : interdit. Comparez avec `slices.Equal`/`maps.Equal` ou
  `reflect.DeepEqual`.
- **`t.Fatal` dans une autre goroutine** : interdit (il doit s'exécuter dans la goroutine du test).
  Renvoyez l'erreur par un canal ([Ch. 20](20-channels-select.md)).

## ⚡ Performance (du cycle de test)

- `go test` **met en cache** les résultats : un package non modifié n'est pas re-testé (mention
  `(cached)`). Forcez avec **`-count=1`**.
- **`t.Parallel()`** fait tourner les tests marqués en parallèle (jusqu'à `GOMAXPROCS`) — utile
  pour des suites lentes (I/O, réseau).
- Activez le **détecteur de data races** sur le code concurrent : `go test -race`
  ([Ch. 23](23-patterns-concurrence.md)).

## 🧪 À tester soi-même

```bash
cd code
go test -v ./ch13-tests/...
go test -cover ./ch13-tests/...
go test -bench=. -benchmem ./ch13-tests/...
go test -fuzz=FuzzSlugify -fuzztime=5s ./ch13-tests/...
```

À essayer :

1. Ajoutez un cas à la table de `TestSlugify` (ex. un texte avec accents) et observez le résultat.
2. Retirez `t.Helper()` de `mustEqual` et comparez la ligne pointée en cas d'échec.
3. Lancez le fuzzing 5 s : Go tente de violer l'idempotence. Trouve-t-il une entrée fautive ?

---

## 📌 À retenir

- Test = fichier **`*_test.go`** + **`func TestXxx(t *testing.T)`** + `go test`. Rien à installer
  (mais le nom doit être exact : `Testfoo` ne s'exécute jamais).
- Le style **table-driven** + **`t.Run`** : une table de cas, un sous-test chacun, ajout = une ligne.
  En parallèle (`t.Parallel()`), `tc := tc` n'est plus nécessaire dès `go.mod` en `go 1.22+`.
- **`t.Helper()`** pour les assertions partagées ; **`t.Skip`/`testing.Short()`** pour les tests
  longs ; **`Example`** + `// Output:` = doc **et** test.
- **`t.TempDir`**/**`t.Cleanup`** isolent les effets de bord (ordre **LIFO**) ; `-cover`/`-func`
  mesurent l'exécution (sans juger la pertinence) ; `go vet` complète le compilateur.
- 🆕 `b.Loop`/`T.Chdir`/`T.Context` (1.24), `T.Attr`/`T.Output` (1.25), `T.ArtifactDir` (1.26) ;
  benchmarks & fuzzing partagent le package `testing`.

## 🔁 Pour aller plus loin

- [Ch. 36 — Tests avancés, benchmarks & fuzzing](36-tests-benchmarks-fuzzing.md) : `benchstat`,
  corpus de fuzz, pièges de micro-bench.
- [Ch. 23 — Tests concurrents](23-patterns-concurrence.md) : `-race`, `testing/synctest`.
- [Ch. 12 — Packages & modules](12-packages-modules.md) : `Example` comme documentation.
- Annexe B — Antisèche des commandes `go`.
