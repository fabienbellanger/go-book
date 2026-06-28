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

Tri **en place**, pivot = dernier élément (partition de Lomuto). Moyenne
**O(n log n)**, pire cas **O(n²)** (entrée déjà triée, pivot dégénéré).

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

### Tri fusion (mergesort)

**O(n log n) garanti** et **stable**, au prix de **O(n)** mémoire. Renvoie une
nouvelle tranche, sans toucher l'entrée.

```go
func MergeSort[T cmp.Ordered](s []T) []T {
	if len(s) < 2 {
		return append([]T(nil), s...)
	}
	mid := len(s) / 2
	return merge(MergeSort(s[:mid]), MergeSort(s[mid:]))
}
```

> 💡 **En vrai, on n'écrit pas ça.** La bibliothèque standard fournit
> `slices.Sort` (un _pattern-defeating quicksort_ / introsort, O(n log n) au pire)
> et `slices.SortFunc` pour un comparateur sur mesure. Réimplémenter un tri n'a
> qu'un intérêt **pédagogique** — 🔁 Ch. 30. ⚡ `slices.Sort` est en place et sans
> allocation.

| Algorithme    | Temps (moy.) | Temps (pire) | Mémoire  | Stable |
| ------------- | :----------: | :----------: | :------: | :----: |
| Quicksort     |  O(n log n)  |    O(n²)     | O(log n) |  non   |
| Mergesort     |  O(n log n)  |  O(n log n)  |   O(n)   |  oui   |
| `slices.Sort` |  O(n log n)  |  O(n log n)  | O(log n) |  non   |

---

## 2. Recherche dichotomique

Sur une tranche **triée** : O(log n). Renvoie l'indice et un booléen de présence ;
sinon, le **point d'insertion** (même contrat que `slices.BinarySearch`).

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

> ⚠️ La tranche **doit** être triée. ⚡ `mid := int(uint(lo+hi) >> 1)` évite le
> débordement d'entier de `(lo+hi)/2` sur de très grandes tranches.
> 💡 Stdlib : `slices.BinarySearch` et `slices.BinarySearchFunc`.

---

## 3. Graphes

On représente un graphe **orienté** par **liste d'adjacence** (`map[int][]int`) :
compacte pour les graphes creux, et l'itération suit l'ordre d'insertion.

```
   1 --> 2 --\
   |          v
   \--> 3 --> 4
```

### Parcours en largeur (BFS) et en profondeur (DFS)

Tous deux en **O(V + E)**. BFS explore par couches (file), DFS plonge d'abord
(récursion).

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

Ordonne les sommets de sorte que chaque arc `u -> v` place `u` avant `v`. Le
booléen vaut `false` si un **cycle** rend l'ordre impossible.

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
	// file des sommets de degré entrant nul (triée -> résultat déterministe)
	// ... on émet, on décrémente les voisins, on enfile ceux qui tombent à 0.
}
```

### Plus court chemin (Dijkstra)

Sur un graphe **pondéré à poids positifs**, via une **file de priorité**
(`container/heap`). Complexité **O((V + E) log V)**.

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

Deux enveloppes minces autour d'une tranche.

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

### Union-Find (ensembles disjoints)

Avec **compression de chemin** + **union par rang** : `Find`/`Union` en temps
quasi constant amorti. Idéal pour les **composantes connexes**.

```go
func (uf *UnionFind) Find(x int) int {
	for uf.parent[x] != x {
		uf.parent[x] = uf.parent[uf.parent[x]] // compression « par moitié »
		x = uf.parent[x]
	}
	return x
}
```

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
