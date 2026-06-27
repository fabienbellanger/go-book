package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 1. Fin normale : on alimente le canal puis on le ferme.
	in := make(chan int)
	go func() {
		defer close(in)
		for _, n := range []int{1, 2, 3} {
			in <- n
		}
	}()
	sum, err := sumUntilCancel(context.Background(), in)
	fmt.Printf("normal  : somme=%d err=%v\n", sum, err) // 6, nil

	// 2. Annulation explicite avec une CAUSE métier.
	ctx2, cancel := context.WithCancelCause(context.Background())
	cancel(ErrTooSlow) // on annule tout de suite avec une cause
	sum, err = sumUntilCancel(ctx2, make(chan int))
	fmt.Printf("annulé  : somme=%d err=%v | Err()=%v\n", sum, err, ctx2.Err())

	// 3. Délai dépassé : WithTimeout déclenche DeadlineExceeded.
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel3() // TOUJOURS appeler cancel pour libérer les ressources
	_, err = sumUntilCancel(ctx3, make(chan int))
	fmt.Printf("timeout : err=%v\n", err)

	// 4. Valeur de contexte : un identifiant qui traverse les appels.
	rctx := WithRequestID(context.Background(), "req-42")
	if id, ok := RequestID(rctx); ok {
		fmt.Println("valeur  : request id =", id)
	}
}
