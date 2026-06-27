// Package main illustre l'observabilité du runtime : lire l'état des goroutines,
// du tas et du GC via runtime/metrics (l'API moderne), runtime.ReadMemStats et
// runtime/debug. Ces sondes alimentent le monitoring (expvar, Prometheus).
package main

import (
	"runtime"
	"runtime/debug"
	"runtime/metrics"
)

// Snapshot regroupe quelques indicateurs clés du runtime à un instant donné.
type Snapshot struct {
	Goroutines        int    // goroutines UTILISATEUR (runtime.NumGoroutine)
	GoroutinesAll     uint64 // TOUTES les goroutines, système comprises (/sched)
	GoroutinesCreated uint64 // total cumulé de goroutines créées (1.26)
	GOMAXPROCS        int    // nombre de P
	HeapAllocBytes    uint64 // octets vivants sur le tas
	HeapObjects       uint64 // nombre d'objets vivants sur le tas
	NumGC             uint64 // cycles de GC effectués
	GoVersion         string // version du toolchain (depuis le BuildInfo)
}

// readUint lit une métrique runtime/metrics de type uint64 (0 si absente).
func readUint(name string) uint64 {
	s := []metrics.Sample{{Name: name}}
	metrics.Read(s)
	if s[0].Value.Kind() == metrics.KindUint64 {
		return s[0].Value.Uint64()
	}
	return 0
}

// ReadSnapshot capture l'état courant. runtime/metrics est l'API recommandée :
// stable, étendue, et plus efficace que ReadMemStats (qui fait un STW partiel).
func ReadSnapshot() Snapshot {
	s := Snapshot{
		Goroutines:        runtime.NumGoroutine(),
		GoroutinesAll:     readUint("/sched/goroutines:goroutines"),
		GoroutinesCreated: readUint("/sched/goroutines-created:goroutines"),
		GOMAXPROCS:        runtime.GOMAXPROCS(0),
		HeapAllocBytes:    readUint("/memory/classes/heap/objects:bytes"),
		HeapObjects:       readUint("/gc/heap/objects:objects"),
		NumGC:             readUint("/gc/cycles/total:gc-cycles"),
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		s.GoVersion = bi.GoVersion
	}
	return s
}

// LegacyHeapAlloc lit le tas vivant via l'ANCIENNE API ReadMemStats, pour
// comparaison. Préférez runtime/metrics dans du code neuf.
func LegacyHeapAlloc() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // déclenche un court arrêt pour figer les stats
	return m.HeapAlloc
}
