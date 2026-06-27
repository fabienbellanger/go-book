package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"time"
)

func makeShapes(n int) []Shape {
	shapes := make([]Shape, n)
	for i := range shapes {
		if i%100 == 0 {
			shapes[i] = Circle{R: float64(i)} // 1 % de Circle
		} else {
			shapes[i] = Square{Side: float64(i)} // 99 % de Square : Square domine
		}
	}
	return shapes
}

func main() {
	shapes := makeShapes(1000)

	// Mode "profile" : écrit default.pgo, exploitable ensuite par `go build -pgo=auto`.
	if len(os.Args) > 1 && os.Args[1] == "profile" {
		f, _ := os.Create("default.pgo")
		defer f.Close()
		_ = pprof.StartCPUProfile(f)
		var sink float64
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			for range 1000 {
				sink += TotalArea(shapes)
			}
		}
		pprof.StopCPUProfile()
		fmt.Printf("default.pgo écrit (sink=%.0f)\n", sink)
		fmt.Println("  go build -pgo=auto -gcflags=-d=pgodebug=2 .")
		return
	}

	fmt.Printf("AddTwice(21)      = %d\n", AddTwice(21))
	fmt.Printf("SumRange(1..5)    = %d\n", SumRange([]int{1, 2, 3, 4, 5}))
	fmt.Printf("SumGather         = %d\n", SumGather([]int{10, 20, 30}, []int{2, 0, 2}))
	fmt.Printf("TotalArea(1000)   = %.0f\n", TotalArea(shapes))
}
