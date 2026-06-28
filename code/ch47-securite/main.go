// Command ch47-securite illustre quelques réflexes de sécurité côté code, tous
// avec la bibliothèque standard : jetons aléatoires sûrs, comparaison en temps
// constant, échappement contextuel des templates, confinement d'accès fichier et
// configuration TLS durcie. Le chapitre 47 détaille le volet « chaîne
// d'approvisionnement » (go.sum, govulncheck, builds reproductibles).
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	htmltemplate "html/template"
	"io"
	"io/fs"
	"os"
	"strings"
	texttemplate "text/template"
)

// newToken renvoie un jeton secret cryptographiquement sûr. On s'appuie sur
// crypto/rand (PAS math/rand) : rand.Text garantit au moins 128 bits d'entropie.
// 🆕 Go 1.24.
func newToken() string {
	return rand.Text()
}

// equalTokens compare deux jetons en TEMPS CONSTANT : la durée ne dépend pas de
// l'endroit où ils diffèrent, ce qui ferme la porte aux attaques temporelles.
// Un simple « a == b » s'arrête au premier octet différent et fuit cette position.
func equalTokens(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// renderUserHTML rend une donnée fournie par l'utilisateur via html/template, qui
// échappe AUTOMATIQUEMENT selon le contexte (ici HTML) : « <script> » devient
// « &lt;script&gt; ». C'est la défense de base contre le XSS.
func renderUserHTML(name string) (string, error) {
	t := htmltemplate.Must(htmltemplate.New("greet").Parse(`<p>Bonjour {{.}}</p>`))
	var sb strings.Builder
	if err := t.Execute(&sb, name); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// renderUserText fait la même chose avec text/template, qui N'ÉCHAPPE RIEN. Rendu
// dans une page HTML, sa sortie est une faille XSS. À n'utiliser que pour du texte
// non-HTML (config, code généré, etc.).
func renderUserText(name string) (string, error) {
	t := texttemplate.Must(texttemplate.New("greet").Parse(`<p>Bonjour {{.}}</p>`))
	var sb strings.Builder
	if err := t.Execute(&sb, name); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// isSafeRelPath rejette les chemins susceptibles de remonter l'arborescence
// (« ../etc/passwd ») ou absolus. fs.ValidPath impose un chemin relatif, propre,
// sans élément « .. » ni « / » initial.
func isSafeRelPath(name string) bool {
	return fs.ValidPath(name)
}

// readWithinRoot lit un fichier en restant CONFINÉ sous dir : os.Root garantit
// qu'aucun chemin (ni lien symbolique) ne s'échappe du sous-arbre, même si name
// contient « .. ». 🆕 Go 1.24. C'est plus robuste qu'un filepath.Clean manuel.
func readWithinRoot(dir, name string) ([]byte, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// hardenedTLSConfig renvoie une configuration TLS durcie : on impose au moins
// TLS 1.2 et on laisse InsecureSkipVerify à false (défaut). ⚠️ Passer
// InsecureSkipVerify à true désactive la vérification de certificat = interdit en prod.
func hardenedTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}

func main() {
	tok := newToken()
	fmt.Printf("jeton          : %s\n", tok)
	fmt.Printf("compare (==)   : %t / %t\n", equalTokens(tok, tok), equalTokens(tok, newToken()))

	html, _ := renderUserHTML("<script>alert(1)</script>")
	text, _ := renderUserText("<script>alert(1)</script>")
	fmt.Printf("html/template  : %s\n", html)
	fmt.Printf("text/template  : %s  (<-- non échappé !)\n", text)

	fmt.Printf("chemin sûr     : %t / %t\n", isSafeRelPath("notes.txt"), isSafeRelPath("../etc/passwd"))

	cfg := hardenedTLSConfig()
	fmt.Printf("TLS MinVersion : 0x%04x (TLS 1.2)\n", cfg.MinVersion)
}
