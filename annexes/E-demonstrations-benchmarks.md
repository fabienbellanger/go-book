# Annexe E — Démonstrations techniques & benchmarks

> **Objectif** — Mettre des **chiffres** sur les intuitions de performance du
> livre. Six micro-études, chacune avec son code, son benchmark et son tableau de
> résultats **mesurés**, pour ancrer des réflexes : allouer sur la pile, préférer
> l'atomique au verrou quand c'est possible, réserver la capacité, et ne jamais
> _supposer_ — toujours **mesurer**.

---

Tout le code de cette annexe vit dans `code/annexe-E-benchmarks/`. Pour tout
relancer :

```bash
cd code
go test -run='^$' -bench=. -benchmem -count=6 ./annexe-E-benchmarks/
go build -gcflags="-m" -o /dev/null ./annexe-E-benchmarks/   # escape analysis
```

> ⚠️ **Les chiffres absolus ci-dessous n'ont aucune valeur universelle.** Ils ont
> été relevés sur **Apple M3, Go 1.26, `arm64`** (`b.Loop` 🆕 1.24, `-benchmem`).
> Sur votre machine, ils différeront — et c'est sans importance : ce qui compte
> est le **rapport** entre les deux versions, pas la valeur brute. Reproduisez-les
> chez vous avec `benchstat` (🔁 Ch. 36).

---

## 1. Pile vs tas (escape analysis)

Une valeur dont la durée de vie ne dépasse pas l'appel reste sur la **pile**
(libérée gratuitement au retour). Si son adresse _fuit_ (on renvoie un pointeur),
elle doit survivre : le compilateur la **fait migrer vers le tas**, à la charge
du ramasse-miettes. C'est l'**escape analysis** (🔁 Ch. 26).

```go
func sumOnStack(a, b int) int {
	p := point{a, b} // usage local => reste sur la pile
	return p.x + p.y
}

func newOnHeap(a, b int) *point {
	p := point{a, b}
	return &p // l'adresse fuit => escapes to heap
}
```

On le **vérifie**, sans deviner, avec `-gcflags="-m"` :

```bash
$ go build -gcflags="-m" -o /dev/null ./annexe-E-benchmarks/
annexe-E-benchmarks/escape.go:26:2: moved to heap: p
```

| Fonction     | ns/op | B/op | allocs/op |
| ------------ | ----: | ---: | --------: |
| `sumOnStack` | ~1,75 |    0 |     **0** |
| `newOnHeap`  |  ~8,5 |   16 |     **1** |

> 💡 **À retenir** : ~5× plus cher et **une allocation** par appel, juste pour
> avoir renvoyé un pointeur. Renvoyer une **valeur** (quand elle est petite) évite
> souvent le tas. `-gcflags="-m"` est l'outil qui tranche.

---

## 2. Mutex vs atomic (compteur sous contention)

Protéger un compteur partagé par un `sync.Mutex` met les goroutines en attente
quand le verrou est pris ; une opération **atomique** (`atomic.Int64`) s'appuie
sur une instruction matérielle, sans verrou (🔁 Ch. 21). On mesure sous
contention avec `b.RunParallel` :

```go
func BenchmarkAtomicCounter(b *testing.B) {
	var c atomicCounter
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc() // c.n.Add(1)
		}
	})
}
```

| Compteur (8 P, contention) | ns/op | allocs/op |
| -------------------------- | ----: | --------: |
| `sync.Mutex`               | ~82,0 |         0 |
| `atomic.Int64`             | ~18,7 |         0 |

> ⚡ L'atomique est ici **~4,4× plus rapide**. Mais attention : l'atomique ne
> convient qu'à des opérations **simples** (incrément, échange, CAS). Dès qu'il
> faut protéger _plusieurs_ champs de façon cohérente, le mutex reste la bonne
> réponse — la simplicité prime sur la micro-optimisation.

---

## 3. Interface vs générique (dispatch)

On appelle la **même méthode** `Double(int) int` de deux façons : à travers une
**interface** (dispatch dynamique via la table de méthodes) et à travers un
**type paramétré** instancié sur le type concret (🔁 Ch. 11, Ch. 33).

| Appel (boucle sur 1024 entiers) | ns/op | allocs/op |
| ------------------------------- | ----: | --------: |
| via interface (`viaInterface`)  |  ~498 |         0 |
| via générique (`viaGeneric`)    |  ~823 |         0 |

> ⚠️ **Surprise instructive : ici, le générique est _plus lent_** (~1,65×). Les
> génériques Go sont compilés par _stenciling_ par **forme de GC** : un type
> paramétré sur un type pointeur/interface passe par un **dictionnaire**
> d'exécution, qui peut **empêcher l'inlining** et ajouter une indirection. La
> leçon vaut de l'or : **les génériques ne sont _pas_ une optimisation de
> performance**. On les choisit pour la **réutilisation** et la **sûreté de
> typage** ; pour la vitesse pure d'un point chaud, on mesure, et parfois le code
> monomorphisé à la main (ou une interface) gagne.

---

## 4. Concaténation de chaînes

Les chaînes sont **immuables** : `s += x` en boucle alloue une **nouvelle**
chaîne à chaque tour — un coût en O(n²). `strings.Builder` accumule dans un
tampon d'octets réutilisé ; avec `Grow` (capacité réservée), il vise **une seule**
allocation (🔁 Ch. 31).

```go
func concatBuilderGrow(parts []string) string {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	var b strings.Builder
	b.Grow(total) // une réservation, plus de redimensionnement
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}
```

| Méthode (512 fragments)  |   ns/op |     B/op | allocs/op |
| ------------------------ | ------: | -------: | --------: |
| `+=` en boucle           | ~83 000 | ~1 117 k |       511 |
| `strings.Builder`        |  ~2 150 |  ~12 500 |        12 |
| `strings.Builder`+`Grow` |  ~2 100 |   ~4 096 |     **1** |

> ⚡ Le `Builder` est **~40× plus rapide** que `+=`, et avec `Grow` on tombe à
> **une seule allocation** (×273 de mémoire en moins). C'est l'un des gains les
> plus rentables et les plus simples du langage.

---

## 5. Préallocation d'un slice

`append` fait croître la capacité par **doublements** : chaque palier réalloue et
**recopie** tout. Si la taille finale est connue, `make([]T, 0, n)` la réserve
d'un coup (🔁 Ch. 30).

| Construction de 10 000 entiers |   ns/op |    B/op | allocs/op |
| ------------------------------ | ------: | ------: | --------: |
| `append` sur slice `nil`       | ~31 300 | 357 626 |        19 |
| `make([]int, 0, n)` + `append` |  ~6 230 |  81 920 |     **1** |

> 💡 **~5× plus rapide, une seule allocation** au lieu de 19. Les 19 allocations
> correspondent aux paliers successifs de croissance ; la préallocation les
> supprime toutes.

---

## 6. Préallocation d'une map

Même principe pour les maps : indiquer la taille attendue à `make(map[K]V, n)`
réduit les redimensionnements internes (rehash) — d'autant que Go 1.24 a adopté
les **Swiss Tables** (🔁 Ch. 32).

| Construction de 10 000 entrées |    ns/op |    B/op | allocs/op |
| ------------------------------ | -------: | ------: | --------: |
| `make(map[int]int)`            | ~258 700 | 591 675 |        81 |
| `make(map[int]int, n)`         |  ~79 100 | 295 601 |        34 |

> ⚡ **~3,3× plus rapide** et **2× moins de mémoire**. Le gain est réel mais moins
> spectaculaire que pour les slices : une map reste plus coûteuse (hachage,
> gestion des collisions) qu'un tableau contigu.

---

## 7. Régler le GC : `GOGC` et `GOMEMLIMIT`

Le ramasse-miettes Go n'a (presque) pas de boutons — deux suffisent (🔁 Ch. 27) :

- **`GOGC`** (défaut `100`) : déclenche un cycle quand le tas a crû de `GOGC %`
  depuis le dernier. `GOGC=200` ⇒ GC moins fréquent, plus de mémoire, moins de
  CPU passé en GC. `GOGC=off` ⇒ désactive le GC (à réserver aux outils courts).
- **`GOMEMLIMIT`** (🆕 1.19) : une **limite souple** de mémoire totale. Sous
  pression, le runtime collecte plus agressivement pour ne pas la dépasser —
  idéal en conteneur pour éviter l'_OOM kill_.

On **observe** le GC avec `GODEBUG=gctrace=1`, qui imprime une ligne par cycle :

```bash
$ GODEBUG=gctrace=1 go test -run='^$' -bench=BenchmarkMapNoPrealloc -benchtime=200x ./annexe-E-benchmarks/
gc 1 @0.006s 1%: 0.056+0.49+0.033 ms clock, ... 3->4->0 MB, 4 MB goal, ... 8 P
gc 2 @0.010s 2%: 0.043+0.30+0.042 ms clock, ... 3->3->1 MB, 4 MB goal, ... 8 P
```

Lecture : `3->4->0 MB` = tas **avant→pendant→après** le cycle ; `4 MB goal` = seuil
de déclenchement (piloté par `GOGC`) ; `1%`, `2%` = part cumulée du temps passée
en GC. C'est qualitatif, mais c'est le premier réflexe pour savoir si le GC est un
problème — avant de toucher au moindre réglage.

> ⚠️ **Ne réglez pas `GOGC`/`GOMEMLIMIT` à l'aveugle.** Le meilleur « tuning GC »
> est presque toujours d'**allouer moins** (sections 1, 4, 5, 6). On ajuste ces
> variables seulement après avoir mesuré, profil mémoire à l'appui (🔁 Ch. 27, 37).

---

## 📌 À retenir

- **Mesurer, pas supposer** : `go test -bench -benchmem`, `b.Loop` (1.24),
  `benchstat`, et `-gcflags="-m"` pour l'escape analysis.
- **Allouer sur la pile** quand on peut (renvoyer une valeur, pas un pointeur) :
  zéro allocation, zéro travail pour le GC.
- **`atomic` ≫ `mutex`** pour un simple compteur (~4×) ; le mutex reste roi dès
  qu'il faut une cohérence multi-champs.
- **Les génériques ne sont pas plus rapides** — ils peuvent même être plus lents
  (dictionnaires). On les choisit pour la réutilisation, pas la vitesse.
- **Réserver la capacité** (`Grow`, `make(..., n)`) supprime les recopies de
  croissance : gains de 5× à 40× pour quasiment aucun effort.
- **Le meilleur réglage du GC, c'est moins d'allocations** ; `GOGC`/`GOMEMLIMIT`
  ne viennent qu'après la mesure.
