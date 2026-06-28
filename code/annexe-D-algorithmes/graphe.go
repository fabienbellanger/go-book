package main

import (
	"container/heap"
	"slices"
)

// Graph est un graphe ORIENTÉ représenté par liste d'adjacence. Les sommets sont
// des entiers ; AddEdge(u, v) ajoute l'arc u -> v.
type Graph struct {
	adj map[int][]int
}

// NewGraph crée un graphe vide.
func NewGraph() *Graph {
	return &Graph{adj: make(map[int][]int)}
}

// AddEdge ajoute l'arc u -> v (et enregistre v comme sommet s'il est nouveau).
func (g *Graph) AddEdge(u, v int) {
	g.adj[u] = append(g.adj[u], v)
	if _, ok := g.adj[v]; !ok {
		g.adj[v] = nil
	}
}

// BFS parcourt le graphe EN LARGEUR depuis start et renvoie les sommets dans
// l'ordre de visite. Complexité O(V + E).
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

// DFS parcourt EN PROFONDEUR depuis start (récursif). Complexité O(V + E).
func (g *Graph) DFS(start int) []int {
	visited := make(map[int]bool)
	var order []int
	var visit func(u int)
	visit = func(u int) {
		visited[u] = true
		order = append(order, u)
		for _, v := range g.adj[u] {
			if !visited[v] {
				visit(v)
			}
		}
	}
	visit(start)
	return order
}

// TopoSort renvoie un ordre topologique des sommets (algorithme de Kahn). Le
// booléen vaut false si le graphe contient un CYCLE (aucun ordre possible).
// On trie les candidats pour un résultat déterministe. Complexité O(V + E).
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
	slices.Sort(queue)

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

// WGraph est un graphe orienté PONDÉRÉ (poids >= 0), pour Dijkstra.
type WGraph struct {
	adj map[int][]edge
}

type edge struct {
	to     int
	weight int
}

// NewWGraph crée un graphe pondéré vide.
func NewWGraph() *WGraph {
	return &WGraph{adj: make(map[int][]edge)}
}

// AddEdge ajoute l'arc pondéré u -> v de poids w.
func (g *WGraph) AddEdge(u, v, w int) {
	g.adj[u] = append(g.adj[u], edge{to: v, weight: w})
	if _, ok := g.adj[v]; !ok {
		g.adj[v] = nil
	}
}

// Dijkstra renvoie la distance minimale de src à chaque sommet atteignable.
// Poids non négatifs requis. Complexité O((V + E) log V) avec file de priorité.
func (g *WGraph) Dijkstra(src int) map[int]int {
	dist := map[int]int{src: 0}
	pqueue := &minHeap{{node: src, dist: 0}}
	for pqueue.Len() > 0 {
		cur := heap.Pop(pqueue).(pqItem)
		if cur.dist > dist[cur.node] {
			continue // entrée périmée : une distance plus courte a déjà été traitée
		}
		for _, e := range g.adj[cur.node] {
			nd := cur.dist + e.weight
			if d, ok := dist[e.to]; !ok || nd < d {
				dist[e.to] = nd
				heap.Push(pqueue, pqItem{node: e.to, dist: nd})
			}
		}
	}
	return dist
}

// pqItem et minHeap implémentent container/heap (file de priorité min par dist).
// 🔁 Le Projet 4 (gends/pqueue) fournit une file de priorité générique réutilisable.
type pqItem struct {
	node int
	dist int
}

type minHeap []pqItem

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].dist < h[j].dist }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(pqItem)) }
func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}
