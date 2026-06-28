package main

// produce illustre « ne communiquez pas en partageant la mémoire ; partagez la
// mémoire en communiquant ». Le producteur génère des valeurs et en CÈDE la
// propriété par le canal : une fois envoyée, une valeur n'est plus touchée par le
// producteur. Le consommateur en devient seul propriétaire. Aucune mémoire n'est
// partagée, donc aucune course n'est possible — c'est la sûreté par conception.
func produce(n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // le PRODUCTEUR ferme : il est le seul à envoyer (🔁 Ch. 20)
		for i := range n {
			out <- i
		}
	}()
	return out
}

// consume additionne les valeurs reçues. range s'arrête à la fermeture du canal :
// pas de signal d'arrêt à inventer, pas de goroutine qui fuit.
func consume(in <-chan int) int {
	total := 0
	for v := range in {
		total += v
	}
	return total
}
