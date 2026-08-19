# Projet 4 — Bibliothèque générique : `gends`

> **Objectif** — Concevoir une **bibliothèque réutilisable** publiable, et pas
> seulement « du code qui marche » : une **API générique** soignée (génériques +
> contraintes), de la **documentation testable** (`Example`), des **tests**, des
> **benchmarks**, du **fuzzing**, et une discipline de **compatibilité** (SemVer)
> outillée par une **CI**.
>
> **Réinvestit** — [Ch. 11 Généricité](../../chapitres/11-genericite.md),
> [Ch. 12 Packages](../../chapitres/12-packages-modules.md),
> [Ch. 13 Tests & outillage](../../chapitres/13-tests-outillage.md),
> [Ch. 36 Benchmarks & fuzzing](../../chapitres/36-tests-benchmarks-fuzzing.md).

---

## 1. Cahier des charges

`gends` (_generic data structures_) regroupe trois structures de données
génériques, chacune dans **son propre package** importable :

| Package  | Type                     | Idée                                          |
| -------- | ------------------------ | --------------------------------------------- |
| `set`    | `Set[T comparable]`      | Ensemble (union, intersection, différence…).  |
| `pqueue` | `Queue[T any]`           | File de priorité (tas binaire + comparateur). |
| `lru`    | `Cache[K comparable, V]` | Cache borné à éviction LRU.                   |

Contraintes de **conception d'API** :

- **Le type zéro est inutilisable mais explicite** : on construit via `New`/`Of`.
- **Contraintes minimales** : `Set` n'exige que `comparable` ; la fonction libre
  `set.Sorted` ajoute `cmp.Ordered` **seulement là où le tri l'impose**.
- **Pas de surprise** : les opérations ensemblistes renvoient un **nouvel**
  ensemble sans modifier les opérandes ; `lru.Contains` ne touche pas la récence.
- **Itération moderne** : `set.All()` renvoie une `iter.Seq[T]` (Go 1.23).
- **Méthode générique** : `set.Map` déclare son propre paramètre de type
  (🆕 Go 1.27) — voir §6 bis.
- **Aucune dépendance externe** : 100 % bibliothèque standard.

```go
import (
    "example.com/gends/lru"
    "example.com/gends/pqueue"
    "example.com/gends/set"
)

s := set.Of(1, 2, 3).Union(set.Of(3, 4)) // {1,2,3,4}
q := pqueue.NewOrdered[int]()             // min-heap
c := lru.New[string, []byte](128)         // cache de 128 entrées
```

---

## 2. Génériques & contraintes : le bon dosage

Le fil conducteur du projet est : **exiger le strict nécessaire**.

```go
// Set se contente de comparable — l'exigence d'une clé de map.
type Set[T comparable] struct{ m map[T]struct{} }

// Mais trier réclame un ordre total : la contrainte cmp.Ordered apparaît
// uniquement ici, dans une fonction LIBRE (pas une méthode).
func Sorted[T cmp.Ordered](s *Set[T]) []T { /* … */ }
```

> 💡 **Pourquoi une fonction et non une méthode ?** Go n'autorise pas une
> méthode à ajouter une contrainte au paramètre de type du récepteur. `Sorted`
> doit donc être une **fonction de package** : c'est le motif idiomatique
> (comme `slices.Sort` vis-à-vis d'une `[]T`).

La file de priorité illustre l'autre voie : **un comparateur en paramètre**
plutôt qu'une contrainte, pour accepter _n'importe quel_ type :

```go
q := pqueue.New(func(a, b task) bool { return a.at < b.at }) // priorité = échéance
// et un raccourci pour les types ordonnés :
q := pqueue.NewOrdered[int]() // équivaut à less = (a < b)
```

---

## 3. Documentation testable (`Example`)

Chaque package fournit des fonctions `Example…` avec un commentaire `// Output:`.
Elles ont **trois rôles** :

1. elles apparaissent dans la **godoc** (`go doc`, pkg.go.dev) ;
2. elles sont **exécutées et vérifiées** par `go test` (la sortie doit
   correspondre, sinon le test échoue) ;
3. elles servent de **mode d'emploi** qui ne peut pas se périmer en silence.

```go
func ExampleCache() {
    c := lru.New[string, int](2)
    c.Put("a", 1); c.Put("b", 2)
    c.Get("a")    // « a » redevient récent ; « b » devient le plus ancien
    c.Put("c", 3) // capacité dépassée : « b » est évincé
    fmt.Println(c.Keys())
    // Output: [c a]
}
```

---

## 4. Tests, benchmarks, fuzzing

```bash
cd projets/4-lib-generique
make test    # go test -race ./...  (tables + Example)
make bench   # benchmarks + -benchmem
make fuzz    # fuzze le cache LRU (FUZZTIME=30s par défaut)
make cover   # couverture + rapport HTML
```

- **Tests table-driven** pour chaque structure, plus un test de **propriété**
  pour le tas (sur 1000 entrées aléatoires, `Pop` rend toujours un ordre
  croissant).
- **Benchmarks** avec la boucle `for b.Loop()` (Go 1.24) et `-benchmem` :

  ```
  BenchmarkAdd-8       193697678   5.842 ns/op   0 B/op   0 allocs/op
  BenchmarkPushPop-8    63729559  18.240 ns/op   0 B/op   0 allocs/op
  BenchmarkPutGet-8     17582498  67.620 ns/op  64 B/op   2 allocs/op
  ```

- **Fuzzing du LRU** (`FuzzCache`) — le cœur du projet. On dérive une suite
  d'opérations d'octets aléatoires et on confronte le cache à un **modèle de
  référence** naïf (`slice + map`) : tant qu'ils restent d'accord (mêmes
  `Get`, même taille, `Len ≤ capacité`), l'implémentation est correcte.

  ```go
  f.Fuzz(func(t *testing.T, ops []byte) {
      c := lru.New[byte, byte](4)
      model := newRefLRU(4)
      // … rejoue (clé, valeur, get?) sur les deux, compare à chaque pas …
  })
  ```

  > 🧪 Le fuzzing trouve des **séquences** d'opérations qu'on n'aurait pas
  > écrites à la main. Un échec est **enregistré** dans `testdata/fuzz/` et
  > rejoué automatiquement aux exécutions suivantes : la régression devient un
  > test unitaire permanent.

---

## 5. SemVer & compatibilité

Une bibliothèque, ça se **maintient sans casser ses utilisateurs**.

- **Versionnage sémantique** : `vMAJEUR.MINEUR.CORRECTIF`.
  - **CORRECTIF** (`v1.2.3 → v1.2.4`) : correction sans changement d'API.
  - **MINEUR** (`→ v1.3.0`) : **ajout** rétrocompatible (nouvelle fonction, type).
  - **MAJEUR** (`→ v2.0.0`) : **rupture** d'API. En Go, une v2+ change le **chemin
    d'import** (`example.com/gends/v2`) — l'ancien et le nouveau coexistent.
- **La surface exportée est le contrat.** Renommer/supprimer un identifiant
  exporté, changer une signature, durcir une contrainte = **rupture**. Préférer
  **ajouter** (nouvelle fonction) à **modifier**.
- **Déprécier en douceur** : garder l'ancien symbole avec un commentaire
  `// Deprecated: utiliser X.` plutôt que de le retirer en mineure.
- **Promesse de compatibilité Go** : le langage et la stdlib ne cassent pas le
  code existant ; viser le même standard pour sa propre lib.

---

## 6. Intégration continue (CI)

`make ci` enchaîne `fmt → vet → test -race`. Le même pipeline en **GitHub
Actions** (matrice de versions Go) :

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go: ["1.27"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "${{ matrix.go }}" }
      - run: gofmt -l . && test -z "$(gofmt -l .)"
      - run: go vet ./...
      - run: go test -race ./...
```

> La **matrice** teste plusieurs versions de Go : c'est ainsi qu'on tient la promesse
> « marche sur Go ≥ X » sans le découvrir chez un utilisateur.

> ⚠️ **Cette matrice s'est réduite à une seule version, et c'est instructif.** Elle listait
> `["1.25", "1.26"]` ; depuis que `set.Map` est une **méthode générique** (§6 bis), la
> bibliothèque exige **Go 1.27** — `go.mod` déclare `go 1.27`, et un build en 1.26 échoue
> immédiatement. La matrice ne peut donc plus tester qu'une version. C'est le **coût réel**
> d'avoir adopté une nouveauté de langage dans une bibliothèque publique, et la matrice est
> précisément l'endroit où ce coût devient visible — avant la publication, pas après. Pour
> continuer à couvrir 1.26, il faudrait sortir `Map` en fonction libre.

---

## 6 bis. Méthode générique : `set.Map` (🆕 Go 1.27)

Jusqu'à Go 1.26, une méthode ne pouvait utiliser que les paramètres de type de son
**récepteur**. Un `Map` était donc impossible en méthode : sa sortie change de type, et ce
nouveau type n'avait nulle part où être déclaré. On se rabattait sur une fonction libre —
`set.Map(s, f)` — qui se lit à l'envers et casse le chaînage.

Depuis **Go 1.27**, la méthode déclare son propre paramètre :

```go
// set/set.go
func (s *Set[T]) Map[U comparable](f func(T) U) *Set[U] {
	out := &Set[U]{m: make(map[U]struct{}, s.Len())}
	for k := range s.m {
		out.m[f(k)] = struct{}{}
	}
	return out
}
```

```go
noms := set.Of(1, 2, 3).
	Filter(func(i int) bool { return i%2 == 1 }).
	Map(strconv.Itoa) // Set[string]{"1", "3"}
```

Trois observations de conception, qui sont l'intérêt du sujet dans une **bibliothèque** :

1. **Le changement de type justifie le paramètre, pas la commodité.** `Filter` renvoie un
   `*Set[T]` : elle n'a aucune raison d'être générique. Ajouter un paramètre de type « au cas
   où » alourdit la signature sans rien apporter.
2. **`Sorted` reste une fonction libre — pour une autre raison.** Elle exige `cmp.Ordered`,
   une contrainte **plus forte** que le `comparable` du récepteur. Un paramètre de méthode ne
   résout pas ce problème-là : une méthode ne peut pas durcir la contrainte de son récepteur.
   Les deux motifs coexistent donc, et il faut savoir lequel s'applique.
3. **`Map` peut réduire l'ensemble.** Deux éléments distincts dont `f` renvoie la même image
   fusionnent : `Of(-1, 1).Map(abs)` a **un** élément. C'est à documenter, pas à corriger.

⚠️ Une méthode générique ne peut ni figurer dans une **interface**, ni en satisfaire une. Si
la bibliothèque doit un jour exposer un contrat `interface{ Map(...) }`, `Map` devra
redevenir une fonction libre — un point à peser **avant** de publier, puisque le retirer
serait une rupture d'API.

---

## 7. Points de vigilance

- **Contrainte au plus juste** : commencer par `any`/`comparable` et ne durcir
  (`cmp.Ordered`, comparateur) **que** lorsqu'une opération l'exige. Durcir plus
  tard est une rupture ; assouplir ne l'est pas.
- **Ne pas retenir de mémoire** : `pqueue.Pop` et l'éviction LRU remettent la
  case à zéro (`var zero T`) pour ne pas garder en vie des objets pointés
  ([Ch. 27 GC](../../chapitres/27-garbage-collector.md)).
- **Méthode générique = plancher de version.** `set.Map` (1.27) fait passer l'exigence
  minimale de la bibliothèque à Go 1.27. Pour une bibliothèque publique, relever ce plancher
  est un **choix éditorial** : il exclut les utilisateurs restés sur une version antérieure.
  À arbitrer explicitement, pas à subir au détour d'un refactoring.
- **`container/list` n'est pas générique** : d'où les assertions
  `el.Value.(*entry[K,V])` dans `lru`. C'est le prix d'un conteneur stdlib
  antérieur aux génériques ; l'API publique, elle, reste typée.
- **Sûreté concurrente non fournie** : documenté explicitement. Mieux vaut une
  structure simple que l'utilisateur protège, qu'un verrou imposé à tous.
- **Exemples déterministes** : l'itération d'un `Set`/d'une map est aléatoire ;
  les `Example` passent par `Sorted` ou par un ordre garanti (récence du LRU).

---

## 8. Pour aller plus loin

- `pqueue` avec mise à jour de priorité (`Fix`), un `lru.Cache` à **TTL** ou avec rappel
  d'éviction (`OnEvict func(K, V)`).
- Donner à `lru.Cache` une méthode générique `MapValues[U any](f func(V) U) *Cache[K, U]`,
  sur le modèle de `set.Map` — et se demander, à chaque fois, si le changement de type
  justifie vraiment le paramètre.
- Variante **thread-safe** dans un sous-package `lru/sync` enveloppant le cache
  d'un `sync.Mutex` (ou `sync.RWMutex`).
- Publier la godoc sur **pkg.go.dev** et viser un score complet (doc de chaque
  symbole exporté, exemples).
- Profiler le LRU (Projet 7) pour réduire les 2 allocations/op de `Put`.

---

## 📌 À retenir

- Une **bibliothèque** = une **API** pensée pour autrui : type zéro explicite,
  contraintes minimales, pas d'effet de bord surprenant.
- **Génériques** : exiger le strict nécessaire. Deux motifs distincts de fonction libre —
  une contrainte **plus forte** que celle du récepteur (`set.Sorted`, toujours d'actualité),
  et un **type de sortie nouveau** (résolu par la méthode générique de 🆕 1.27, `set.Map`).
- Les **`Example`** documentent **et** testent ; les **benchmarks** chiffrent ;
  le **fuzzing** + modèle de référence éprouvent les invariants.
- **SemVer** : ajouter plutôt que modifier ; une v2 change le chemin d'import.
- Une **CI** (fmt + vet + test -race, en matrice) tient la promesse de
  compatibilité.
