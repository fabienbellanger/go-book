# 35 — `unsafe` & interopérabilité bas niveau

> **Objectif** — Contourner le typage **en connaissance de cause** : `unsafe.Pointer` et ses **règles de
> conversion** (les patterns valides), `Sizeof`/`Alignof`/`Offsetof`, l'**alignement** et le **padding**,
> `unsafe.Slice`/`unsafe.String` (zéro-copie), `//go:linkname` ; et l'**interop** avec C (**cgo**) et le
> **SIMD**, avec leurs nouveautés Go 1.26.
>
> **Prérequis** — [Ch. 8](08-structs-methodes.md), [Ch. 33](33-interfaces-profondeur.md)

---

## Introduction

Go est un langage **à sûreté mémoire** : pas d'arithmétique de pointeur, pas de réinterprétation de
types. Cette sûreté repose sur un fait précis : le compilateur et le GC connaissent, pour **chaque**
valeur, son **type exact** — c'est ce qui permet au GC de savoir où se trouvent les pointeurs vivants (et
donc ce qu'il doit conserver) et au vérificateur de types de refuser de mélanger un `*int` et un
`*string`. Le package **`unsafe`** ouvre une porte de sortie — pour le zéro-copie, l'interop C, ou parler
au matériel — en court-circuitant précisément ce typage : dès qu'un `unsafe.Pointer` entre en jeu, le
compilateur **perd la capacité de suivre la valeur**, et c'est exactement ce que vous lui retirez en
échange du contrôle bas niveau. Le prix : **vous** devenez responsable de la correction, et vous perdez la
garantie de portabilité et de compatibilité. Ce chapitre **clôt la Partie V** ; il s'utilise avec
parcimonie et **toujours** mesure à l'appui. Code dans [`code/ch35-unsafe-cgo/`](../code/ch35-unsafe-cgo/).

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

`unsafe.Pointer` est le **pointeur universel** : il convertit entre types de pointeurs, en cassant
volontairement le typage statique. La règle d'or : un **`uintptr` n'est PAS une référence** — c'est un
simple entier, que le GC **ne suit pas**. Deux mécanismes concrets rendent un `uintptr` stocké dangereux :

- Le GC ne marque **pas** un objet comme vivant via un `uintptr` qui pointe vers lui : si plus aucun
  `unsafe.Pointer`/pointeur typé ne le référence, le GC peut le **libérer** pendant que le `uintptr`
  pointe encore vers son ancien emplacement — un accès ultérieur lit (ou écrit) de la mémoire déjà
  réutilisée.
- La **pile d'une goroutine peut être déplacée** : quand elle croît, le runtime alloue une pile plus
  grande, **copie** son contenu et **réécrit** tous les pointeurs qui y font référence
  ([Ch. 26](26-allocation-escape.md)). Un `unsafe.Pointer` est réécrit avec elle ; un `uintptr` stocké à
  part **ne l'est pas** — il continue de désigner l'ancienne adresse, devenue invalide.

```
  avant croissance de pile           après croissance (copie + réécriture)
  +-------------------+              +-----------------------------+
  | x  @ 0xc0001000   |   copie      | x  @ 0xc0002000 (nouvelle    |
  +-------------------+  -------->   |    adresse, pile agrandie)   |
                                      +-----------------------------+
  unsafe.Pointer(&x)  -> réécrit automatiquement vers 0xc0002000 (le runtime suit le Pointer)
  uintptr(unsafe.Pointer(&x)) -> reste 0xc0001000 : adresse PERIMEE, lecture = comportement indéfini
```

C'est pour cela que la conversion `unsafe.Pointer` → `uintptr` → `unsafe.Pointer` n'est valide que **dans
la même expression**, sans appel de fonction entre les deux (rien ne doit pouvoir déplacer l'objet entre
la conversion et la reconversion). Les **patterns valides** :

1. **`*T1` → `unsafe.Pointer` → `*T2`** : réinterpréter, si les **layouts sont compatibles**.
2. **Arithmétique** via **`unsafe.Add(ptr, offset)`** (1.17) : sucre syntaxique sûr pour le pattern brut
   `unsafe.Pointer(uintptr(ptr) + offset)` **dans une seule expression** — préférez `unsafe.Add`, qui
   élimine le risque de fractionner ce calcul sur plusieurs lignes (et donc de violer la règle ci-dessus).
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
modifié après l'appel (la string le suppose immuable). La portée du risque dépasse la variable locale :
si cette string sert de **clé de map**, son hachage est calculé une fois à l'insertion ; la muter ensuite
désynchronise le hachage de son contenu et corrompt silencieusement la map (recherches qui échouent,
doublons apparents), sans jamais paniquer. Une seule entorse = corruption silencieuse, difficile à
rejouer en débogage. À réserver aux chemins **mesurés** où la copie domine.

## `//go:linkname` (mention)

La directive **`//go:linkname locale distante`** lie un symbole local à un symbole **non exporté** d'un
autre package (souvent `runtime`). C'est ainsi que certaines bibliothèques accèdent à `runtime.nanotime`
ou à des entrailles. Très **fragile** (casse entre versions), à éviter sauf nécessité absolue. Depuis
**Go 1.23**, le linker distingue deux usages : en **push** — le package qui définit le symbole l'expose
lui-même via son propre `//go:linkname` — toujours autorisé ; en **pull** — un package tiers vise un
symbole interne qui ne s'est pas déclaré exportable ainsi — désormais **bloqué** pour tout nouveau
symbole (les usages déjà recensés dans l'écosystème open source restent tolérés, pour ne pas casser
l'existant). Le flag du linker `-checklinkname=0` désactive ce contrôle, à réserver au débogage.

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

Le coût d'un appel cgo n'est **pas** celui d'un appel Go : il traverse une frontière entre deux mondes
d'exécution qui ne partagent ni pile ni convention d'appel.

```
  pile Go (goroutine g, gérée par le runtime)      pile C (système, taille fixe)
  +--------------------------+                     +------------------------+
  | appelant Go               |                    |                        |
  +--------------------------+   runtime.cgocall    |                        |
  | C.sqrt(x)                 | ------------------> |   sqrt(x)  (code C)   |
  +--------------------------+  1. bascule de pile   +------------------------+
              |                  2. M marqué « en syscall » (comme un appel
              |                     système, Ch. 28) : son P est libéré, un
              |                     autre M peut l'utiliser pendant ce temps
              v                  3. exécution C, hors du contrôle du GC/de
  +--------------------------+      l'ordonnanceur Go
  | reprise après C.sqrt(x)   | <------------------ |  retour de sqrt(x)    |
  +--------------------------+   4. ré-acquisition d'un P avant de continuer
```

Cette traversée a un coût **fixe, par appel**, même pour un `sqrt` trivial : changement de pile, marquage/
démarquage du M, et **perte d'inlining** — une fonction Go qui appelle du C n'est **jamais inlinée** (le
compilateur ne sait pas inliner un appel qui change de monde d'exécution). On **regroupe** donc le
travail côté C plutôt que de multiplier les allers-retours : un appel qui traite 10 000 éléments bat
10 000 appels qui en traitent un chacun.

cgo impose aussi des **règles de passage de pointeurs** vérifiées à l'exécution : un pointeur Go passé à
C ne doit pas pointer vers de la mémoire contenant **elle-même** des pointeurs Go, et le code C ne doit
**pas conserver** ce pointeur au-delà de l'appel — le GC doit pouvoir continuer à localiser tous les
pointeurs Go, y compris ceux momentanément visibles côté C. Violer cette règle ne se voit pas à la
compilation : le runtime la vérifie via `GODEBUG=cgocheck` (`1` par défaut — coût faible ; `2` —
vérification exhaustive ; `0` — désactivé) et **plante** le programme avec un diagnostic s'il détecte une
infraction.

cgo complique enfin le build (compilateur C requis, `CGO_ENABLED=1`) et la cross-compilation — la
compilation croisée native décrite au [Ch. 1](01-installation-toolchain.md) bascule sur `CGO_ENABLED=0`
dès que `GOOS`/`GOARCH` diffère de la machine hôte, donc **sans cgo** — d'où la préférence Go pour le
**tout-Go** quand c'est possible.

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
