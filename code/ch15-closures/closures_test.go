package main

import (
	"slices"
	"testing"
	"time"
)

// counter : chaque instance a son propre état, indépendant des autres.
func TestCounterIsIndependent(t *testing.T) {
	a, b := counter(), counter()
	if got := []int{a(), a(), a()}; !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("a : got %v ; attendu [1 2 3]", got)
	}
	if got := b(); got != 1 { // b n'a pas bougé pendant qu'on appelait a
		t.Errorf("b() = %d ; attendu 1", got)
	}
}

// makeAdders : portée par itération (1.22) -> chaque closure capture une i distincte.
func TestPerIterationCapture(t *testing.T) {
	var got []int
	for _, add := range makeAdders() {
		got = append(got, add())
	}
	if want := []int{0, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("makeAdders = %v ; attendu %v (avant 1.22 : [3 3 3])", got, want)
	}
}

// memoize : fn n'est appelée qu'une fois par entrée distincte.
func TestMemoizeCachesResults(t *testing.T) {
	square, calls := memoize(func(x int) int { return x * x })
	_ = square(8)
	_ = square(8) // servi par le cache
	_ = square(9)
	if *calls != 2 { // 8 et 9 -> 2 calculs réels, malgré 3 appels
		t.Errorf("calculs réels = %d ; attendu 2", *calls)
	}
}

// chain : les middlewares s'appliquent dans l'ordre fourni (tagged puis upper).
func TestMiddlewareChain(t *testing.T) {
	h := chain(
		func(req string) string { return "hello " + req },
		tagged("api"),
		upper,
	)
	if got, want := h("go"), "api:HELLO GO"; got != want {
		t.Errorf("h(go) = %q ; attendu %q", got, want)
	}
}

// NewServer : les options écrasent les défauts, sans toucher au reste.
func TestFunctionalOptions(t *testing.T) {
	srv := NewServer("localhost", WithPort(9090))
	if srv.port != 9090 {
		t.Errorf("port = %d ; attendu 9090", srv.port)
	}
	if srv.timeout != 30*time.Second { // défaut conservé
		t.Errorf("timeout = %s ; attendu 30s", srv.timeout)
	}
}
