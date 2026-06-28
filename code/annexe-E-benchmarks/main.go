// Commande de démonstration de l'Annexe E : micro-études chiffrées.
//
// L'essentiel vit dans les fichiers _test.go (les benchmarks). main() ne sert
// qu'à exécuter rapidement les fonctions et à montrer qu'elles coïncident.
//
//	go test -run='^$' -bench=. -benchmem ./annexe-E-benchmarks/
//	go build -gcflags="-m" ./annexe-E-benchmarks/   # voir l'escape analysis
package main

import "fmt"

func main() {
	fmt.Println("pile :", sumOnStack(3, 4), " tas :", newOnHeap(3, 4).x+newOnHeap(3, 4).y)

	parts := []string{"a", "b", "c", "d"}
	fmt.Println("concat :", concatPlus(parts), concatBuilder(parts), concatBuilderGrow(parts))

	xs := []int{1, 2, 3}
	fmt.Println("interface :", viaInterface(intDoubler{}, xs), " générique :", viaGeneric(intDoubler{}, xs))

	var mc mutexCounter
	var ac atomicCounter
	for range 100 {
		mc.Inc()
		ac.Inc()
	}
	fmt.Println("compteurs :", mc.Value(), ac.Value())

	fmt.Println("slices :", len(sliceNoPrealloc(5)), len(slicePrealloc(5)))
	fmt.Println("maps :", len(mapNoPrealloc(5)), len(mapPrealloc(5)))
}
