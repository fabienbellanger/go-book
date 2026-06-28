package main

import (
	"slices"
	"testing"
)

func TestQuickSort(t *testing.T) {
	cases := [][]int{
		{},
		{1},
		{2, 1},
		{5, 2, 9, 1, 5, 6},
		{3, 3, 3},
		{9, 8, 7, 6, 5, 4, 3, 2, 1},
	}
	for _, in := range cases {
		got := slices.Clone(in)
		QuickSort(got)
		want := slices.Clone(in)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("QuickSort(%v) = %v, voulu %v", in, got, want)
		}
	}
}

func TestMergeSort(t *testing.T) {
	in := []string{"banane", "pomme", "cerise", "abricot"}
	got := MergeSort(in)
	want := slices.Clone(in)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("MergeSort = %v, voulu %v", got, want)
	}
	if in[0] != "banane" {
		t.Error("MergeSort ne doit pas modifier la tranche d'entrée")
	}
}
