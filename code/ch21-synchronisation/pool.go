package main

import (
	"bytes"
	"strconv"
	"sync"
)

// bufPool recycle des bytes.Buffer pour éviter d'en réallouer à chaque appel.
// sync.Pool est un cache d'objets temporaires (par processeur logique) : Get
// renvoie un objet recyclé — ou en fabrique un via New — et Put le rend au pool.
// Le GC peut vider le pool à tout moment : n'y stockez que du jetable.
var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// joinInts concatène des entiers en réutilisant un buffer du pool : zéro
// allocation de buffer en régime établi, et c'est sûr en concurrence.
func joinInts(nums []int, sep string) string {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()            // un objet recyclé peut être « sale » : on le remet à zéro
	defer bufPool.Put(b) // rendre au pool pour le prochain appelant
	for i, n := range nums {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(strconv.Itoa(n))
	}
	return b.String()
}
