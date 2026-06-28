// Package example montre enumgen en situation : deux énumérations annotées et
// le fichier example_enum.go produit par « go generate ».
//
// Pour régénérer (le binaire enumgen doit être dans le PATH — voir « make install ») :
//
//	go generate ./...
package example

//go:generate enumgen

// Color est une énumération à valeurs contiguës (iota). Le préfixe « Color » est
// rogné des libellés : ColorRed.String() rend "Red".
//
//enumgen:stringer trimprefix=Color
type Color int

const (
	ColorRed Color = iota
	ColorGreen
	ColorBlue
)

// Priority montre des valeurs explicites et non contiguës : String() s'appuie
// sur une map, donc l'ordre et les trous sont sans importance.
//
//enumgen:stringer trimprefix=Priority
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityMedium   Priority = 5
	PriorityHigh     Priority = 10
	PriorityCritical Priority = 20
)
