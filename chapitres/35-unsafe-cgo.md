# Ch. 35 — `unsafe` & interopérabilité bas niveau

> **Objectif** — Contourner le typage **en connaissance de cause** : `unsafe.Pointer` et ses **règles de
> conversion** (les patterns valides), `Sizeof`/`Alignof`/`Offsetof`, l'**alignement** et le **padding**,
> `unsafe.Slice`/`unsafe.String` (zéro-copie), `//go:linkname` ; et l'**interop** avec C (**cgo**) et le
> **SIMD**, avec leurs nouveautés Go 1.26.
>
> **Prérequis** — [Ch. 8](08-structs-methodes.md), [Ch. 33](33-interfaces-profondeur.md)

---

## Introduction

Go est un langage **à sûreté mémoire** : pas d'arithmétique de pointeur, pas de réinterprétation de
types. Le package **`unsafe`** ouvre une porte de sortie — pour le zéro-copie, l'interop C, ou parler au
matériel. Le prix : **vous** devenez responsable de la correction, et vous perdez la garantie de
portabilité et de compatibilité. Ce chapitre **clôt la Partie V** ; il s'utilise avec parcimonie et
**toujours** mesure à l'appui. Code dans [`code/ch35-unsafe-cgo/`](../code/ch35-unsafe-cgo/).

---

## Alignement & padding

Le compilateur **aligne** chaque champ sur son alignement naturel (un `int64` à une adresse multiple de
8). Résultat : l'**ordre des champs** change la taille de la struct, à cause du **padding** inséré.

```
  type Padded struct { a bool; b int64; c bool }   -> 24 octets

  offset:  0                    8                16
          +----+----------------+----------------+----+----------------+
          | a  |  padding (7 o) |       b        | c  | padding (7 o)  |
          +----+----------------+----------------+----+----------------+
           bool                  int64 (aligné 8)  bool

  type Packed struct { b int64; a bool; c bool }   -> 16 octets
          +----------------+----+----+----------------+
          |       b        | a  | c  | padding (6 o)  |
          +----------------+----+----+----------------+
```

```go
// code/ch35-unsafe-cgo/unsafex.go : Sizeof, Alignof, Offsetof
unsafe.Sizeof(Padded{})    // 24
unsafe.Sizeof(Packed{})    // 16  -> 8 octets économisés en réordonnant
unsafe.Offsetof(p.b)       // 8
unsafe.Alignof(int64(0))   // 8
```

📌 **Réordonnez les champs du plus grand alignement au plus petit** pour des structs compactes — gain
direct de mémoire et de cache sur de gros volumes. ⚠️ Ces fonctions sont des **constantes de
compilation** : aucun coût à l'exécution.

## `unsafe.Pointer` : les règles du jeu

`unsafe.Pointer` est le **pointeur universel** : il convertit entre types de pointeurs. La règle d'or :
un **`uintptr` n'est PAS une référence** — le GC ne le suit pas. Stocker un `uintptr` puis le
reconvertir est un **bug** (l'objet a pu être déplacé ou collecté). Les **patterns valides** :

1. **`*T1` → `unsafe.Pointer` → `*T2`** : réinterpréter, si les **layouts sont compatibles**.
2. **Arithmétique** via **`unsafe.Add(ptr, offset)`** (1.17) — jamais de calcul manuel sur `uintptr`.
3. **`unsafe.Slice(ptr, n)`** / **`unsafe.String(ptr, n)`** : bâtir un slice/une string depuis un pointeur.
4. Contrats documentés de **`syscall`** et **`reflect`** (`Value.Pointer`, `UnsafeAddr`).

```go
// code/ch35-unsafe-cgo/unsafex.go : arithmétique contrôlée
func SecondElem(arr *[4]int32) int32 {
	base := unsafe.Pointer(arr)
	second := (*int32)(unsafe.Add(base, unsafe.Sizeof(arr[0]))) // base + 4 octets
	return *second
}
```

## Zéro-copie : `unsafe.Slice` & `unsafe.String`

La conversion `string`↔`[]byte` **copie** ([Ch. 31](31-strings-profondeur.md)). `unsafe` permet de
**partager le backing** — zéro copie, zéro allocation :

```go
// code/ch35-unsafe-cgo/unsafex.go
func BytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b)) // partage le backing de b
}
```

Mesuré :

| Conversion                   | ns/op     | B/op  | allocs/op |
| ---------------------------- | --------- | ----- | --------- |
| `string(b)` (sûre, copie)    | **21,07** | 80    | **1**     |
| `unsafe.String` (zéro-copie) | **3,23**  | **0** | **0**     |

**6,5× plus rapide, 0 allocation.** ⚠️ Le contrat : le `[]byte` source ne doit **plus jamais** être
modifié (la string le suppose immuable). Une seule entorse = corruption silencieuse. À réserver aux
chemins **mesurés** où la copie domine.

## `//go:linkname` (mention)

La directive **`//go:linkname locale distante`** lie un symbole local à un symbole **non exporté** d'un
autre package (souvent `runtime`). C'est ainsi que certaines bibliothèques accèdent à `runtime.nanotime`
ou à des entrailles. Très **fragile** (casse entre versions), à éviter sauf nécessité absolue ; Go 1.23+
**restreint** d'ailleurs son usage vers le runtime.

## Interopérabilité C : cgo

**cgo** appelle du code **C** depuis Go. Un commentaire spécial avant `import "C"` déclare le C ; les
symboles deviennent `C.fonction`, `C.type` :

```go
/*
#include <math.h>
*/
import "C"
// résultat := float64(C.sqrt(C.double(2))) // appel de la libm
```

Le coût d'un appel cgo n'est **pas** celui d'un appel Go : changement de pile, barrière pour
l'ordonnanceur ([Ch. 28](28-ordonnanceur-gmp.md)). On **regroupe** donc le travail côté C plutôt que de
multiplier les allers-retours. cgo complique aussi le build (compilateur C requis, `CGO_ENABLED=1`) et la
cross-compilation — d'où la préférence Go pour le **tout-Go** quand c'est possible.

> Le code de ce chapitre reste **sans cgo** (aucune dépendance C) pour que `go test ./...` passe partout.

---

## 🆕 Go 1.2x

- **1.20** — **`unsafe.SliceData`**, **`unsafe.StringData`**, **`unsafe.String`** complètent
  `unsafe.Slice`/`unsafe.Add` (1.17) : conversions pointeur↔slice↔string **sans** bricolage de `uintptr`.
- **1.26** — les **appels cgo sont ~30 % plus rapides** (d'après les notes de version) : l'interop C
  devient moins pénalisante.
- **1.26** — **SIMD expérimental** : avec `GOEXPERIMENT=simd`, la bibliothèque expose `simd/archsimd`
  (types vectoriels `Float32x8`, `Int8x16`… — **AMD64** pour l'instant). Hors promesse de compatibilité Go 1.
- **1.26** — **`runtime/secret`** (expérimental, `GOEXPERIMENT=runtimesecret`) : zones mémoire pour
  données sensibles. Ces deux expériences sont **désactivées par défaut** (vérifié sur 1.26.4).

## ⚠️ Pièges

- **Stocker un `uintptr`** puis le reconvertir en pointeur — le GC ne le suit pas : l'objet a pu bouger.
  Utilisez `unsafe.Add` dans **la même expression**.
- **Modifier le `[]byte`** après `unsafe.String` — viole l'immutabilité ; corruption indétectable.
- **Supposer un layout** — taille/offsets dépendent de l'**architecture** (32 vs 64 bits, alignement).
  Calculez avec `Sizeof`/`Offsetof`, ne codez pas « 8 » en dur.
- **`unsafe` pour gagner sans preuve** — la plupart du temps, le code sûr est assez rapide. Mesurez **avant**.
- **cgo par confort** — il alourdit build, portabilité et profiling ; cherchez une solution pure Go d'abord.

## ⚡ Performance

- **Réordonner les champs** (grand → petit alignement) réduit la taille des structs **sans** `unsafe`.
- **`unsafe.String`/`Slice`** éliminent la copie sur un chemin chaud **prouvé** — sous le contrat
  d'immutabilité.
- **Regroupez** le travail cgo : un gros appel vaut mieux que mille petits (barrière d'ordonnanceur).
- 🔁 [Ch. 31](31-strings-profondeur.md) (conversions), [Ch. 26](26-allocation-escape.md) (allocations),
  [Ch. 37](37-profiling-pprof.md) (prouver le gain).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch35-unsafe-cgo
go test ./ch35-unsafe-cgo/...
go test -bench=. -benchmem -run=^$ ./ch35-unsafe-cgo/...
```

À essayer :

1. Ajoutez un champ `int16` à `Padded`/`Packed` et prédisez les nouvelles tailles avant de mesurer.
2. Vérifiez avec `Offsetof` qu'un `struct{ a int32; b int64 }` aligne `b` sur 8 (4 octets de padding).
3. Écrivez `go build "-gcflags=-m"` et constatez que `Sizeof` & co. n'apparaissent pas (constantes).

---

## 📌 À retenir

- L'**ordre des champs** change la taille via le **padding** ; réordonnez **grand → petit** alignement.
  `Sizeof`/`Alignof`/`Offsetof` sont des **constantes** de compilation.
- **`unsafe.Pointer`** réinterprète les pointeurs ; un **`uintptr` n'est pas suivi par le GC**.
  Arithmétique via **`unsafe.Add`**, dans la même expression.
- **`unsafe.String`/`unsafe.Slice`** offrent le **zéro-copie** (6,5× ici) — sous le contrat d'immutabilité
  du backing.
- **cgo** appelle du C (`import "C"`) mais coûte (pile, ordonnanceur, build) ; **1.26** le rend ~30 %
  plus rapide. Regroupez les appels.
- **SIMD** (`simd/archsimd`) et **`runtime/secret`** sont **expérimentaux** (1.26, off par défaut, hors
  compat Go 1). `unsafe` : avec **parcimonie** et **mesure**.

## 🔁 Pour aller plus loin

- [Ch. 8 — Structs & méthodes](08-structs-methodes.md) : layout, padding côté usage.
- [Ch. 31 — Strings en profondeur](31-strings-profondeur.md) : la conversion sûre que `unsafe` court-circuite.
- [Ch. 33 — Interfaces en profondeur](33-interfaces-profondeur.md) : réinterpréter `eface`/`iface` (à vos risques).
- [Ch. 37 — Profiling pprof](37-profiling-pprof.md) : prouver qu'`unsafe` apporte vraiment le gain.
- Doc : `go doc unsafe` ; `go doc cmd/cgo` ; notes de version Go 1.26.
