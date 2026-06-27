package main

import (
	"slices"
	"testing"
)

func TestDedupAgrees(t *testing.T) {
	cases := [][]string{
		{},
		{"a"},
		{"a", "a", "a"},
		{"a", "b", "a", "c", "b"},
		makeItems(200),
	}
	for _, in := range cases {
		naive := DedupNaive(in)
		fast := Dedup(in)
		if !slices.Equal(naive, fast) {
			t.Errorf("désaccord pour %v : naive=%v fast=%v", in, naive, fast)
		}
	}
}

func TestDedupPreservesOrder(t *testing.T) {
	got := Dedup([]string{"c", "a", "c", "b", "a"})
	want := []string{"c", "a", "b"}
	if !slices.Equal(got, want) {
		t.Errorf("Dedup = %v ; attendu %v (ordre de 1re apparition)", got, want)
	}
}
