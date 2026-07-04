package main

import "testing"

func TestCh05Compose(t *testing.T) {
	inc := func(x int) int { return x + 1 }
	dbl := func(x int) int { return x * 2 }
	f := ch05Compose(inc, dbl) // x -> (x*2) + 1
	if got := f(3); got != 7 {
		t.Errorf("compose(inc, dbl)(3) = %d, veut 7", got)
	}
}

func TestCh05SumDivMod(t *testing.T) {
	// sum(divmod(17, 5)) : 17/5=3, 17%5=2, somme = 5.
	if got := ch05Sum(ch05DivMod(17, 5)); got != 5 {
		t.Errorf("sum(divmod(17,5)) = %d, veut 5", got)
	}
}

func TestCh05Options(t *testing.T) {
	s := ch05NewServer(ch05WithPort(9000), ch05WithMaxConns(50))
	if s.port != 9000 || s.maxConns != 50 {
		t.Errorf("options mal appliquées : %+v", s)
	}
	// Aucune option : les valeurs par défaut tiennent.
	if d := ch05NewServer(); d.port != 8080 || d.maxConns != 100 {
		t.Errorf("défauts : %+v", d)
	}
}
