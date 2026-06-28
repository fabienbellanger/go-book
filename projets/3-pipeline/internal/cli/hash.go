package cli

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"iter"
	"os"
	"strings"
)

// hashResult associe un chemin à son empreinte SHA-256.
type hashResult struct {
	path string
	sum  string
}

// pathsFrom produit la séquence des chemins à traiter : les arguments s'il y en
// a, sinon les lignes non vides de stdin. C'est une iter.Seq (Go 1.23), donc la
// source est *paresseuse* : le pipeline tire les chemins au rythme des workers,
// ce qui fait remonter la pression arrière jusqu'à la lecture de stdin.
func pathsFrom(args []string, stdin io.Reader) iter.Seq[string] {
	return func(yield func(string) bool) {
		if len(args) > 0 {
			for _, a := range args {
				if !yield(a) {
					return
				}
			}
			return
		}
		sc := bufio.NewScanner(stdin)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			if !yield(line) {
				return
			}
		}
		// Une erreur de lecture de stdin termine simplement la séquence.
		if err := sc.Err(); err != nil {
			return
		}
	}
}

// hashFile est l'étape (Stage) du pipeline : elle calcule le SHA-256 d'un
// fichier. La lecture respecte l'annulation du contexte (copyCtx), pour qu'un
// gros fichier n'empêche pas un arrêt rapide.
func hashFile(ctx context.Context, path string) (hashResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return hashResult{}, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := copyCtx(ctx, h, f); err != nil {
		return hashResult{}, err
	}
	return hashResult{path: path, sum: hex.EncodeToString(h.Sum(nil))}, nil
}

// copyCtx copie src vers dst par blocs, en vérifiant l'annulation entre chaque
// bloc. C'est l'équivalent annulable de io.Copy.
func copyCtx(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return total, werr
			}
			total += int64(n)
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}
