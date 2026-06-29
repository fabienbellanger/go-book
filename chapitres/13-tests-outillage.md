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
l'écosystème.

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
et doit la libérer.

## Couverture

```bash
go test -cover ./ch13-tests/...                    # pourcentage de lignes couvertes
go test -coverprofile=cover.out ./ch13-tests/...   # profil détaillé
go tool cover -html=cover.out                      # visualisation ligne à ligne
```

> ⚠️ La couverture **mesure l'exécution, pas la pertinence**. 100 % de lignes exécutées ne garantit
> pas que les **bons cas** sont testés. C'est un indicateur, pas un objectif.

## `go vet` : l'analyse statique

`go vet` détecte des erreurs que le compilateur laisse passer : verbe `Printf` incohérent,
copie de `sync.Mutex`, code mort… `go test` lance d'ailleurs **un sous-ensemble** de `vet`
automatiquement avant les tests.

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

- **1.24** — **`b.Loop()`** : nouvelle façon de cadencer un benchmark (remplace `for i := 0; i < b.N`).
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
  test doit être **indépendant** et idéalement parallélisable (`t.Parallel()`).
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

- Test = fichier **`*_test.go`** + **`func TestXxx(t *testing.T)`** + `go test`. Rien à installer.
- Le style **table-driven** + **`t.Run`** : une table de cas, un sous-test chacun, ajout = une ligne.
- **`t.Helper()`** pour les assertions partagées ; **`Example`** + `// Output:` = doc **et** test.
- **`t.TempDir`**/**`t.Cleanup`** isolent les effets de bord ; `-cover` mesure (sans juger la
  pertinence) ; `go vet` complète le compilateur.
- 🆕 `b.Loop` (1.24), `T.Attr`/`T.Output` (1.25), `T.ArtifactDir` (1.26) ; benchmarks & fuzzing
  partagent le package `testing`.

## 🔁 Pour aller plus loin

- [Ch. 36 — Tests avancés, benchmarks & fuzzing](36-tests-benchmarks-fuzzing.md) : `benchstat`,
  corpus de fuzz, pièges de micro-bench.
- [Ch. 23 — Tests concurrents](23-patterns-concurrence.md) : `-race`, `testing/synctest`.
- [Ch. 12 — Packages & modules](12-packages-modules.md) : `Example` comme documentation.
- Annexe B — Antisèche des commandes `go`.
