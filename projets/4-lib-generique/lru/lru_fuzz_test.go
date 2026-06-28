package lru_test

import (
	"testing"

	"example.com/gends/lru"
)

// FuzzCache pilote le cache avec une suite d'opérations dérivée d'octets
// aléatoires, et vérifie ses invariants face à un modèle de référence naïf.
//
// Le fuzzing trouve des suites d'opérations auxquelles on n'aurait pas pensé :
// ici, on confronte le cache à une réimplémentation triviale (slice + map) dont
// l'éviction LRU est évidente, et on exige qu'ils restent d'accord.
//
// Lancer : go test -run=Fuzz -fuzz=FuzzCache ./lru
func FuzzCache(f *testing.F) {
	// Quelques graines : (clé, valeur, get?) encodées dans des octets.
	f.Add([]byte{1, 10, 2, 20, 1, 0, 3, 30})
	f.Add([]byte{})
	f.Add([]byte{5, 5, 5, 5, 5, 5})

	const capacity = 4

	f.Fuzz(func(t *testing.T, ops []byte) {
		c := lru.New[byte, byte](capacity)
		model := newRefLRU(capacity)

		// On consomme les octets par triplets (clé, valeur, action).
		for i := 0; i+2 < len(ops); i += 3 {
			key := ops[i] & 0x07 // restreint l'espace des clés (0..7) pour des collisions
			val := ops[i+1]
			isGet := ops[i+2]&1 == 1

			if isGet {
				gotV, gotOK := c.Get(key)
				wantV, wantOK := model.get(key)
				if gotOK != wantOK || (gotOK && gotV != wantV) {
					t.Fatalf("Get(%d) = (%d,%v), modèle (%d,%v)", key, gotV, gotOK, wantV, wantOK)
				}
			} else {
				c.Put(key, val)
				model.put(key, val)
			}

			// Invariants structurels, vrais après chaque opération.
			if c.Len() > capacity {
				t.Fatalf("Len=%d dépasse la capacité %d", c.Len(), capacity)
			}
			if c.Len() != model.len() {
				t.Fatalf("Len=%d, modèle=%d", c.Len(), model.len())
			}
		}
	})
}

// refLRU est un cache LRU de référence, volontairement naïf (O(n)), servant
// d'oracle au fuzzing. order va du plus ancien (index 0) au plus récent.
type refLRU struct {
	capacity int
	order    []byte
	values   map[byte]byte
}

func newRefLRU(capacity int) *refLRU {
	return &refLRU{capacity: capacity, values: make(map[byte]byte)}
}

func (r *refLRU) touch(key byte) {
	for i, k := range r.order {
		if k == key {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	r.order = append(r.order, key) // le plus récent en fin
}

func (r *refLRU) get(key byte) (byte, bool) {
	v, ok := r.values[key]
	if ok {
		r.touch(key)
	}
	return v, ok
}

func (r *refLRU) put(key, val byte) {
	if _, ok := r.values[key]; ok {
		r.values[key] = val
		r.touch(key)
		return
	}
	r.values[key] = val
	r.touch(key)
	if len(r.order) > r.capacity {
		oldest := r.order[0]
		r.order = r.order[1:]
		delete(r.values, oldest)
	}
}

func (r *refLRU) len() int { return len(r.values) }
