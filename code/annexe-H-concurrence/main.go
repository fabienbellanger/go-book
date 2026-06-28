package main

import "fmt"

// main fait tourner brièvement les trois patterns sûrs (à lancer avec « go run . »
// ou, mieux, « go test -race ./... » pour la validation).
func main() {
	var c Counter
	var wg = make(chan struct{})
	for range 100 {
		go func() { c.Inc(); wg <- struct{}{} }()
	}
	for range 100 {
		<-wg
	}
	fmt.Println("compteur :", c.Value()) // 100

	a, b := NewAccount(1, 100), NewAccount(2, 100)
	Transfer(a, b, 30)
	fmt.Println("soldes :", a.Balance(), b.Balance()) // 70 130

	fmt.Println("somme 0..99 :", consume(produce(100))) // 4950
}
