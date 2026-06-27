// Démonstrations du chapitre 8 : structs, méthodes (valeur vs pointeur), embedding,
// padding. Lancement : depuis code/, `go run ./ch08-structs`
package main

import (
	"fmt"
	"unsafe"
)

func main() {
	// =========================================================================
	// STRUCTS : déclaration, littéraux, zero value, comparaison
	// =========================================================================

	// --- Littéraux : positionnel, nommé, imbriqué, pointeur.
	p1 := Point{1, 2}                                     // positionnel (ordre des champs)
	p2 := Point{X: 3}                                     // nommé (Y prend sa zero value 0)
	rect := Rectangle{Min: Point{0, 0}, Max: Point{3, 4}} // imbriqué
	pp := &Point{X: 5, Y: 6}                              // pointeur vers struct
	fmt.Printf("struct : p1=%v p2=%v rect=%+v pp=%v\n", p1, p2, rect, *pp)

	// --- Zero value : tous les champs à leur valeur zéro, sans initialisation.
	var zero Point
	fmt.Printf("struct : zero value = %+v\n", zero)

	// --- Accès aux champs via un pointeur : déréférencement AUTOMATIQUE.
	pp.X = 50 // équivaut à (*pp).X = 50
	fmt.Printf("struct : pp.X après mutation via pointeur = %v\n", pp.X)

	// --- Comparaison : == si tous les champs sont comparables.
	fmt.Printf("struct : Point{1,2}==Point{1,2} ? %t\n", Point{1, 2} == Point{1, 2})

	// --- Formats utiles pour le debug.
	fmt.Printf("struct : %%v=%v  %%+v=%+v  %%#v=%#v\n", p1, p1, p1)

	// =========================================================================
	// MÉTHODES : récepteur valeur vs pointeur
	// =========================================================================

	fmt.Printf("\nméthode: Distance({0,0},{3,4}) = %v\n", Point{0, 0}.Distance(Point{3, 4}))
	fmt.Printf("méthode: Rectangle Area=%v Width=%v Height=%v\n", rect.Area(), rect.Width(), rect.Height())

	// --- Récepteur POINTEUR : la mutation persiste.
	acc := Account{Owner: "alice"}
	acc.Deposit(100) // auto-adressage : (&acc).Deposit(100)
	acc.Deposit(50)
	if err := acc.Withdraw(30); err != nil {
		fmt.Println("erreur:", err)
	}
	fmt.Printf("méthode: solde après dépôts/retrait (ptr) = %d\n", acc.Balance())
	if err := acc.Withdraw(1000); err != nil {
		fmt.Println("méthode: retrait refusé ->", err)
	}

	// =========================================================================
	// EMBEDDING : composition, promotion, override
	// =========================================================================

	audited := &AuditedAccount{Account: Account{Owner: "bob"}}
	audited.Deposit(200)     // Deposit REDÉFINI : délègue + journalise
	_ = audited.Withdraw(50) // Withdraw PROMU depuis Account (non journalisé)
	fmt.Printf("\nembed  : Owner (promu)=%q Balance (promu)=%d\n", audited.Owner, audited.Balance())
	fmt.Printf("embed  : journal (Deposit redéfini seulement) = %v\n", audited.Log())
	fmt.Printf("embed  : accès explicite au champ embarqué : audited.Account.Owner=%q\n", audited.Account.Owner)

	// =========================================================================
	// STRUCTS VIDES & PADDING
	// =========================================================================

	fmt.Printf("\nlayout : Sizeof(struct{}{}) = %d (le struct vide ne pèse rien)\n", unsafe.Sizeof(struct{}{}))
	padded, packed := FieldSizes()
	fmt.Printf("layout : Sizeof(Padded)=%d  Sizeof(Packed)=%d  -> réordonner économise %d octets\n",
		padded, packed, padded-packed)

	// --- Struct anonyme (sans nom de type) : pratique pour un regroupement local.
	pair := struct{ Key, Value string }{Key: "go", Value: "1.26"}
	fmt.Printf("layout : struct anonyme = %+v\n", pair)
}
