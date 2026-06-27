// Package main illustre le modèle mémoire : comment GARANTIR qu'une écriture
// faite par une goroutine est VISIBLE par une autre. La règle est « happens-before »,
// établie par les canaux, sync et sync/atomic — jamais par le simple passage du temps.
package main

import (
	"sync"
	"sync/atomic"
)

// Config est une donnée « lourde » construite une fois puis partagée.
type Config struct {
	Addr  string
	Port  int
	Ready bool
}

// buildConfig simule une construction coûteuse (plusieurs écritures de champs).
func buildConfig() *Config {
	c := &Config{}
	c.Addr = "localhost" // ces trois écritures doivent être visibles EN BLOC
	c.Port = 8080        // par quiconque reçoit le pointeur après synchronisation
	c.Ready = true
	return c
}

// PublishViaChannel : une goroutine construit la config, puis l'ENVOIE sur un
// canal. La réception établit un « happens-before » : tout ce que la goroutine a
// écrit AVANT l'envoi est garanti visible APRÈS la réception. Pas de course.
func PublishViaChannel() *Config {
	ch := make(chan *Config)
	go func() {
		ch <- buildConfig() // l'envoi « happens-before » la réception
	}()
	return <-ch // ici, *Config est entièrement et sûrement visible
}

// --- Initialisation paresseuse correcte : sync.Once ---
//
// Le « double-checked locking » naïf (lire un flag sans verrou, puis verrouiller)
// est BUGGÉ en Go : sans synchronisation, rien ne garantit la visibilité des
// écritures du constructeur. sync.Once résout le problème : le retour de Do(f)
// « happens-before » le retour de tout autre Do — donc cfg est vu complet.

var (
	once sync.Once
	cfg  *Config
)

// GetConfig construit la config au premier appel, puis renvoie toujours la même.
// Sûr et correct même appelé par mille goroutines en parallèle.
func GetConfig() *Config {
	once.Do(func() { cfg = buildConfig() })
	return cfg
}

// --- Publication sans verrou : atomic.Pointer ---
//
// Pour un drapeau/pointeur partagé mis à jour à la volée, atomic.Pointer fournit
// la barrière mémoire : un Store « happens-before » le Load qui le lit.

var current atomic.Pointer[Config]

// SwapConfig remplace atomiquement la config courante (lecteurs sans verrou).
func SwapConfig(c *Config) { current.Store(c) }

// LoadConfig lit la config courante (nil tant qu'aucun Store n'a eu lieu).
func LoadConfig() *Config { return current.Load() }
