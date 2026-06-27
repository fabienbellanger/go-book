// Package main illustre l'ordre d'initialisation d'un package : variables
// d'abord (dans l'ordre de leurs DÉPENDANCES, pas du source), puis les init().
package main

import "runtime"

// initOrder enregistre, dans l'ordre, ce qui s'est initialisé. C'est notre
// « traceur » : chaque étape d'init y ajoute son nom via track.
var initOrder []string

// track note une étape et renvoie sa valeur, pour pouvoir être appelé depuis
// l'initialiseur d'une variable de package.
func track(name string) string {
	initOrder = append(initOrder, name)
	return name
}

// derived DÉPEND de base (il l'utilise dans son initialiseur). Bien qu'il soit
// déclaré AVANT base, Go initialise base en premier : l'ordre suit le graphe de
// dépendances, pas l'ordre du texte.
var derived = track("derived(after " + base + ")")
var base = track("base")

// Les deux init() s'exécutent APRÈS toutes les variables, dans l'ordre du source.
func init() { track("init #1") }
func init() { track("init #2") }

// InitOrder renvoie la trace d'initialisation observée.
func InitOrder() []string { return initOrder }

// RuntimeInfo regroupe quelques constantes décidées au démarrage du runtime.
type RuntimeInfo struct {
	Version    string // version du toolchain ayant compilé le binaire
	NumCPU     int    // cœurs logiques vus par le runtime
	GOMAXPROCS int    // P actifs (par défaut = NumCPU, ou la limite cgroup)
	GOOS       string
	GOARCH     string
}

// CurrentRuntime lit l'état du runtime sans le modifier (GOMAXPROCS(0) = lecture).
func CurrentRuntime() RuntimeInfo {
	return RuntimeInfo{
		Version:    runtime.Version(),
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}
}
