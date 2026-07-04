// Command ch50-fichiers-fs illustre la manipulation de fichiers et de systèmes
// de fichiers avec la bibliothèque standard : lecture/écriture, écriture
// atomique via Rename, parcours de répertoire (ReadDir/WalkDir), l'abstraction
// io/fs côté consommateur, et le confinement 1.24 os.Root contre la traversée
// de chemin.
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// writeFileAtomic écrit data dans path de façon ATOMIQUE : on écrit d'abord dans
// un fichier temporaire situé dans le MÊME répertoire, on le synchronise, puis on
// le renomme sur path. Rename est atomique sur un même volume : un lecteur voit
// soit l'ancien contenu, soit le nouveau — jamais un fichier tronqué, même si le
// programme meurt en cours d'écriture.
func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	// Le temporaire DOIT être dans le même dossier que la cible : Rename n'est
	// atomique qu'à l'intérieur d'un même système de fichiers.
	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// En cas d'échec après création, on nettoie le temporaire résiduel.
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	// Sync force l'écriture sur le disque avant le Rename (durabilité).
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	// Le basculement final : instantané et atomique.
	return os.Rename(tmpName, path)
}

// listDir renvoie, triés, les noms des entrées de dir en distinguant les
// répertoires. os.ReadDir renvoie des fs.DirEntry, plus légers qu'un FileInfo :
// le type se lit sans appel Stat supplémentaire.
func listDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// countGoFiles parcourt récursivement un système de fichiers fs.FS et compte les
// fichiers d'extension .go. Écrire la fonction contre fs.FS (et non contre le
// disque) la rend triviale à tester avec fstest.MapFS, et compatible avec un
// embed.FS (🔁 ch46).
func countGoFiles(fsys fs.FS) (int, error) {
	count := 0
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // TOUJOURS gérer l'erreur passée au callback
		}
		if !d.IsDir() && filepath.Ext(path) == ".go" {
			count++
		}
		return nil
	})
	return count, err
}

// safeReadUnder ouvre name à l'intérieur de root SANS pouvoir en sortir : os.Root
// (1.24) confine toutes les opérations au répertoire ouvert. Un name malveillant
// comme "../../etc/passwd" échoue au lieu de fuir hors de root — la défense idéale
// quand le chemin vient d'une entrée utilisateur.
func safeReadUnder(root *os.Root, name string) ([]byte, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f) // un *os.File confiné reste un io.Reader (🔁 ch41)
}

func main() {
	dir, err := os.MkdirTemp("", "ch50-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	// 1. Écriture atomique puis relecture.
	cfg := filepath.Join(dir, "config.txt")
	if err := writeFileAtomic(cfg, []byte("mode=prod\n"), 0o644); err != nil {
		panic(err)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("config = %q\n", data)

	// 2. Créer une arborescence et la lister.
	if err := os.MkdirAll(filepath.Join(dir, "pkg"), 0o755); err != nil {
		panic(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "pkg", "a.go"), []byte("package pkg\n"), 0o644)
	names, err := listDir(dir)
	if err != nil {
		panic(err)
	}
	fmt.Println("entrées :", names)

	// 3. Parcours récursif via fs.FS (os.DirFS expose un dossier en fs.FS).
	n, err := countGoFiles(os.DirFS(dir))
	if err != nil {
		panic(err)
	}
	fmt.Println("fichiers .go :", n)

	// 4. Confinement os.Root : lecture sûre, traversée refusée.
	root, err := os.OpenRoot(dir)
	if err != nil {
		panic(err)
	}
	defer root.Close()
	if _, err := safeReadUnder(root, "config.txt"); err != nil {
		panic(err)
	}
	fmt.Println("lecture confinée OK")
	if _, err := safeReadUnder(root, "../../../etc/hosts"); err != nil {
		fmt.Println("traversée refusée comme prévu")
	}
}
