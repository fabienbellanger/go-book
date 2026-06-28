// Package lru fournit un cache générique borné à éviction LRU (Least Recently
// Used) : quand la capacité est atteinte, l'entrée la moins récemment utilisée
// est évincée.
//
// Construire le cache avec [New] ; le type zéro [Cache] n'est pas utilisable.
// Cache n'est pas sûr pour un usage concurrent.
package lru

import "container/list"

// entry est la valeur stockée dans chaque élément de liste. On y garde la clé
// pour pouvoir retrouver (et supprimer) l'entrée de la map lors de l'éviction.
type entry[K comparable, V any] struct {
	key   K
	value V
}

// Cache est un cache LRU associant des clés K à des valeurs V.
//
// L'ordre de récence est tenu par une liste doublement chaînée : l'entrée la
// plus récente est en tête (Front), la plus ancienne en queue (Back). La map
// donne un accès O(1) à l'élément de liste de chaque clé.
type Cache[K comparable, V any] struct {
	capacity int
	ll       *list.List // de la plus récente (Front) à la plus ancienne (Back)
	items    map[K]*list.Element
}

// New crée un cache de capacité maximale donnée. capacity doit être ≥ 1.
func New[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity < 1 {
		panic("lru: la capacité doit être >= 1")
	}
	return &Cache[K, V]{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[K]*list.Element, capacity),
	}
}

// Get renvoie la valeur associée à key et la marque comme la plus récente.
// ok vaut false si la clé est absente.
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	if el, hit := c.items[key]; hit {
		c.ll.MoveToFront(el)
		return el.Value.(*entry[K, V]).value, true
	}
	return value, false
}

// Put insère ou met à jour une association, et la marque comme la plus récente.
// evicted indique si l'insertion a provoqué l'éviction d'une autre entrée.
func (c *Cache[K, V]) Put(key K, value V) (evicted bool) {
	if el, hit := c.items[key]; hit {
		c.ll.MoveToFront(el)
		el.Value.(*entry[K, V]).value = value
		return false
	}

	el := c.ll.PushFront(&entry[K, V]{key: key, value: value})
	c.items[key] = el

	if c.ll.Len() > c.capacity {
		c.evictOldest()
		return true
	}
	return false
}

// Contains indique si key est présente, sans modifier la récence (contrairement
// à [Cache.Get]).
func (c *Cache[K, V]) Contains(key K) bool {
	_, ok := c.items[key]
	return ok
}

// Remove supprime une entrée et renvoie true si elle existait.
func (c *Cache[K, V]) Remove(key K) bool {
	el, ok := c.items[key]
	if !ok {
		return false
	}
	c.removeElement(el)
	return true
}

// Len renvoie le nombre d'entrées actuellement en cache.
func (c *Cache[K, V]) Len() int { return c.ll.Len() }

// Cap renvoie la capacité maximale du cache.
func (c *Cache[K, V]) Cap() int { return c.capacity }

// Keys renvoie les clés de la plus récente à la plus ancienne.
func (c *Cache[K, V]) Keys() []K {
	keys := make([]K, 0, c.ll.Len())
	for el := c.ll.Front(); el != nil; el = el.Next() {
		keys = append(keys, el.Value.(*entry[K, V]).key)
	}
	return keys
}

// evictOldest retire l'entrée en queue de liste (la moins récemment utilisée).
func (c *Cache[K, V]) evictOldest() {
	if el := c.ll.Back(); el != nil {
		c.removeElement(el)
	}
}

// removeElement retire un élément de la liste et de la map.
func (c *Cache[K, V]) removeElement(el *list.Element) {
	c.ll.Remove(el)
	delete(c.items, el.Value.(*entry[K, V]).key)
}
