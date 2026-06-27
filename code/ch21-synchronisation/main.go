package main

import "fmt"

func main() {
	// 1. Compteurs concurrents : mutex vs atomic, même résultat (1000).
	var safe SafeCounter
	runConcurrently(1000, safe.Inc)
	fmt.Println("SafeCounter (mutex)   :", safe.Value())

	var fast AtomicCounter
	runConcurrently(1000, fast.Inc)
	fmt.Println("AtomicCounter (atomic):", fast.Value())

	// 2. Initialisation paresseuse : expensiveInit ne tourne qu'UNE fois,
	// malgré 100 appelants concurrents.
	runConcurrently(100, func() { _ = config() })
	fmt.Printf("config() = %v, chargé %d fois\n", config(), loadCount.Load())

	// 3. Cache lecture-intensif (RWMutex).
	reg := NewRegistry()
	reg.Set("go", 1)
	if v, ok := reg.Get("go"); ok {
		fmt.Println("Registry[go]          :", v)
	}

	// 4. Échange d'état sans verrou (atomic.Pointer) : publication atomique.
	cfg := NewConfig(&Settings{Level: 1})
	cfg.Store(&Settings{Level: 2, Verbose: true})
	fmt.Printf("Settings              : %+v\n", *cfg.Load())

	// 5. Buffer recyclé (sync.Pool).
	fmt.Println("joinInts              :", joinInts([]int{1, 2, 3}, "-"))
}
