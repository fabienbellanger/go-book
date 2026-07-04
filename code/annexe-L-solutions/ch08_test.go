package main

import (
	"testing"
	"unsafe"
)

func TestCh08PointerReceiver(t *testing.T) {
	a := &ch08Account{}
	a.Deposit(100)
	a.Deposit(50)
	if a.Balance() != 150 {
		t.Errorf("solde = %d, veut 150 (récepteur pointeur)", a.Balance())
	}
}

func TestCh08FieldOrdering(t *testing.T) {
	padded := unsafe.Sizeof(ch08Padded{})
	packed := unsafe.Sizeof(ch08Packed{})
	if packed >= padded {
		t.Errorf("le réordonnancement devrait réduire la taille : packed=%d padded=%d", packed, padded)
	}
	t.Logf("padded=%d octets, packed=%d octets", padded, packed)
}
