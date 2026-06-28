package main

import "testing"

func TestBinarySearch(t *testing.T) {
	s := []int{1, 3, 5, 7, 9}
	tests := []struct {
		target    int
		wantIdx   int
		wantFound bool
	}{
		{5, 2, true},
		{1, 0, true},
		{9, 4, true},
		{0, 0, false},  // avant le premier
		{4, 2, false},  // entre 3 et 5
		{10, 5, false}, // après le dernier
	}
	for _, tc := range tests {
		idx, found := BinarySearch(s, tc.target)
		if idx != tc.wantIdx || found != tc.wantFound {
			t.Errorf("BinarySearch(%d) = (%d, %v), voulu (%d, %v)", tc.target, idx, found, tc.wantIdx, tc.wantFound)
		}
	}
}
