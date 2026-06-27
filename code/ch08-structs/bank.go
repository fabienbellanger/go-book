package main

import "fmt"

// Account détient un solde en centimes. Ses méthodes qui MODIFIENT l'état utilisent un
// récepteur POINTEUR : un récepteur valeur modifierait une copie, laissant le compte de
// l'appelant inchangé.
//
// 💡 Règle d'idiome : dès qu'UNE méthode a besoin d'un pointeur, on met TOUTES les
// méthodes du type sur pointeur, pour la cohérence du method set (pas de mélange).
type Account struct {
	Owner   string
	balance int // non exporté : centimes
}

// Deposit crédite le compte.
func (a *Account) Deposit(cents int) {
	a.balance += cents
}

// Withdraw débite le compte, ou renvoie une erreur si les fonds sont insuffisants.
func (a *Account) Withdraw(cents int) error {
	if cents > a.balance {
		return fmt.Errorf("fonds insuffisants : solde %d, demandé %d", a.balance, cents)
	}
	a.balance -= cents
	return nil
}

// Balance expose le solde (non exporté) en lecture seule.
func (a *Account) Balance() int { return a.balance }

// AuditedAccount EMBARQUE Account (champ anonyme). Par PROMOTION, il expose
// directement les champs (Owner) et méthodes (Deposit, Withdraw, Balance) d'Account,
// et y ajoute un journal. Il REDÉFINIT Deposit pour tracer l'opération, en déléguant
// explicitement à la méthode promue via a.Account.Deposit.
type AuditedAccount struct {
	Account          // embedding : champ sans nom
	log     []string // historique des opérations
}

// Deposit enveloppe la méthode promue et journalise l'opération.
func (a *AuditedAccount) Deposit(cents int) {
	a.Account.Deposit(cents) // délégation explicite à la méthode embarquée
	a.log = append(a.log, fmt.Sprintf("deposit %d", cents))
}

// Log renvoie les opérations enregistrées par ce compte audité.
func (a *AuditedAccount) Log() []string { return a.log }
