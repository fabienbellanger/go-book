package main

import "fmt"

func main() {
	// Closure à état : c1 et c2 ont chacune SON compteur indépendant.
	c1, c2 := counter(), counter()
	fmt.Println("c1:", c1(), c1(), c1(), "| c2:", c2()) // c1: 1 2 3 | c2: 1

	// Portée par itération (1.22) : chaque closure a capturé une i distincte.
	results := make([]int, 0, 3)
	for _, add := range makeAdders() {
		results = append(results, add())
	}
	fmt.Println("makeAdders ->", results) // [0 1 2]

	// Décorateur : double, mais tracé.
	double := logged("double", func(x int) int { return x * 2 })
	double(21)

	// Mémoïsation : le second appel ne recalcule pas.
	square, calls := memoize(func(x int) int { return x * x })
	fmt.Println("square(8):", square(8), "| square(8):", square(8), "| calculs réels:", *calls)

	// Middleware : tagged("api") puis upper, autour d'un handler de base.
	h := chain(
		func(req string) string { return "hello " + req },
		tagged("api"),
		upper,
	)
	fmt.Println("handler ->", h("go")) // api:HELLO GO

	// Option pattern : on ne règle que ce qu'on veut, le reste prend ses défauts.
	srv := NewServer("localhost", WithPort(9090))
	fmt.Printf("server -> %s:%d timeout=%s\n", srv.host, srv.port, srv.timeout)
}
