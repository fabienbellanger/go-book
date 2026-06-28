package analyze

import (
	"strings"
	"testing"
)

// corpus fabrique un texte de taille réaliste (~ quelques centaines de Kio) en
// répétant un paragraphe. Assez gros pour que le profil CPU soit lisible.
func corpus() string {
	const para = `La concurrence en Go repose sur les goroutines et les canaux.
Une goroutine est légère ; le runtime en ordonnance des milliers sur quelques
threads. Les canaux transmettent des valeurs entre goroutines, et le mot-clé
select attend sur plusieurs canaux. Le ramasse-miettes libère la mémoire qui
n'est plus atteignable, sans intervention du programmeur. Profiler un programme
Go, c'est mesurer où passent le temps CPU et les allocations, puis itérer. `
	return strings.Repeat(para, 2000)
}

// On compare les deux implémentations avec b.Loop() (API 1.24) : le corpus est
// préparé HORS de la boucle chronométrée, et b.Loop empêche l'élimination du
// résultat par le compilateur.
//
//	go test -run=^$ -bench=TopWords -benchmem ./internal/analyze
//	# puis benchstat pour comparer V1 et V2.
func BenchmarkTopWordsRegexp(b *testing.B) {
	text := corpus()
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		_ = TopWordsRegexp(text, 10)
	}
}

func BenchmarkTopWordsScan(b *testing.B) {
	text := corpus()
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		_ = TopWordsScan(text, 10)
	}
}
