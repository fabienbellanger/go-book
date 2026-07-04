package main

import "testing"

func TestCh04Classify(t *testing.T) {
	cases := map[int]string{
		100: "excellent", 95: "excellent", 90: "excellent",
		85: "bien", 70: "bien",
		65: "passable", 50: "passable",
		40: "insuffisant", 0: "insuffisant",
	}
	for score, want := range cases {
		if got := ch04Classify(score); got != want {
			t.Errorf("classify(%d) = %q, veut %q", score, got, want)
		}
	}
}

func TestCh04FirstPair(t *testing.T) {
	xs := []int{1, 2, 3}
	ys := []int{4, 5, 6}
	a, b, ok := ch04FirstPair(xs, ys, 7)
	if !ok || a+b != 7 {
		t.Errorf("veut un couple de somme 7, got %d+%d ok=%v", a, b, ok)
	}
	if _, _, ok := ch04FirstPair(xs, ys, 100); ok {
		t.Error("aucune paire ne fait 100")
	}
}
