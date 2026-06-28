// Commande de démonstration de l'annexe D : algorithmes et structures de données
// classiques, implémentés idiomatiquement en Go générique. Lancer : go run .
package main

import "fmt"

func main() {
	nums := []int{5, 2, 9, 1, 5, 6}
	QuickSort(nums)
	fmt.Println("trié :", nums)

	if idx, found := BinarySearch(nums, 6); found {
		fmt.Printf("6 trouvé à l'index %d\n", idx)
	}

	g := NewGraph()
	g.AddEdge(1, 2)
	g.AddEdge(1, 3)
	g.AddEdge(2, 4)
	g.AddEdge(3, 4)
	fmt.Println("BFS depuis 1 :", g.BFS(1))
	if order, ok := g.TopoSort(); ok {
		fmt.Println("ordre topologique :", order)
	}

	wg := NewWGraph()
	wg.AddEdge(0, 1, 4)
	wg.AddEdge(0, 2, 1)
	wg.AddEdge(2, 1, 1)
	wg.AddEdge(1, 3, 1)
	fmt.Println("distance 0 -> 3 :", wg.Dijkstra(0)[3])

	uf := NewUnionFind(5)
	uf.Union(0, 1)
	uf.Union(1, 2)
	fmt.Println("0 et 2 connectés ?", uf.Connected(0, 2))
}
