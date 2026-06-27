package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func main() {
	ctx := context.Background()

	// 1. Pipeline en flux : source -> double -> +1. Les maillons tournent en
	// parallèle ; rien n'est matérialisé entre eux.
	nums := source(ctx, 1, 2, 3, 4)
	doubled := stage(ctx, nums, func(n int) int { return n * 2 })
	plusOne := stage(ctx, doubled, func(n int) int { return n + 1 })
	fmt.Println("pipeline   :", collect(plusOne)) // [3 5 7 9]

	// 2. Parallélisme borné : 3 workers pour 8 tâches, ordre préservé.
	squares := workerPool(3, []int{1, 2, 3, 4, 5, 6, 7, 8}, func(n int) int { return n * n })
	fmt.Println("workerPool :", squares) // [1 4 9 16 25 36 49 64]

	// 3. errgroup maison : une tâche échoue, les autres sont annulées via gctx.
	g, gctx := NewGroup(ctx)
	g.Go(func() error { return nil })
	g.Go(func() error { return errors.New("tâche 2 a échoué") })
	g.Go(func() error { <-gctx.Done(); return gctx.Err() }) // s'arrête à l'annulation
	fmt.Println("group.Wait :", g.Wait())

	// 4. Rate limiting (en temps réel ici : ~3 x 20 ms).
	start := time.Now()
	var seen []int
	rateLimited(ctx, []int{10, 20, 30}, 20*time.Millisecond, func(n int) { seen = append(seen, n) })
	fmt.Printf("rateLimited: %v en ~%v\n", seen, time.Since(start).Round(10*time.Millisecond))
}
