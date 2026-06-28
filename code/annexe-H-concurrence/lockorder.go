package main

import "sync"

// Account est un compte protégé par son propre verrou. Le champ id sert à imposer
// un ORDRE GLOBAL de verrouillage entre comptes — la clé pour éliminer le deadlock
// classique « AB-BA » (une goroutine tient A et attend B, l'autre tient B et
// attend A).
type Account struct {
	id      int
	mu      sync.Mutex
	balance int64
}

// NewAccount crée un compte d'identifiant et de solde donnés.
func NewAccount(id int, balance int64) *Account {
	return &Account{id: id, balance: balance}
}

// Transfer déplace amount de from vers to SANS risque d'interblocage : les deux
// verrous sont toujours acquis dans le même ordre (id croissant), quel que soit
// le sens du virement. Deux virements croisés (A->B et B->A) lancés en parallèle
// ne peuvent donc plus se bloquer mutuellement.
func Transfer(from, to *Account, amount int64) {
	if from == to {
		return // même compte : éviter un double Lock du même mutex (auto-interblocage)
	}
	// Toujours verrouiller le plus petit id en premier : c'est l'ordre GLOBAL.
	first, second := from, to
	if first.id > second.id {
		first, second = second, first
	}
	first.mu.Lock()
	defer first.mu.Unlock()
	second.mu.Lock()
	defer second.mu.Unlock()

	from.balance -= amount
	to.balance += amount
}

// Balance lit le solde sous verrou.
func (a *Account) Balance() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.balance
}
