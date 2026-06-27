package main

import "fmt"

func main() {
	// 1) Publication par canal : la valeur reçue est entièrement visible.
	c := PublishViaChannel()
	fmt.Printf("via canal   : %s:%d ready=%t\n", c.Addr, c.Port, c.Ready)

	// 2) Initialisation paresseuse via sync.Once : même pointeur à chaque appel.
	a, b := GetConfig(), GetConfig()
	fmt.Printf("via Once    : même instance ? %t (%s:%d)\n", a == b, a.Addr, a.Port)

	// 3) Publication sans verrou via atomic.Pointer.
	fmt.Printf("atomic avant: %v\n", LoadConfig())
	SwapConfig(buildConfig())
	fmt.Printf("atomic après: %s:%d\n", LoadConfig().Addr, LoadConfig().Port)

	// Lancez « go run -race . » : aucune course, car chaque partage passe par
	// un point de synchronisation (canal, Once, atomic).
}
