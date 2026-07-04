package main

// ch05Compose renvoie la composition f∘g : x -> f(g(x)) (exercice 3). Les
// fonctions sont des valeurs de première classe ; on peut donc en renvoyer une
// nouvelle qui capture f et g par closure.
func ch05Compose(f, g func(int) int) func(int) int {
	return func(x int) int { return f(g(x)) }
}

// ch05DivMod renvoie quotient ET reste : un appel multivalué.
func ch05DivMod(a, b int) (int, int) {
	return a / b, a % b
}

// ch05Sum additionne deux entiers. `ch05Sum(ch05DivMod(17, 5))` compile car les
// DEUX valeurs de retour de DivMod alimentent directement les deux paramètres
// de Sum (exercice 4). En revanche `q := ch05DivMod(17, 5)` ne compile pas : on
// ne peut pas affecter un résultat à DEUX valeurs à une seule variable.
func ch05Sum(a, b int) int {
	return a + b
}

// --- Options fonctionnelles (exercice 1 : ajouter WithMaxConns) ---

type ch05Server struct {
	port     int
	maxConns int
}

type ch05Option func(*ch05Server)

func ch05WithPort(p int) ch05Option     { return func(s *ch05Server) { s.port = p } }
func ch05WithMaxConns(n int) ch05Option { return func(s *ch05Server) { s.maxConns = n } }

// ch05NewServer applique les options dans l'ordre. Ajouter WithMaxConns ne
// change AUCUN appel existant : c'est tout l'intérêt du patron.
func ch05NewServer(opts ...ch05Option) *ch05Server {
	s := &ch05Server{port: 8080, maxConns: 100} // valeurs par défaut
	for _, opt := range opts {
		opt(s)
	}
	return s
}
