package main

import "testing"

func TestAddTwice(t *testing.T) {
	if got := AddTwice(21); got != 42 {
		t.Errorf("AddTwice(21) = %d ; attendu 42", got)
	}
}

func TestSumVariants(t *testing.T) {
	xs := []int{1, 2, 3, 4, 5}
	if got := SumRange(xs); got != 15 {
		t.Errorf("SumRange = %d ; attendu 15", got)
	}
	// SumGather(xs, idx) = xs[2]+xs[0]+xs[4] = 3+1+5 = 9.
	if got := SumGather(xs, []int{2, 0, 4}); got != 9 {
		t.Errorf("SumGather = %d ; attendu 9", got)
	}
	if got := SumHinted(xs); got != 10 { // xs[0..3] = 1+2+3+4
		t.Errorf("SumHinted = %d ; attendu 10", got)
	}
	if got := SumHinted([]int{1, 2}); got != 0 { // trop court
		t.Errorf("SumHinted(court) = %d ; attendu 0", got)
	}
}

func TestTotalArea(t *testing.T) {
	shapes := []Shape{Square{Side: 2}, Circle{R: 1}, Square{Side: 3}}
	// 4 + 3.14159 + 9 = 16.14159
	if got := TotalArea(shapes); got < 16.14 || got > 16.15 {
		t.Errorf("TotalArea = %v ; attendu ~16.14", got)
	}
}
