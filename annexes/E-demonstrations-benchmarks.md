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

> 🔬 **Pourquoi `sinkPtr` ?** `bench_test.go` n'écrit jamais le résultat dans une
> variable locale : il l'assigne à une variable **de niveau package**
> (`sinkPtr`, `sinkInt`…). Sans ce « puits », le compilateur pourrait constater
> que le résultat ne sert jamais et **éliminer l'appel** (dead-code elimination) —
> le benchmark mesurerait alors une boucle vide. Cette assignation a un second
> effet, instructif : c'est elle qui **rend l'évasion réelle**.
> `-gcflags="-m"` confirme `p escapes to heap` exactement à la ligne
> `sinkPtr = newOnHeap(3, 4)` de `bench_test.go` — alors que dans `main.go`, le
> même appel `newOnHeap(3, 4).x` (immédiatement déréférencé, jamais conservé)
> **ne déclenche aucune ligne** « moved to heap » une fois inliné. Le verdict
> d'évasion ne dépend donc pas que de la fonction : il dépend de ce que
> **l'appelant** fait du pointeur reçu.

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

L'écart vient de ce que chaque mécanisme déclenche sous le capot. `c.n.Add(1)`
compile vers une **primitive atomique du processeur** : le cœur prend
brièvement le bus mémoire pour son incrément, sans jamais solliciter le
planificateur Go. `sync.Mutex` doit, lui, **coordonner des goroutines** :
en cas de contention, il commence par un court **spin actif** (quelques tours
sans céder le CPU — utile seulement si le verrou semble sur le point de se
libérer), puis **met en sommeil** la goroutine perdante via le sémaphore
d'exécution du runtime (`runtime_SemacquireMutex`) si la contention persiste —
un aller-retour par le planificateur bien plus coûteux qu'une instruction
matérielle. Passé **1 ms** d'attente, le mutex bascule même en **mode famine**
(« starvation mode ») : il désactive le spin et **transmet directement** la
propriété du verrou au prochain en file, pour garantir l'équité au prix du
débit (voir `internal/sync/mutex.go` dans les sources de Go).

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

`-gcflags="-m -m"` permet de voir **exactement** où passe le temps :

```bash
$ go build -gcflags="-m -m" -o /dev/null ./annexe-E-benchmarks/ 2>&1 | grep dispatch.go
annexe-E-benchmarks/dispatch.go:34:6: can inline viaGeneric[go.shape.struct {}] with cost 80 ...
annexe-E-benchmarks/dispatch.go:34:6: cannot inline viaGeneric[main.intDoubler]: function too complex: cost 87 exceeds budget 80
```

Le compilateur ne génère **qu'une seule** version de `viaGeneric`, partagée par
toutes les formes de GC identiques (`go.shape.struct {}` — toute structure vide,
quel que soit son nom) : elle reçoit un **dictionnaire** caché
(`*[2]uintptr`) décrivant `intDoubler` au runtime, et cette version-forme tient
de justesse sous le budget d'inlining (coût 80). La version pleinement
**monomorphisée** pour `intDoubler` — celle qui égalerait l'interface — ne
l'est pas (coût 87 > 80) : elle n'est jamais produite à ce prix. Conséquence
visible dans les traces du benchmark : l'appel `viaInterface` voit
`intDoubler.Double` **inliné jusqu'au bout** (« inlining call to
`intDoubler.Double` »), tandis que l'appel `viaGeneric` ne le voit **jamais** —
`Double` reste invoqué **indirectement à travers le dictionnaire**, à chaque
itération de la boucle. C'est cette indirection répétée, et non une différence
de nature algorithmique, qui coûte les ~325 ns mesurés.

> 💡 Autre détail révélé par `-m` : pour `viaInterface`, le diagnostic signale
> `parameter d leaks to {heap}` — et pourtant les deux benchmarks affichent
> **0 allocs/op**. Convertir une valeur de **taille nulle** (`intDoubler{}`) en
> interface ne déclenche aucune allocation : toutes les valeurs de taille zéro
> partagent un pointeur sentinelle interne au runtime. Le diagnostic d'évasion
> est donc nécessaire mais pas suffisant pour prédire une allocation — il
> indique une **fuite potentielle**, pas un coût garanti ; seul `-benchmem`
> tranche.

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

> 💡 Pourquoi **511** allocations ici (un nombre _exact_, `n-1` pour `n=512`
> fragments) contre seulement **19** pour le slice de la section suivante,
> alors que les deux structures grandissent dans une boucle ? Parce que `s +=
p` n'a **aucune marge de manœuvre** : chaque chaîne est immuable et de taille
> fixe, donc chaque tour alloue _exactement_ la nouvelle longueur totale, sans
> rien réserver en plus. `append`, lui, **sur-alloue volontairement** (capacité
> doublée, voir section 5) : il amortit le coût sur plusieurs insertions futures
> au prix d'un peu de mémoire inutilisée. `strings.Builder` applique cette même
> idée de sur-allocation à un tampon d'octets — c'est lui, pas l'immutabilité
> des chaînes, qui change de régime.

---

## 5. Préallocation d'un slice

`append` fait croître la capacité par paliers : chaque palier réalloue et
**recopie** tout. La progression exacte (`runtime.nextslicecap`) double la
capacité tant qu'elle reste **sous 256 éléments**, puis se rapproche d'un
facteur **1,25×** au-delà — une croissance plus douce, adoptée en **Go 1.18**
pour limiter le surcoût mémoire des très grands slices (doubler un slice de
plusieurs millions d'éléments gaspillerait trop). Si la taille finale est
connue, `make([]T, 0, n)` réserve la capacité finale **d'un coup** et court-
circuite toute cette progression (🔁 Ch. 30).

| Construction de 10 000 entiers |   ns/op |    B/op | allocs/op |
| ------------------------------ | ------: | ------: | --------: |
| `append` sur slice `nil`       | ~31 300 | 357 626 |        19 |
| `make([]int, 0, n)` + `append` |  ~6 230 |  81 920 |     **1** |

> 💡 **~5× plus rapide, une seule allocation** au lieu de 19. Chacune des 19
> allocations correspond à un repalier de la progression doublement/1,25×
> décrite ci-dessus, nécessaire pour atteindre une capacité ≥ 10 000 ; la
> préallocation les supprime **toutes** en réservant la capacité finale dès le
> `make`.

---

## 6. Préallocation d'une map

Même principe pour les maps : indiquer la taille attendue à `make(map[K]V, n)`
réduit les redimensionnements internes (rehash) — d'autant que Go 1.24 a adopté
les **Swiss Tables** (🔁 Ch. 32). Le principe : les entrées sont rangées par
**groupes de 8 emplacements** (`abi.MapGroupSlots`), chaque groupe disposant
d'un octet de contrôle par emplacement qui permet de sonder les 8 d'un coup
plutôt qu'un par un. Une table grandit dès que sa charge moyenne dépasse **7/8**
(87,5 %, `maxAvgGroupLoad`) — un seuil plus élevé, donc moins de
redimensionnements, que l'ancienne implémentation par buckets chaînés. Une
carte volumineuse se découpe en **plusieurs tables** indexées par un
**répertoire** ; chaque table stocke ses groupes dans un tableau **contigu**
(une seule allocation par table). Pré-dimensionner avec `n` réserve d'emblée
une table à la bonne capacité, mais la croissance reste **discrétisée** par ces
paliers internes — d'où les 34 allocations résiduelles, contre un unique `make`
pour le slice préalloué (un bloc contigu, sans cette indirection par tables).

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

Lecture, champ par champ (format documenté par `go doc runtime` → `GODEBUG`,
section `gctrace`) :

| Champ                       | Sens                                                                                                                                                                                                                                                           |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `gc 1`                      | numéro de cycle, incrémenté à chaque GC.                                                                                                                                                                                                                       |
| `@0.006s`                   | secondes écoulées depuis le démarrage du programme.                                                                                                                                                                                                            |
| `1%`                        | part **cumulée** du temps CPU passée en GC depuis le démarrage (pas seulement ce cycle).                                                                                                                                                                       |
| `0.056+0.49+0.033 ms clock` | durée des **trois phases** du cycle : STW de terminaison du balayage précédent, marquage **concurrent** (le programme continue de tourner), puis STW de terminaison du marquage. Seules les deux STW arrêtent tout le programme — volontairement très courtes. |
| `3->4->0 MB`                | taille du tas au **début** du cycle, à la **fin** du cycle, et taille du tas **vivant** restant après balayage.                                                                                                                                                |
| `4 MB goal`                 | taille cible du tas pour ce cycle, dérivée de `GOGC` (et de `GOMEMLIMIT` si plus contraignant).                                                                                                                                                                |
| `8 P`                       | nombre de processeurs logiques utilisés (`GOMAXPROCS`).                                                                                                                                                                                                        |

C'est qualitatif, mais c'est le premier réflexe pour savoir si le GC est un
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
