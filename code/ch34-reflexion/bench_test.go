package main

import "testing"

// fillDirect : l'équivalent ÉCRIT À LA MAIN de FillDefaults, sans réflexion.
func fillDirect(s *Server) {
	if s.Host == "" {
		s.Host = "localhost"
	}
	if s.Port == 0 {
		s.Port = 8080
	}
}

// La réflexion paie l'introspection à chaque appel : nettement plus lente que le code
// direct. D'où la règle « confiner la réflexion aux frontières » (décodage, sérialisation).
func BenchmarkFillReflect(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var s Server
		_ = FillDefaults(&s)
	}
}

func BenchmarkFillDirect(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var s Server
		fillDirect(&s)
	}
}
