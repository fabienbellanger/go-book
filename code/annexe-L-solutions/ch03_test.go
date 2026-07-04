package main

import "testing"

func TestCh03ToInt8Unchecked(t *testing.T) {
	// 200 déborde int8 : la conversion tronque à 200 - 256 = -56.
	if got := ch03ToInt8Unchecked(200); got != -56 {
		t.Errorf("int8(200) = %d, veut -56", got)
	}
}

func TestCh03ToInt8Checked(t *testing.T) {
	if _, err := ch03ToInt8Checked(200); err == nil {
		t.Error("200 devrait être rejeté")
	}
	got, err := ch03ToInt8Checked(100)
	if err != nil || got != 100 {
		t.Errorf("100 : got %d, err %v", got, err)
	}
}
