package main

import (
	"math"
	"slices"
	"testing"
)

// almostEqual compare deux flottants à epsilon près (les == exacts sont fragiles).
func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestPointDistance(t *testing.T) {
	if d := (Point{0, 0}).Distance(Point{3, 4}); !almostEqual(d, 5) {
		t.Errorf("Distance({0,0},{3,4}) = %v ; attendu 5", d)
	}
	if d := (Point{1, 1}).Distance(Point{1, 1}); !almostEqual(d, 0) {
		t.Errorf("Distance(p,p) = %v ; attendu 0", d)
	}
	// Comparabilité des structs (== compile car tous les champs le sont).
	if (Point{1, 2}) != (Point{1, 2}) {
		t.Error("deux Points égaux devraient être ==")
	}
}

func TestRectangleArea(t *testing.T) {
	r := Rectangle{Min: Point{0, 0}, Max: Point{3, 4}}
	if !almostEqual(r.Width(), 3) || !almostEqual(r.Height(), 4) {
		t.Errorf("Width/Height = %v/%v ; attendu 3/4", r.Width(), r.Height())
	}
	if !almostEqual(r.Area(), 12) {
		t.Errorf("Area = %v ; attendu 12", r.Area())
	}
}

func TestAccountPointerReceiver(t *testing.T) {
	acc := Account{Owner: "alice"}
	acc.Deposit(100)
	acc.Deposit(50)
	if got := acc.Balance(); got != 150 {
		t.Fatalf("solde = %d ; attendu 150 (le récepteur pointeur doit muter l'original)", got)
	}
	if err := acc.Withdraw(30); err != nil {
		t.Fatalf("Withdraw(30) erreur inattendue : %v", err)
	}
	if got := acc.Balance(); got != 120 {
		t.Errorf("solde après retrait = %d ; attendu 120", got)
	}
	// Retrait excessif : erreur et solde inchangé.
	if err := acc.Withdraw(1000); err == nil {
		t.Error("Withdraw(1000) aurait dû échouer (fonds insuffisants)")
	}
	if got := acc.Balance(); got != 120 {
		t.Errorf("solde après retrait refusé = %d ; attendu 120 (inchangé)", got)
	}
}

func TestAuditedAccountEmbedding(t *testing.T) {
	a := &AuditedAccount{Account: Account{Owner: "bob"}}

	// Champ et méthode promus depuis Account.
	if a.Owner != "bob" {
		t.Errorf("Owner promu = %q ; attendu \"bob\"", a.Owner)
	}

	a.Deposit(200)                         // méthode REDÉFINIE : délègue + journalise
	if err := a.Withdraw(50); err != nil { // méthode PROMUE : non journalisée
		t.Fatalf("Withdraw promu, erreur inattendue : %v", err)
	}

	if got := a.Balance(); got != 150 {
		t.Errorf("Balance promu = %d ; attendu 150", got)
	}
	// Seul Deposit (redéfini) journalise ; Withdraw (promu) n'apparaît pas.
	if want := []string{"deposit 200"}; !slices.Equal(a.Log(), want) {
		t.Errorf("journal = %v ; attendu %v", a.Log(), want)
	}
}

func TestFieldSizesPadding(t *testing.T) {
	padded, packed := FieldSizes()
	if padded != 24 {
		t.Errorf("Sizeof(Padded) = %d ; attendu 24", padded)
	}
	if packed != 16 {
		t.Errorf("Sizeof(Packed) = %d ; attendu 16", packed)
	}
	if packed >= padded {
		t.Error("le réagencement des champs devrait réduire la taille")
	}
}
