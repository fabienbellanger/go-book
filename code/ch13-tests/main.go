// Démonstration du chapitre 13 : du code testable (Slugify) et une écriture de
// fichier (SaveLines). L'essentiel du chapitre est dans slugify_test.go.
// Lancement : depuis code/, `go run ./ch13-tests`
package main

import (
	"fmt"
	"os"
)

func main() {
	titles := []string{"Hello, World!", "  Go 1.26 : Top!  ", "Room 101"}

	slugs := make([]string, len(titles))
	for i, title := range titles {
		slugs[i] = Slugify(title)
		fmt.Printf("%-22q -> %q\n", title, slugs[i])
	}

	// Écrit les slugs dans un fichier temporaire, puis le supprime.
	dir, err := os.MkdirTemp("", "ch13")
	if err != nil {
		fmt.Println("erreur:", err)
		return
	}
	defer os.RemoveAll(dir)

	path, err := SaveLines(dir, "slugs.txt", slugs)
	if err != nil {
		fmt.Println("erreur:", err)
		return
	}
	fmt.Println("écrit dans :", path)
}
