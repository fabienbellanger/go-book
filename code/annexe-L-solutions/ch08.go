package main

// ch08Account illustre le récepteur POINTEUR : Deposit mute le solde en place.
// Avec un récepteur valeur, la méthode travaillerait sur une COPIE et le solde
// appelant ne bougerait pas (exercice 1).
type ch08Account struct {
	balance int
}

func (a *ch08Account) Deposit(n int) { a.balance += n }
func (a ch08Account) Balance() int   { return a.balance }

// ch08Padded et ch08Packed contiennent les mêmes champs, mais l'ordre change la
// taille à cause de l'alignement (exercice 3). En groupant les champs du plus
// grand au plus petit, on supprime le rembourrage (padding) intermédiaire.
type ch08Padded struct {
	a bool  // 1 octet + 7 de padding avant le int64
	b int64 // 8 octets
	c bool  // 1 octet + 7 de padding en fin
} // -> 24 octets

type ch08Packed struct {
	b int64 // 8 octets
	a bool  // 1
	c bool  // 1 (+ 6 de padding final seulement)
} // -> 16 octets
