package main

import "fmt"

// ByteSize est un type DÉFINI sur int64 : il possède sa propre identité de type
// (un ByteSize n'est pas un int64 et réciproquement sans conversion).
type ByteSize int64

// Multiples binaires définis avec iota.
// La 1re ligne (blank) absorbe iota=0 ; la ligne KB fixe le type ByteSize ET
// l'expression « 1 << (10 * iota) », que les lignes suivantes répètent.
const (
	_  ByteSize = iota             // 0 (ignoré)
	KB ByteSize = 1 << (10 * iota) // iota=1 -> 1<<10 = 1024
	MB                             // iota=2 -> 1<<20
	GB                             // iota=3 -> 1<<30
	TB                             // iota=4 -> 1<<40
	// Ajouter PB (1<<50), EB (1<<60) reste valide ; 1<<70 déborderait int64
	// et provoquerait une ERREUR DE COMPILATION.
)

// humanSize formate une taille en octets de façon lisible.
// Elle illustre les conversions explicites (ByteSize -> float64).
func humanSize(n ByteSize) string {
	switch {
	case n >= GB:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(GB))
	case n >= MB:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(MB))
	case n >= KB:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(KB))
	default:
		return fmt.Sprintf("%d B", int64(n))
	}
}
