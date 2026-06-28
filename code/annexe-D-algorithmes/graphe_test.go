package main

import "testing"

func TestBFSDFS(t *testing.T) {
	g := NewGraph()
	g.AddEdge(1, 2)
	g.AddEdge(1, 3)
	g.AddEdge(2, 4)
	g.AddEdge(3, 4)

	bfs := g.BFS(1)
	if len(bfs) != 4 || bfs[0] != 1 {
		t.Errorf("BFS depuis 1 = %v, attendu 4 sommets commençant par 1", bfs)
	}
	dfs := g.DFS(1)
	if len(dfs) != 4 || dfs[0] != 1 {
		t.Errorf("DFS depuis 1 = %v, attendu 4 sommets commençant par 1", dfs)
	}
}

func TestTopoSort(t *testing.T) {
	g := NewGraph()
	g.AddEdge(1, 2)
	g.AddEdge(1, 3)
	g.AddEdge(2, 4)
	g.AddEdge(3, 4)
	order, ok := g.TopoSort()
	if !ok {
		t.Fatal("graphe acyclique : un ordre topologique devrait exister")
	}
	// Propriété : pour tout arc u -> v, u précède v dans l'ordre.
	pos := make(map[int]int, len(order))
	for i, v := range order {
		pos[v] = i
	}
	for u, neighbors := range g.adj {
		for _, v := range neighbors {
			if pos[u] > pos[v] {
				t.Errorf("ordre invalide : %d (pos %d) après %d (pos %d)", u, pos[u], v, pos[v])
			}
		}
	}
}

func TestTopoSortCycle(t *testing.T) {
	g := NewGraph()
	g.AddEdge(1, 2)
	g.AddEdge(2, 3)
	g.AddEdge(3, 1) // cycle
	if _, ok := g.TopoSort(); ok {
		t.Error("un graphe cyclique ne doit pas avoir d'ordre topologique")
	}
}

func TestDijkstra(t *testing.T) {
	g := NewWGraph()
	g.AddEdge(0, 1, 4)
	g.AddEdge(0, 2, 1)
	g.AddEdge(2, 1, 1)
	g.AddEdge(1, 3, 1)
	dist := g.Dijkstra(0)
	// Plus court chemin 0->2->1->3 = 3 (et non 0->1->3 = 5).
	want := map[int]int{0: 0, 2: 1, 1: 2, 3: 3}
	for node, d := range want {
		if dist[node] != d {
			t.Errorf("dist[%d] = %d, voulu %d", node, dist[node], d)
		}
	}
}
