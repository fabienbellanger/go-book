package main

import "testing"

type ch11Celsius float64 // type défini sur float64 : accepté grâce au ~ de Number

func TestCh11Sum(t *testing.T) {
	if got := ch11Sum([]int{1, 2, 3}); got != 6 {
		t.Errorf("sum int = %d", got)
	}
	// Un type défini est accepté par la contrainte ~float64.
	if got := ch11Sum([]ch11Celsius{1.5, 2.5}); got != 4 {
		t.Errorf("sum Celsius = %v", got)
	}
}

func TestCh11Index(t *testing.T) {
	if got := ch11Index([]string{"a", "b", "c"}, "b"); got != 1 {
		t.Errorf("index = %d, veut 1", got)
	}
	if got := ch11Index([]int{1, 2}, 9); got != -1 {
		t.Errorf("absent -> %d, veut -1", got)
	}
}

func TestCh11Zero(t *testing.T) {
	// T doit être explicite : ch11Zero() seul ne compilerait pas.
	if ch11Zero[int]() != 0 {
		t.Error("zéro int")
	}
	if ch11Zero[string]() != "" {
		t.Error("zéro string")
	}
}
