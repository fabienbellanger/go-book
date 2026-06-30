# Annexe D — Algorithmes & structures de données en Go

> **Objectif** — Réviser les algorithmes et structures de données classiques sous
> leur forme **idiomatique Go** : génériques quand c'est utile, commentés, et
> **testés**. Pour chaque brique, on rappelle l'**équivalent de la bibliothèque
> standard** — car en pratique, on réécrit rarement un tri à la main.

> **Prérequis** — [Ch. 11 Généricité](../chapitres/11-genericite.md),
> [Ch. 30 Slices en profondeur](../chapitres/30-slices-profondeur.md). Le code
> compilable et testé vit dans [`code/annexe-D-algorithmes/`](../code/annexe-D-algorithmes/).

```bash
cd code && go test ./annexe-D-algorithmes/...
```

---

## 1. Tri

### Tri rapide (quicksort)

Tri **en place**, pivot = dernier élément (partition de Lomuto). La boucle de
partition fait un seul passage sur la tranche (**O(n)**) et sépare les
éléments plus petits que le pivot (à gauche) des plus grands (à droite) ; le
pivot se retrouve alors à sa position finale, et l'algorithme récurse sur les
deux moitiés. Quand les deux moitiés sont à peu près équilibrées, la récursion
compte **log n** niveaux, chacun coûtant O(n) au total : **O(n log n)** en
moyenne.

Le pire cas, **O(n²)**, survient quand le pivot est systématiquement
**l'extrême** de la tranche restante — typiquement une entrée **déjà triée**
avec ce choix « dernier élément » : le pivot est alors le maximum, la
partition place tous les autres éléments à gauche, et la récursion ne réduit
la taille du problème que d'**un seul élément** par niveau (n niveaux au lieu
de log n).

```go
func QuickSort[T cmp.Ordered](s []T) {
	if len(s) < 2 {
		return
	}
	pivot := s[len(s)-1]
	i := 0 // frontière des éléments < pivot
	for j := 0; j < len(s)-1; j++ {
		if s[j] < pivot {
			s[i], s[j] = s[j], s[i]
			i++
		}
	}
	s[i], s[len(s)-1] = s[len(s)-1], s[i]
	QuickSort(s[:i])
	QuickSort(s[i+1:])
}
```

```
s = [5 2 9 1 6 3], pivot = 3 (dernier élément), i = 0

j=0  5 < 3 ? non                     [5 2 9 1 6 3]
j=1  2 < 3 ? oui -> swap(i=0,j=1)    [2 5 9 1 6 3]   i=1
j=2  9 < 3 ? non                     [2 5 9 1 6 3]
j=3  1 < 3 ? oui -> swap(i=1,j=3)    [2 1 9 5 6 3]   i=2
j=4  6 < 3 ? non                     [2 1 9 5 6 3]

swap(i=2, dernier) -> pivot en place [2 1 3 5 6 9]
                              ^
                    < pivot (gauche) | > pivot (droite)
```

> 💡 Avec un tableau déjà trié `[1 2 3 5 6 9]`, le pivot (toujours le dernier,
> donc le maximum) ne laisserait **aucun** élément à droite :
> `QuickSort(s[:i])` récurserait sur n-1 éléments, encore avec un pivot
> maximal — d'où le O(n²). C'est pour cela qu'une implémentation de
> production choisit un pivot moins prévisible (médiane de trois, ou
> aléatoire) plutôt qu'une position fixe.

### Tri fusion (mergesort)

**O(n log n) garanti** et **stable**, au prix de **O(n)** mémoire. Renvoie une
nouvelle tranche, sans toucher l'entrée.

Contrairement au quicksort, la coupe `mid := len(s) / 2` est **fixe** : elle
ne dépend jamais du contenu de la tranche, donc la récursion compte toujours
exactement **log n** niveaux quelle que soit l'entrée — d'où la garantie (et
pas seulement la moyenne). Le O(n) mémoire vient de `merge` : fusionner deux
tranches déjà triées sans écraser des éléments pas encore lus exige un
**tampon séparé** (`out`) ; fusionner en place est possible mais nettement
plus complexe et rarement utilisé.

```go
func MergeSort[T cmp.Ordered](s []T) []T {
	if len(s) < 2 {
		return append([]T(nil), s...)
	}
	mid := len(s) / 2
	return merge(MergeSort(s[:mid]), MergeSort(s[mid:]))
}

// merge fusionne deux tranches déjà triées en une seule tranche triée.
func merge[T cmp.Ordered](a, b []T) []T {
	out := make([]T, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] { // <= et non < : à égalité, l'élément de gauche sort d'abord
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	out = append(out, a[i:]...)
	out = append(out, b[j:]...)
	return out
}
```

> 💡 La **stabilité** tient à ce `<=` : à valeurs égales, `merge` préfère
> toujours l'élément de la moitié **gauche**, qui était déjà avant l'autre
> dans la tranche d'origine — leur ordre relatif est donc préservé. Un `<`
> strict romprait cette garantie en laissant passer l'élément de droite en
> premier sur une égalité.

> 💡 **En vrai, on n'écrit pas ça.** La bibliothèque standard fournit
> `slices.Sort` (un _pattern-defeating quicksort_, ou **pdqsort** : un
> quicksort hybride qui combine plusieurs garde-fous pour éliminer le O(n²))
> et `slices.SortFunc` pour un comparateur sur mesure :
>
> - pivot choisi par **médiane de trois** (voire de neuf sur les grandes
>   tranches) plutôt qu'une position fixe, pour résister aux entrées qui le
>   dégénèrent ;
> - bascule vers un **tri par insertion** sous un seuil (tranches courtes, où
>   le surcoût de récursion dépasse le gain) ;
> - bascule vers un **heapsort** si la profondeur de récursion dépasse `2 log
n` (signe d'un pivot pathologique), ce qui **garantit** le O(n log n) au
>   pire — contrairement au `QuickSort` ci-dessus.
>
> Réimplémenter un tri n'a qu'un intérêt **pédagogique** — 🔁 Ch. 30. ⚡
> `slices.Sort` est en place et sans allocation.

| Algorithme    | Temps (moy.) | Temps (pire) | Mémoire  | Stable |
| ------------- | :----------: | :----------: | :------: | :----: |
| Quicksort     |  O(n log n)  |    O(n²)     | O(log n) |  non   |
| Mergesort     |  O(n log n)  |  O(n log n)  |   O(n)   |  oui   |
| `slices.Sort` |  O(n log n)  |  O(n log n)  | O(log n) |  non   |

---

## 2. Recherche dichotomique

Sur une tranche **triée** : O(log n). Renvoie l'indice et un booléen de présence ;
sinon, le **point d'insertion** (même contrat que `slices.BinarySearch`).

À chaque itération, l'intervalle `[lo, hi)` (semi-ouvert : `hi` n'est jamais
testé) est coupé en deux par `mid` : selon la comparaison, on élimine soit
`[lo, mid]`, soit `[mid, hi)` — la **moitié** de l'espace de recherche restant
disparaît à chaque tour. Pour une tranche de n éléments, il faut au plus
`log2(n)` divisions pour réduire l'intervalle à zéro, d'où le O(log n).

```go
func BinarySearch[T cmp.Ordered](s []T, target T) (int, bool) {
	lo, hi := 0, len(s)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1) // (lo+hi)/2 sans débordement
		switch {
		case s[mid] < target:
			lo = mid + 1
		case s[mid] > target:
			hi = mid
		default:
			return mid, true
		}
	}
	return lo, false
}
```

```
indices  0  1  2  3  4  5  6
valeurs  1  3  4  6  8  9 11        cible = 9
        lo                  hi (=7, exclu)

itération 1 : mid=(0+7)/2=3 -> s[3]=6 < 9 -> lo=4
                       lo                hi
                       4  5  6  7

itération 2 : mid=(4+7)/2=5 -> s[5]=9 = 9 -> trouvé, index 5
```

> ⚠️ La tranche **doit** être triée — sur une tranche désordonnée, l'algorithme
> ne lève aucune erreur, il renvoie juste un résultat **silencieusement faux**.
> ⚡ `mid := int(uint(lo+hi) >> 1)` évite le débordement d'entier de
> `(lo+hi)/2` sur de très grandes tranches.
> 💡 Stdlib : `slices.BinarySearch` et `slices.BinarySearchFunc`.

---

## 3. Graphes

On représente un graphe **orienté** par **liste d'adjacence**
(`map[int][]int`) : chaque sommet associe la liste de ses successeurs. C'est
le choix par défaut pour la quasi-totalité des graphes rencontrés en pratique
(réseau, dépendances, carte routière), qui sont **creux** — le nombre
d'arêtes E reste loin du maximum théorique V². Face à la matrice d'adjacence
(`[][]bool` de taille V×V), le compromis est net :

| Représentation                      | Mémoire | Tester l'arête (u,v) | Itérer les voisins de u |
| ----------------------------------- | :-----: | :------------------: | :---------------------: |
| Liste d'adjacence (`map[int][]int`) | O(V+E)  |      O(deg(u))       |   O(deg(u)) — optimal   |
| Matrice d'adjacence (`[][]bool`)    |  O(V²)  |         O(1)         |          O(V)           |

BFS, DFS, le tri topologique et Dijkstra parcourent **toujours** tous les
voisins d'un sommet ; aucun ne teste une arête isolée. La liste d'adjacence
est donc strictement meilleure ici — la matrice ne se justifie que sur un
graphe **dense** (E proche de V²), où le test d'arête en O(1) devient
déterminant.

```
   1 --> 2 --\
   |          v
   \--> 3 --> 4
```

> 💡 `g.adj[u]` est une **tranche** : la parcourir respecte l'ordre
> d'**insertion** des arêtes. Ne pas confondre avec l'itération sur les clés
> de la map `g.adj` elle-même (`for u := range g.adj`), volontairement
> **randomisée** par Go — 🔁 Ch. 7. `TopoSort`, plus bas, doit explicitement
> trier ses files d'attente pour rester déterministe malgré cela.

### Parcours en largeur (BFS) et en profondeur (DFS)

Tous deux en **O(V + E)** : chaque sommet est enfilé/visité **une seule fois**
grâce à `visited`, et chaque arête n'est examinée qu'une fois, au moment où
l'on parcourt `g.adj[u]` — d'où un coût total proportionnel au nombre de
sommets plus le nombre d'arêtes, jamais à V² même sur un graphe dense. BFS
explore par **couches** (file : on épuise tous les sommets à distance d avant
de passer à d+1), DFS plonge d'abord (récursion, qui utilise implicitement la
pile d'appels comme une pile explicite le ferait).

```go
func (g *Graph) BFS(start int) []int {
	visited := map[int]bool{start: true}
	queue := []int{start}
	var order []int
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		order = append(order, u)
		for _, v := range g.adj[u] {
			if !visited[v] {
				visited[v] = true
				queue = append(queue, v)
			}
		}
	}
	return order
}
```

### Tri topologique (Kahn)

Ordonne les sommets de sorte que chaque arc `u -> v` place `u` avant `v` —
utile pour ordonner des tâches sous dépendances (compilation, migrations de
base de données, résolution de paquets). Le booléen vaut `false` si un
**cycle** rend l'ordre impossible.

L'algorithme de Kahn maintient le **degré entrant** de chaque sommet (le
nombre d'arcs qui pointent vers lui) : un degré entrant nul signifie « plus
aucune dépendance non résolue », donc le sommet peut être émis. On part des
sommets de degré nul, on les émet, on décrémente le degré de leurs voisins, et
on enfile ceux qui tombent à zéro à leur tour.

```go
func (g *Graph) TopoSort() ([]int, bool) {
	indeg := make(map[int]int)
	for u := range g.adj {
		if _, ok := indeg[u]; !ok {
			indeg[u] = 0
		}
		for _, v := range g.adj[u] {
			indeg[v]++
		}
	}

	var queue []int
	for u, d := range indeg {
		if d == 0 {
			queue = append(queue, u)
		}
	}
	slices.Sort(queue) // résultat déterministe : l'ordre des maps est randomisé

	var order []int
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		order = append(order, u)
		var next []int
		for _, v := range g.adj[u] {
			indeg[v]--
			if indeg[v] == 0 {
				next = append(next, v)
			}
		}
		slices.Sort(next)
		queue = append(queue, next...)
	}
	if len(order) != len(indeg) {
		return nil, false // tous les sommets non émis : il reste un cycle
	}
	return order, true
}
```

Sur l'exemple `1 -> 2`, `1 -> 3`, `2 -> 4`, `3 -> 4` ci-dessus : seul `1` a un
degré entrant nul au départ (`indeg = {1:0, 2:1, 3:1, 4:2}`). On l'émet, ce
qui fait tomber `2` et `3` à 0 (émis ensuite, triés pour le déterminisme),
puis `4` à 0 une fois ses deux prédécesseurs émis : résultat `[1 2 3 4]`.

Si un cycle existe, les sommets qui en font partie ne voient **jamais** leur
degré entrant retomber à zéro (chacun attend qu'un autre membre du même cycle
soit émis en premier) : ils ne sont donc jamais enfilés, et `order` reste plus
court que l'ensemble des sommets — d'où le test final `len(order) !=
len(indeg)`.

### Plus court chemin (Dijkstra)

Sur un graphe **pondéré à poids positifs**, via une **file de priorité**
(`container/heap`). Complexité **O((V + E) log V)** : chaque sommet n'est
extrait du tas qu'une fois traité (V extractions `heap.Pop`, O(log V)
chacune), et chaque arête peut déclencher un ajout au tas quand elle révèle
une distance plus courte (au plus E insertions `heap.Push`, O(log V)
chacune) — d'où (V + E) opérations à O(log V). Une variante sans tas, qui
cherche le sommet non traité de distance minimale par balayage linéaire à
chaque tour, est en O(V²) : plus simple, et même préférable sur un graphe
**dense** où E approche V² — le facteur log V du tas ne compense alors plus
son surcoût de gestion.

```go
func (g *WGraph) Dijkstra(src int) map[int]int {
	dist := map[int]int{src: 0}
	pqueue := &minHeap{{node: src, dist: 0}}
	for pqueue.Len() > 0 {
		cur := heap.Pop(pqueue).(pqItem)
		if cur.dist > dist[cur.node] {
			continue // entrée périmée
		}
		for _, e := range g.adj[cur.node] {
			nd := cur.dist + e.weight
			if d, ok := dist[e.to]; !ok || nd < d { // jamais vu, ou plus court
				dist[e.to] = nd
				heap.Push(pqueue, pqItem{node: e.to, dist: nd})
			}
		}
	}
	return dist
}
```

> ⚠️ Dijkstra suppose des poids **non négatifs**. Avec des poids négatifs, il faut
> Bellman-Ford. 💡 Le « truc de l'entrée périmée » (`cur.dist > dist[cur.node]`)
> évite de retirer/mettre à jour des entrées du tas : on les ignore à la sortie.

---

## 4. Structures de données génériques

### Pile (LIFO) et file (FIFO)

Deux enveloppes minces autour d'une tranche — un choix **délibéré**. Dans un
langage où la liste chaînée est l'outil par défaut, pile et file
s'implémentent souvent nœud par nœud (un pointeur par élément). En Go, on leur
préfère une tranche sous-jacente pour deux raisons :

- un **seul bloc mémoire contigu**, donc une bonne **localité de cache** :
  parcourir les derniers éléments empilés revient à lire une zone mémoire
  continue, alors qu'une liste chaînée disperse chaque nœud où l'allocateur
  l'a placé, avec un déréférencement de pointeur — et un cache miss potentiel
  — à chaque saut ;
- une seule structure qui grossit par `append` (réallocations amorties en
  O(1) en moyenne) plutôt qu'**une allocation par élément** — 🔁 Ch. 30.

```go
type Stack[T any] struct{ items []T }

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items[len(s.items)-1] = zero // libère la référence : pas de fuite mémoire
	s.items = s.items[:len(s.items)-1]
	return v, true
}
```

> ⚠️ En dépilant, on **remet la valeur zéro** dans la case libérée. Sans cela, la
> tranche garderait une référence vers l'ancien élément (pointeur, slice, map),
> empêchant le GC de le récupérer — 🔁 Ch. 27.

`Queue[T]` suit le même principe en sens inverse : `Enqueue` ajoute en queue
(`append`), `Dequeue` retire en tête via `q.items = q.items[1:]` (après avoir,
là aussi, remis la case à zéro).

> ⚠️ Cette technique de file a une limite : `s[1:]` avance le **début** de la
> tranche mais ne réduit jamais le tableau sous-jacent — toute la mémoire
> allouée au pic d'occupation reste réservée tant qu'aucun `append` ne dépasse
> la capacité restante et ne force une réallocation. Pour une file de
> **longue durée** dont la taille oscille beaucoup, mieux vaut compacter
> périodiquement (`copy` vers une tranche neuve quand `len` est très inférieur
> à `cap`) ou utiliser une structure en anneau à taille fixe.

### Union-Find (ensembles disjoints)

Avec **compression de chemin** + **union par rang** : `Find`/`Union` en temps
quasi constant amorti. Idéal pour les **composantes connexes**.

Les deux optimisations se complètent :

- **union par rang** (dans `Union`) attache toujours le plus petit arbre sous
  la racine du plus grand — sans cette règle, une séquence d'`Union`
  malheureuse peut construire une chaîne dégénérée de profondeur n, et `Find`
  redevient O(n) ;
- **compression de chemin** (dans `Find`, ci-dessous) fait sauter chaque nœud
  traversé vers son **grand-parent** plutôt que son parent immédiat
  (compression « par moitié », moins coûteuse en écritures qu'une compression
  complète à deux passages, pour un effet d'aplatissement quasiment
  identique sur la durée).

```go
func (uf *UnionFind) Find(x int) int {
	for uf.parent[x] != x {
		uf.parent[x] = uf.parent[uf.parent[x]] // compression « par moitié »
		x = uf.parent[x]
	}
	return x
}
```

```
avant Find(d) :  a <- b <- c <- d   (parent[d]=c, parent[c]=b, parent[b]=a=racine)

pendant Find(d) :  x=d, parent[d] <- parent[parent[d]] = b   (saute par-dessus c)
                    x=b, parent[b] reste a (déjà la racine)

après Find(d) :  a <- b <- c
                       ^---- d      (parent[d] passe de c à b)
```

Combinées, ces deux optimisations bornent le coût amorti d'une séquence de m
opérations sur n éléments par **O(m · α(n))**, où α est l'**inverse de la
fonction d'Ackermann** — une fonction qui croît si lentement qu'elle reste
**inférieure à 5** pour toute valeur de n représentable en pratique (bien
au-delà du nombre d'atomes de l'univers observable). D'où l'approximation
usuelle « temps quasi constant ».

> 🔁 **Le Projet 4 (`gends`)** fournit des structures génériques prêtes à l'emploi
> et bien testées : `set` (ensemble), `pqueue` (file de priorité), `lru` (cache).
> Côté stdlib : `container/list` (liste doublement chaînée) et `container/heap`
> (interface de tas, base d'une file de priorité).

---

## 🧪 À tester soi-même

```bash
cd code && go test ./annexe-D-algorithmes/...
```

- Ajouter `Bellman-Ford` (poids négatifs) et comparer à Dijkstra.
- Rendre `Graph` **générique** sur le type de sommet (`Graph[T comparable]`).
- Mesurer `QuickSort` contre `slices.Sort` avec un benchmark (🔁 Annexe E).

---

## 📌 À retenir

- **Connaître** ces algorithmes, mais **utiliser la stdlib** : `slices.Sort`,
  `slices.BinarySearch`, `container/heap`, `container/list`.
- Le **choix de la structure** prime sur la micro-optimisation : liste
  d'adjacence pour un graphe creux, file de priorité pour Dijkstra.
- En Go, **remettre la valeur zéro** dans les cases libérées d'une pile/file
  évite de retenir des références et de gêner le GC.
- Les **génériques** (`[T cmp.Ordered]`, `[T any]`) rendent ces briques
  réutilisables sans `interface{}` ni assertions de type.
