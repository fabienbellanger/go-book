package lru_test

import (
	"slices"
	"testing"

	"example.com/gends/lru"
)

func TestPutGet(t *testing.T) {
	c := lru.New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)

	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Errorf("Get a = (%d, %v), voulu (1, true)", v, ok)
	}
	if _, ok := c.Get("absent"); ok {
		t.Error("Get sur clé absente doit renvoyer ok=false")
	}
}

func TestEviction(t *testing.T) {
	c := lru.New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Get("a") // a redevient la plus récente, b devient la plus ancienne
	if ev := c.Put("c", 3); !ev {
		t.Error("Put au-delà de la capacité doit signaler une éviction")
	}
	if c.Contains("b") {
		t.Error("b (la plus ancienne) aurait dû être évincée")
	}
	if !c.Contains("a") || !c.Contains("c") {
		t.Error("a et c doivent rester en cache")
	}
	if c.Len() != 2 {
		t.Errorf("Len = %d, voulu 2", c.Len())
	}
}

func TestUpdateDoesNotEvict(t *testing.T) {
	c := lru.New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	if ev := c.Put("a", 11); ev {
		t.Error("mettre à jour une clé existante ne doit pas évincer")
	}
	if v, _ := c.Get("a"); v != 11 {
		t.Errorf("valeur après mise à jour = %d, voulu 11", v)
	}
}

func TestKeysOrder(t *testing.T) {
	c := lru.New[int, int](3)
	c.Put(1, 0)
	c.Put(2, 0)
	c.Put(3, 0)
	c.Get(1) // 1 redevient le plus récent
	if got := c.Keys(); !slices.Equal(got, []int{1, 3, 2}) {
		t.Errorf("Keys = %v, voulu [1 3 2] (du plus récent au plus ancien)", got)
	}
}

func TestRemove(t *testing.T) {
	c := lru.New[string, int](2)
	c.Put("a", 1)
	if !c.Remove("a") || c.Remove("a") {
		t.Error("Remove doit renvoyer true une fois, puis false")
	}
	if c.Len() != 0 {
		t.Errorf("Len = %d après suppression, voulu 0", c.Len())
	}
}

func TestNewPanicsOnBadCapacity(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New(0) aurait dû paniquer")
		}
	}()
	lru.New[int, int](0)
}

func BenchmarkPutGet(b *testing.B) {
	c := lru.New[int, int](1024)
	for i := 0; b.Loop(); i++ {
		c.Put(i&0x7ff, i) // ~moitié de hits, moitié d'évictions
		c.Get(i & 0x7ff)
	}
}
