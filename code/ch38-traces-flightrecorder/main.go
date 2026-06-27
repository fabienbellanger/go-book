package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i * i
	}

	// Mode "trace" : écrit trace.out pour `go tool trace`.
	if len(os.Args) > 1 && os.Args[1] == "trace" {
		f, _ := os.Create("trace.out")
		defer f.Close()
		_ = CaptureTrace(f, func() {
			for range 50 {
				_ = processBatch(context.Background(), items)
			}
		})
		fmt.Println("trace écrite : trace.out")
		fmt.Println("  go tool trace trace.out")
		return
	}

	sum := processBatch(context.Background(), items)
	fmt.Printf("somme des racines = %d\n", sum)

	// Flight Recorder : on simule une étape qui « décroche » au tour 7.
	var buf capture
	captured, _ := MonitorLatency(&buf, 20*time.Millisecond, 20, func(i int) {
		work := time.Millisecond
		if i == 7 {
			work = 40 * time.Millisecond // pic de latence rare
		}
		time.Sleep(work)
	})
	fmt.Printf("Flight Recorder : capture déclenchée = %v (%d octets figés)\n",
		captured, buf.n)
}

// capture est un io.Writer minimal qui compte les octets reçus.
type capture struct{ n int }

func (c *capture) Write(p []byte) (int, error) {
	c.n += len(p)
	return len(p), nil
}
