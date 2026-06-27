package main

import (
	"bytes"
	"testing"
)

func TestCollatzSteps(t *testing.T) {
	cases := map[int]int{1: 0, 2: 1, 6: 8, 27: 111}
	for n, want := range cases {
		if got := collatzSteps(n); got != want {
			t.Errorf("collatzSteps(%d) = %d ; attendu %d", n, got, want)
		}
	}
}

func TestWordFrequencies(t *testing.T) {
	counts := wordFrequencies("Go go GO profil", 3)
	if counts["go"] != 9 { // 3 occurrences x 3 répétitions
		t.Errorf("counts[go] = %d ; attendu 9", counts["go"])
	}
	if counts["profil"] != 3 {
		t.Errorf("counts[profil] = %d ; attendu 3", counts["profil"])
	}
}

func TestCountProfiles(t *testing.T) {
	// Profils prédéfinis par défaut : allocs, block, goroutine, heap, mutex, threadcreate.
	if n := CountProfiles(); n != 6 {
		t.Errorf("CountProfiles = %d ; attendu 6", n)
	}
}

// isPprof vérifie l'en-tête gzip (0x1f 0x8b) : un profil pprof est un protobuf
// compressé en gzip.
func isPprof(b []byte) bool {
	return len(b) > 2 && b[0] == 0x1f && b[1] == 0x8b
}

func TestCaptureCPUProfile(t *testing.T) {
	var buf bytes.Buffer
	err := CaptureCPUProfile(&buf, func() { _ = HotCompute(5000, 1) })
	if err != nil {
		t.Fatalf("CaptureCPUProfile: %v", err)
	}
	if !isPprof(buf.Bytes()) {
		t.Errorf("le profil CPU ne ressemble pas à du pprof (gzip) ; %d octets", buf.Len())
	}
}

func TestCaptureHeapProfile(t *testing.T) {
	_ = wordFrequencies("alloc des mots pour le tas", 100)
	var buf bytes.Buffer
	if err := CaptureHeapProfile(&buf); err != nil {
		t.Fatalf("CaptureHeapProfile: %v", err)
	}
	if !isPprof(buf.Bytes()) {
		t.Errorf("le profil tas ne ressemble pas à du pprof (gzip) ; %d octets", buf.Len())
	}
}
