# 36 — Tests avancés, benchmarks & fuzzing

> **Objectif** — **Mesurer** et **sécuriser** correctement : écrire des **benchmarks** fiables
> (`testing.B`, `b.Loop`, `b.ReportAllocs`), éviter l'**élimination par le compilateur**, comparer
> deux versions avec **`benchstat`** (significativité statistique), et **fuzzer** une fonction
> (`testing.F`, corpus) pour débusquer les cas limites.
>
> **Prérequis** — [Ch. 13](13-tests-outillage.md), [Ch. 26](26-allocation-escape.md), [Ch. 31](31-strings-profondeur.md)

---

## Introduction

Le [Ch. 13](13-tests-outillage.md) a posé la **culture du test**. Cette Partie VI **ouvre la boîte à
outils de la performance**, et tout commence ici : on ne peut pas **optimiser ce qu'on ne mesure pas**.
Le même package `testing` gère trois familles — tests, **benchmarks**, **fuzzing** — sans dépendance
externe. Ce chapitre se concentre sur les deux dernières, et sur la **rigueur statistique** qui sépare
une vraie amélioration d'un bruit de mesure. Code dans [`code/ch36-benchmarks-fuzzing/`](../code/ch36-benchmarks-fuzzing/).

Fil rouge : `FormatThousands(n)` insère une espace tous les trois chiffres (`1234567` → `"1 234 567"`),
en deux implémentations — une **naïve** (concaténation, [Ch. 31](31-strings-profondeur.md)) et une via
**`strings.Builder`**. On va prouver laquelle est meilleure.

---

## Un benchmark fiable : `b.Loop`

Un benchmark est une fonction **`BenchmarkXxx(b *testing.B)`**. Depuis Go **1.24**, on cadence la mesure
avec **`b.Loop()`** (et non plus `for i := 0; i < b.N; i++`) : il **réinitialise le chrono** au premier
tour (le setup ne compte pas) et l'**arrête** au dernier (le cleanup non plus).

Avant 1.24, deux pièges classiques pesaient sur le style `b.N`. D'abord, le chrono démarre dès l'entrée
dans la fonction : tout **setup** placé avant la boucle (génération de données, ouverture de fichier)
était mesuré, sauf à l'exclure à la main avec `b.ResetTimer()` — facile à oublier :

```go
// Style b.N (avant 1.24, illustratif) : sans b.ResetTimer(), le setup entre dans la mesure.
func BenchmarkOld(b *testing.B) {
	data := buildLargeFixture() // coûteux : sans la ligne suivante, ce setup est mesuré
	b.ResetTimer()              // remet le chrono à zéro : exclut tout ce qui précède
	for i := 0; i < b.N; i++ {
		process(data)
	}
}
```

Ensuite, pour calibrer `b.N`, `go test` devait **rappeler toute la fonction** — donc rejouer le setup —
avec des valeurs croissantes jusqu'à approcher la durée cible. `b.Loop()` règle les deux d'un coup : la
fonction n'est appelée **qu'une seule fois** par mesure, et c'est la boucle elle-même qui gère son
nombre d'itérations en interne.

```go
// code/ch36-benchmarks-fuzzing/bench_test.go
var sink string // voir « anti-élimination » ci-dessous

func BenchmarkBuilder(b *testing.B) {
	b.ReportAllocs()           // ajoute B/op et allocs/op au rapport
	for b.Loop() {
		sink = FormatThousands(1234567)
	}
}
```

```
$ go test -bench=. -benchmem -run=^$ ./ch36-benchmarks-fuzzing/...
BenchmarkBuilder-8   17174445   69.30 ns/op   24 B/op   2 allocs/op
BenchmarkNaive-8     11294998  106.6  ns/op   24 B/op   3 allocs/op
```

- `-bench=.` sélectionne les benchmarks (regexp) ; `-run=^$` **désactive les tests** classiques.
- `-benchmem` (ou `b.ReportAllocs()`) ajoute **`B/op`** et **`allocs/op`** — souvent plus parlants que
  les nanosecondes. `-8` = `GOMAXPROCS`.
- Le nombre de tours (`17174445`) est **choisi par `go test`** pour atteindre une durée stable (~1 s par
  défaut, réglable par `-benchtime`) : il mesure le temps d'un petit nombre d'itérations, **extrapole**
  combien il en faudrait pour atteindre la cible, relance avec ce total, et répète jusqu'à converger — en
  un seul appel à la fonction grâce à `b.Loop()` (contre plusieurs appels, setup compris, avec l'ancien
  style `b.N` ci-dessus).

## ⚠️ L'élimination par le compilateur (dead-code elimination)

Le piège n°1 du micro-benchmark : si le **résultat n'est pas utilisé**, le compilateur peut **supprimer
l'appel** que vous croyez mesurer. La parade : affecter le résultat à une **variable de package**
(le `sink`), que le compilateur ne peut pas prouver inutile.

```
  SANS sink (FAUX)                  AVEC sink (correct)
  for b.Loop() {                    for b.Loop() {
      FormatThousands(123)              sink = FormatThousands(123)
  }   ^                                 ^
      +-- résultat ignoré                   +-- résultat publié hors de la
          -> appel potentiellement              fonction -> l'appel DOIT
             éliminé -> 0.3 ns/op                 s'exécuter -> mesure réelle
```

> 💡 **`b.Loop` aide aussi, mais seulement entre ses accolades.** Le compilateur garde **vivants** les
> paramètres, résultats et variables affectées **syntaxiquement à l'intérieur** du `for b.Loop() { ... }`
> (un `runtime.KeepAlive` inséré automatiquement depuis **1.26** — 🆕 Go 1.2x plus bas) : même sans
> `sink`, une simple variable locale affectée dans la boucle échapperait déjà à l'élimination. Mais cette
> garantie s'arrête à la frontière syntaxique de la boucle — un appel **indirect** (via une fonction
> intermédiaire) ou un ancien benchmark en style `b.N` n'en bénéficient pas. Le `sink` de package reste
> donc l'idiome le **plus robuste** : explicite, et indépendant de la forme exacte du code mesuré.

## Sous-benchmarks : `b.Run`

Comme `t.Run`, **`b.Run`** crée des **sous-benchmarks** nommés — idéal pour balayer des tailles d'entrée
et voir le coût **croître** :

```go
func BenchmarkSizes(b *testing.B) {
	for _, in := range []struct{ name string; n int }{
		{"digits=3", 999}, {"digits=7", 1234567}, {"digits=13", 1234567890123},
	} {
		b.Run(in.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink = FormatThousands(in.n)
			}
		})
	}
}
```

```
BenchmarkSizes/digits=3-8    52.52 ns/op   16 B/op   2 allocs/op
BenchmarkSizes/digits=7-8    69.58 ns/op   24 B/op   2 allocs/op
BenchmarkSizes/digits=13-8   92.41 ns/op   40 B/op   2 allocs/op
```

Comme pour `t.Run` ([Ch. 13](13-tests-outillage.md)), le nom du sous-benchmark se compose avec `/` :
`go test -bench=Sizes/digits=7 -run=^$` ne lance que ce cas précis — pratique pour ré-isoler une
régression repérée sur une seule taille sans relancer toute la suite.

## Comparer deux versions : `benchstat`

Un benchmark isolé ne dit rien : **107 ns, est-ce mieux ou moins bien ?** La réponse exige une
**comparaison statistique**. L'outil officiel est **`benchstat`** (package `golang.org/x/perf`) :

```bash
go install golang.org/x/perf/cmd/benchstat@latest   # une fois

# benchstat apparie les mesures PAR NOM de benchmark : on renomme les deux
# versions sous un nom commun (Format) pour comparer l'ancienne à la nouvelle.
go test -bench=Naive   -count=10 -run=^$ ./ch36-benchmarks-fuzzing/... | sed 's/Naive/Format/'   > old.txt
go test -bench=Builder -count=10 -run=^$ ./ch36-benchmarks-fuzzing/... | sed 's/Builder/Format/' > new.txt
benchstat old.txt new.txt
```

```
            |   old.txt   |             new.txt              |
            |   sec/op    |   sec/op     vs base             |
Format-8      106.90n ± 8%   69.60n ± 1%  -34.89% (p=0.000 n=10)
            |  allocs/op  |  allocs/op   vs base             |
Format-8       3.000 ± 0%    2.000 ± 0%  -33.33% (p=0.000 n=10)
```

- **`± 8%`** est la **variation** (médiane des écarts) : une mesure à ±20 % est trop bruitée, augmentez
  `-count` ou stabilisez la machine.
- **`p=0.000`** : la différence est **statistiquement significative**. `benchstat` compare les deux
  séries avec le **test de Mann-Whitney** (non paramétrique : il ne suppose pas une distribution normale
  des temps — réaliste ici, un GC ou un voisin bruyant créent une **traîne** vers le haut, pas une
  dispersion symétrique) et résume chaque série par sa **médiane**, plus robuste qu'une moyenne face à
  ces valeurs aberrantes. Le seuil par défaut est **`p < 0,05`**, réglable via `-alpha`. Un **`~`**
  signifierait « indistinguable du bruit » — ne jamais conclure dans ce cas.
- Verdict : `strings.Builder` est **−35 % de temps** et **−33 % d'allocations**. Réel, mesuré, prouvé.

> ⚠️ **Toujours `-count=10`** (ou plus) avant de conclure. Une seule mesure A et une seule mesure B ne
> permettent **aucune** inférence : le bruit (fréquence CPU, GC, voisins) domine souvent l'écart.

## Fuzzing : casser les invariants

Un test vérifie les cas **auxquels vous avez pensé** ; le **fuzzing** génère des entrées pour trouver
**ceux que vous avez oubliés**. Une fonction **`FuzzXxx(f *testing.F)`** sème un **corpus** (`f.Add`)
puis confie à `f.Fuzz` une propriété qui doit **toujours** tenir :

```go
// code/ch36-benchmarks-fuzzing/format_test.go
func FuzzFormatThousands(f *testing.F) {
	// Corpus de seeds : on sème les cas limites connus (0, signe, MinInt/MaxInt).
	for _, seed := range []int{0, 1, -1, 999, 1000, -1000, math.MaxInt, math.MinInt} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, n int) {
		got := FormatThousands(n)
		// Invariant 1 : retirer les séparateurs redonne les chiffres d'origine.
		if stripped := strings.ReplaceAll(got, " ", ""); stripped != strconv.Itoa(n) {
			t.Errorf("FormatThousands(%d) = %q ; sans espaces %q != %q", n, got, stripped, strconv.Itoa(n))
		}
		// Invariant 2 : les deux implémentations coïncident toujours.
		if naive := formatNaive(n); naive != got {
			t.Errorf("désaccord pour %d : builder=%q naive=%q", n, got, naive)
		}
	})
}
```

Le second invariant illustre le **fuzzing différentiel** : plutôt qu'une assertion métier figée, on
compare **deux implémentations indépendantes** de la même spécification — `formatNaive` et
`FormatThousands`, le fil rouge du chapitre. Tout désaccord entre elles est un bug dans l'une des deux,
sans même avoir à formuler l'invariant exact.

```bash
go test -run=^$ ./ch36-benchmarks-fuzzing/...                 # n'exécute QUE le corpus de seeds
go test -fuzz=FuzzFormatThousands -fuzztime=6s ./ch36-...     # fuzzing actif : génère des entrées
```

```
fuzz: elapsed: 6s, execs: 1445640 (248577/sec), new interesting: 1 (total: 9)
PASS
```

1,4 **million** d'entrées en 6 s, sur 8 workers, sans violer l'invariant. **`new interesting: 1 (total:
9)`** : sur ces 1,4 million d'entrées générées, une seule a élargi la **couverture** par rapport au
corpus existant (9 entrées au total, seeds comprises) — toutes les autres ne faisaient que repasser sur
des chemins déjà connus. C'est la clé du fuzzing Go : il n'avance pas **au hasard**, il **mute** les
entrées du corpus (inversion de bit, insertion, troncature, recombinaison...) et ne **garde** que celles
qui font exécuter du code **jamais atteint** jusque-là, repéré par une instrumentation de couverture
insérée au build (architectures AMD64/ARM64 uniquement). Une recherche purement aléatoire sur un `int`
64 bits a une probabilité quasi nulle de retomber un jour sur `math.MinInt` ; une recherche guidée par la
couverture, elle, repère qu'une mutation fait basculer la branche `neg` ou approcher un débordement, et
**mute autour** jusqu'à l'exploiter :

```
   seed corpus (f.Add)          mutation (bit-flip, insertion,        corpus généré
   + testdata/fuzz/       --->  troncature, recombinaison...)   --->  ($GOCACHE/fuzz)
                                          |
                                          v
                           exécution instrumentée (couverture)
                                          |
                    +---------------------+---------------------+
                    |                                           |
          branche déjà connue                         branche jamais exécutée
          -> entrée écartée                            -> "new interesting"
                                                         -> ajoutée au corpus,
                                                            base de futures mutations
```

Un bon invariant capture une **vérité métier** : ici, _le formatage ne perd ni n'invente de chiffre_.
C'est ainsi que le fuzzing trouve les classiques : débordement sur `math.MinInt`, indices hors bornes,
paniques.

> 💡 **Deux corpus**. Les seeds `f.Add` et les entrées « intéressantes » découvertes vivent dans un
> **cache** (`$GOCACHE/fuzz`). Quand le fuzzer trouve un **crash**, il **minimise** d'abord l'entrée
> fautive — la réduit à la plus petite forme qui reproduit encore l'échec, plus lisible pour le
> diagnostic — puis l'écrit dans **`testdata/fuzz/FuzzXxx/`** : ce fichier, **versionné**, devient un cas
> de **non-régression** rejoué à chaque `go test`. Le bug ne pourra plus jamais repasser inaperçu.

## Couverture (rappel)

```bash
go test -cover ./ch36-benchmarks-fuzzing/...                       # % de lignes exécutées
go test -coverprofile=cover.out ./ch36-benchmarks-fuzzing/...
go tool cover -html=cover.out                                      # visualisation ligne à ligne
```

La couverture **mesure l'exécution, pas la pertinence** ([Ch. 13](13-tests-outillage.md)) : c'est un
indicateur, pas un objectif. Le fuzzing, lui, **augmente** mécaniquement la couverture en explorant des
branches oubliées.

---

## 🆕 Go 1.2x

- **1.24** — **`b.Loop()`** : la façon moderne de cadencer un benchmark. La fonction n'est appelée
  **qu'une seule fois** par mesure (setup/cleanup ne sont plus rejoués à chaque calibration de `b.N`) et
  le chrono est géré automatiquement — c'est désormais l'idiome recommandé.
- **1.24** — le nouvel analyzer `go vet` **`tests`** détecte les déclarations malformées de
  `Test`/`Benchmark`/`Fuzz`/`Example` (nom incorrect, signature invalide, `Example` documentant un
  identifiant inexistant) ; il fait partie du sous-ensemble lancé automatiquement par `go test`
  (🔁 [Ch. 13](13-tests-outillage.md)).
- **1.25** — **`testing/synctest`** passe en **GA** : exécute un test dans une **bulle isolée** à
  **horloge virtuelle**, pour vérifier un timeout, un rate limiter ou un backoff de façon
  **déterministe et instantanée**, sans `time.Sleep` réel. Détaillé au
  [Ch. 23 — Tests concurrents](23-patterns-concurrence.md) ; ce chapitre-ci s'en tient aux benchmarks et
  au fuzzing.
- **1.26** — **`b.Loop` n'empêche plus l'inlining** du corps mesuré : en 1.24, la protection contre
  l'élimination était implémentée en **bloquant l'inlining** dans le corps de la boucle, ce qui pouvait
  fausser un micro-benchmark (le code mesuré n'étant plus optimisé comme en production). Depuis 1.26, le
  compilateur insère plutôt un **`runtime.KeepAlive`** automatique sur les valeurs affectées dans la
  boucle (cf. l'encart plus haut) : l'inlining redevient possible, le code mesuré est désormais optimisé
  **comme en production**.

## ⚠️ Pièges

- **Résultat ignoré** → appel éliminé → benchmark qui mesure du vide (0,3 ns/op suspect). Publiez via un
  **`sink`** de package.
- **Une seule mesure** par version → conclusion invalide. **`-count=10`** + **`benchstat`** + vérifier
  **`p < 0,05`**.
- **Setup dans la boucle** → vous mesurez le setup. Sortez-le avant `for b.Loop()` (le chrono démarre au
  premier tour).
- **Machine instable** (turbo, thermique, autres process) → `± 20 %`. Fermez les voisins, fixez la
  fréquence si possible, augmentez `-count`.
- **Invariant de fuzz trop faible** (`!= ""`) → ne trouve rien. Visez une **propriété métier** forte
  (round-trip, idempotence, conservation), ou un fuzzing **différentiel** contre une seconde implémentation.
- **Corpus de seeds pauvre** (un seul cas « heureux ») → la mutation part de peu et met plus de temps à
  atteindre les chemins rares. Semez les **frontières connues** — zéro, signe, débordement, valeurs
  vides — comme ici avec `math.MinInt`/`math.MaxInt`.

## ⚡ Performance

- **`B/op` et `allocs/op` d'abord** : une allocation en moins vaut souvent plus que quelques
  nanosecondes ([Ch. 26](26-allocation-escape.md)). Ici, `strings.Builder` économise une allocation
  **et** du temps.
- **`-benchtime=10000x`** fixe un **nombre de tours** (au lieu d'une durée) pour comparer à charge
  strictement égale.
- 🔁 Les chiffres ne disent pas **où** part le temps : c'est le rôle du **profiling**
  ([Ch. 37](37-profiling-pprof.md)) et des **traces** ([Ch. 38](38-traces-flight-recorder.md)).

## 🧪 À tester soi-même

```bash
cd code
go test -bench=. -benchmem -run=^$ ./ch36-benchmarks-fuzzing/...
go test -fuzz=FuzzFormatThousands -fuzztime=10s ./ch36-benchmarks-fuzzing/...
```

À essayer :

1. Retirez le `sink` de `BenchmarkBuilder` et observez le temps tomber à ~0,3 ns/op (appel éliminé).
2. Lancez les deux versions avec `-count=10` et comparez-les vous-même avec `benchstat`.
3. Introduisez un bug (oubliez le signe `-`) et relancez le fuzzer : il dépose un crasher dans
   `testdata/fuzz/`.

---

## 📌 À retenir

- Benchmark = **`BenchmarkXxx(b *testing.B)`** + **`for b.Loop()`** (🆕 1.24) + **`b.ReportAllocs()`** ;
  `-benchmem` donne `B/op` / `allocs/op`.
- **Toujours publier le résultat** dans un `sink` de package : `b.Loop()` protège déjà ce qui est
  **syntaxiquement dans la boucle**, mais le `sink` reste robuste face à l'indirection et à l'ancien
  style `b.N`.
- Une mesure isolée ne conclut pas : **`-count=10`** + **`benchstat`** (test de Mann-Whitney sur les
  médianes) + **`p < 0,05`** distinguent le gain réel du bruit.
- **Fuzzing** = **`FuzzXxx(f *testing.F)`** + seeds **`f.Add`** (semez les frontières connues) +
  **invariant** dans `f.Fuzz`, exploré par **mutation guidée par la couverture** plutôt que le hasard ;
  un crash **minimisé** est figé dans **`testdata/fuzz/`** comme test de non-régression.
- La **couverture** mesure l'exécution, pas la pertinence — un indicateur, jamais un objectif.

## 🔁 Pour aller plus loin

- [Ch. 23 — Tests concurrents](23-patterns-concurrence.md) : `testing/synctest`, pour tester un timeout
  ou un rate limiter de façon déterministe, sans temps réel.
- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : trouver **où** le temps et la mémoire partent.
- [Ch. 39 — Compilation & inlining](39-compilation-inlining-pgo.md) : pourquoi un appel est inliné (ou non).
- [Ch. 40 — Méthodologie de performance](40-methodologie-performance.md) : la boucle mesure → hypothèse → re-mesure.
- Doc : `go help test`, `go help testflag`, `go doc testing.B`, `pkg.go.dev/golang.org/x/perf/cmd/benchstat`.
