package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SaveLines écrit chaque ligne (suivie d'un '\n') dans dir/name et renvoie le chemin
// complet du fichier créé. C'est la fonction exercée avec t.TempDir dans les tests.
func SaveLines(dir, name string, lines []string) (string, error) {
	path := filepath.Join(dir, name)
	data := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return "", fmt.Errorf("écriture %s: %w", path, err)
	}
	return path, nil
}
