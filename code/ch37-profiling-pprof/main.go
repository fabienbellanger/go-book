package main

import (
	"fmt"
	"os"
	"strings"
)

// Texte de départ, multiplié pour donner du grain au profileur.
const sample = `Go rend le profiling simple et intégré au langage.
Profiler tôt, profiler souvent, mais toujours mesurer avant d'optimiser.
Le profil CPU montre où part le temps, le profil tas où part la mémoire.`

func main() {
	text := strings.Repeat(sample+" ", 50)

	// Mode "profile" : écrit cpu.prof et mem.prof pour `go tool pprof`.
	if len(os.Args) > 1 && os.Args[1] == "profile" {
		cpu, _ := os.Create("cpu.prof")
		defer cpu.Close()
		_ = CaptureCPUProfile(cpu, func() {
			_ = HotCompute(200000, 60) // CPU-bound, sans allocation
		})

		mem, _ := os.Create("mem.prof")
		defer mem.Close()
		_ = wordFrequencies(text, 20000) // peuple le tas avant l'instantané
		_ = CaptureHeapProfile(mem)

		fmt.Println("profils écrits : cpu.prof, mem.prof")
		fmt.Println("  go tool pprof -top cpu.prof")
		return
	}

	counts := wordFrequencies(text, 1)
	fmt.Printf("%d mots distincts ; \"profil\" apparaît %d fois\n",
		len(counts), counts["profil"])
	fmt.Printf("%d profils prédéfinis disponibles (allocs, block, goroutine, heap, mutex, threadcreate)\n",
		CountProfiles())
}

// --- Exposer les profils d'un SERVICE (net/http/pprof) ---
//
// Un simple import à effet de bord greffe /debug/pprof/ sur le mux par défaut :
//
//	import _ "net/http/pprof"
//
//	func init() {
//		// Port INTERNE uniquement (jamais public : fuite d'infos + DoS).
//		go http.ListenAndServe("localhost:6060", nil)
//	}
//
// Puis, à chaud, sans redémarrer le service :
//
//	go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30  # CPU 30 s
//	go tool pprof http://localhost:6060/debug/pprof/heap                # tas
