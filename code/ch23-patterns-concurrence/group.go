package main

import (
	"context"
	"sync"
)

// Group exécute des tâches concurrentes, capture la PREMIÈRE erreur et annule
// les autres via le contexte. C'est une version minimale, en stdlib pure, de
// golang.org/x/sync/errgroup — la référence à utiliser en production.
type Group struct {
	wg     sync.WaitGroup
	once   sync.Once
	err    error
	cancel context.CancelFunc
}

// NewGroup crée un groupe et un contexte dérivé, annulé dès la première erreur
// (ou à l'appel de Wait).
func NewGroup(parent context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancel(parent)
	return &Group{cancel: cancel}, ctx
}

// Go lance une tâche. La première qui échoue enregistre son erreur (via Once,
// pour ne garder que la première) et annule le contexte partagé, invitant les
// autres tâches à s'arrêter au plus tôt.
func (g *Group) Go(f func() error) {
	g.wg.Go(func() {
		if err := f(); err != nil {
			g.once.Do(func() {
				g.err = err
				g.cancel()
			})
		}
	})
}

// Wait attend toutes les tâches et renvoie la première erreur survenue (ou nil).
func (g *Group) Wait() error {
	g.wg.Wait()
	g.cancel() // libère le contexte dans tous les cas
	return g.err
}
