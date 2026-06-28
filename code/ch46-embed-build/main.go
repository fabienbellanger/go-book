// Command ch46-embed-build illustre l'embarquement d'assets (//go:embed),
// la lecture des informations de build (runtime/debug) et les build tags.
//
// Construire avec une version injectée :
//
//	go build -ldflags="-s -w -X main.version=1.2.3" -trimpath ./ch46-embed-build
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"runtime/debug"
	"strings"
	"text/template"
)

// version est la cible classique de l'injection par l'éditeur de liens :
// « go build -ldflags="-X main.version=1.2.3" ». La valeur par défaut sert
// pendant le développement (binaire compilé sans -ldflags).
var version = "dev"

// embeddedVersion embarque un fichier unique dans une string. Le cas string/[]byte
// exige l'import blanc « _ "embed" » ; ici l'import est déjà tiré par embed.FS.
//
//go:embed version.txt
var embeddedVersion string

// templatesFS embarque tout un sous-arbre dans un système de fichiers en lecture
// seule. Le motif récupère chaque fichier sous « templates/ ».
//
//go:embed templates
var templatesFS embed.FS

// renderWelcome rend le gabarit embarqué « templates/welcome.txt ».
func renderWelcome(name string) (string, error) {
	t, err := template.ParseFS(templatesFS, "templates/welcome.txt")
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	data := struct{ Name, Version string }{Name: name, Version: buildVersion()}
	if err := t.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// buildVersion privilégie la version injectée par -ldflags ; à défaut, elle se
// rabat sur le fichier embarqué (lui-même versionné dans le dépôt).
func buildVersion() string {
	if version != "dev" {
		return version
	}
	return strings.TrimSpace(embeddedVersion)
}

// vcsRevision lit le hash de commit que « go build » injecte automatiquement
// dans les BuildInfo.Settings (clé « vcs.revision »). Pratique pour tracer un
// binaire sans aucun -ldflags. Renvoie "" hors d'un build module/VCS (ex. go test).
func vcsRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
}

func main() {
	fmt.Println("version    :", buildVersion())
	fmt.Println("feature    :", featureName()) // dépend des build tags (voir feature_*.go)
	if rev := vcsRevision(); rev != "" {
		fmt.Println("commit     :", rev)
	}

	msg, err := renderWelcome("Go")
	if err != nil {
		fmt.Println("erreur de rendu:", err)
		return
	}
	fmt.Print(msg)

	// L'embed.FS implémente fs.FS : on peut le parcourir comme un vrai disque.
	_ = fs.WalkDir(templatesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			fmt.Println("asset      :", path)
		}
		return nil
	})
}
