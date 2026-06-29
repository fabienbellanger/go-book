// Package sommaire lit SOMMAIRE.md et en extrait l'arbre de navigation du livre
// (parties → pages), qui fixe l'ordre canonique du livre.
package sommaire

import (
	"bufio"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"example.com/gobook-site/internal/model"
)

var (
	// En-tête de partie : « ## Partie I — … » ou « ## Annexes ».
	rePart = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	// Entrée de liste avec un lien Markdown : « - … [titre](chemin) … ».
	reLink = regexp.MustCompile(`^\s*-\s+.*?\[([^\]]+)\]\(([^)]+)\)`)
)

// Parse lit un flux SOMMAIRE.md et renvoie le livre partiellement construit :
// les parties ordonnées, chaque page avec son SrcPath/OutPath et un titre
// provisoire (le titre définitif viendra du H1 au rendu). Les entrées pointant
// vers un dossier (projets) deviennent des Link, non des pages rendues.
func Parse(r io.Reader, title string) (*model.Book, error) {
	book := &model.Book{Title: title}
	var cur *model.Part
	// Mémorise la dernière partie pour rattacher une éventuelle description.
	var pendingDescFor *model.Part

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)

		// Le titre H1 du document n'est pas une partie.
		if strings.HasPrefix(line, "# ") {
			continue
		}

		if m := rePart.FindStringSubmatch(line); m != nil {
			cur = &model.Part{Title: strings.TrimSpace(m[1])}
			book.Parts = append(book.Parts, cur)
			pendingDescFor = cur
			continue
		}

		// Blockquote juste sous un titre de partie → description de la partie.
		if cur != nil && pendingDescFor == cur && strings.HasPrefix(trimmed, ">") {
			desc := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			if cur.Desc == "" {
				cur.Desc = desc
			} else {
				cur.Desc += " " + desc
			}
			continue
		}

		if m := reLink.FindStringSubmatch(line); m != nil {
			if cur == nil {
				continue // lien hors partie : ignoré
			}
			pendingDescFor = nil // une entrée close la zone de description
			linkTitle := strings.TrimSpace(m[1])
			href := strings.TrimSpace(m[2])

			if isRenderable(href) {
				page := &model.Page{
					SrcPath:   href,
					OutPath:   mdToHTML(href),
					Title:     linkTitle,
					PartTitle: cur.Title,
				}
				cur.Pages = append(cur.Pages, page)
				book.Pages = append(book.Pages, page)
			} else {
				cur.Links = append(cur.Links, &model.Link{Title: linkTitle, Href: href})
			}
			continue
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	linkPrevNext(book)
	return book, nil
}

// ResolveDirLinks transforme les liens vers un dossier (ex: « projets/1-cli/ »)
// en pages rendues lorsqu'un README.md y est présent : le README devient une
// page du livre, sinon le lien reste un simple renvoi vers le dépôt. À appeler
// après Parse, avec la racine du livre sur le disque. Ré-établit Prev/Next.
func ResolveDirLinks(book *model.Book, srcDir string) {
	for _, part := range book.Parts {
		var keptLinks []*model.Link
		for _, l := range part.Links {
			href := l.Href
			if !strings.HasSuffix(href, "/") {
				keptLinks = append(keptLinks, l)
				continue
			}
			readme := path.Join(href, "README.md")
			if _, err := os.Stat(filepath.Join(srcDir, filepath.FromSlash(readme))); err != nil {
				keptLinks = append(keptLinks, l) // pas de README : reste un lien
				continue
			}
			page := &model.Page{
				SrcPath:   readme,
				OutPath:   strings.TrimSuffix(readme, ".md") + ".html",
				Title:     l.Title,
				PartTitle: part.Title,
			}
			part.Pages = append(part.Pages, page)
			book.Pages = append(book.Pages, page)

			// Publie aussi les Markdown compagnons du projet (ex. RAPPORT.md)
			// pour que les liens du README vers ces fichiers résolvent dans le
			// site. Ils sont rendus et chaînés (préc./suiv.) sans figurer comme
			// entrée distincte de la navigation.
			for _, comp := range companionMarkdown(srcDir, href) {
				book.Pages = append(book.Pages, &model.Page{
					SrcPath:   comp,
					OutPath:   strings.TrimSuffix(comp, ".md") + ".html",
					Title:     path.Base(comp), // titre provisoire ; le H1 fait foi
					PartTitle: part.Title,
				})
			}
		}
		part.Links = keptLinks
	}
	linkPrevNext(book)
}

// companionMarkdown liste les fichiers Markdown d'un dossier de projet, hormis
// le README.md déjà publié, dans l'ordre alphabétique. Renvoie nil si le dossier
// est illisible.
func companionMarkdown(srcDir, dirHref string) []string {
	entries, err := os.ReadDir(filepath.Join(srcDir, filepath.FromSlash(dirHref)))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == "README.md" || !strings.HasSuffix(name, ".md") {
			continue
		}
		out = append(out, path.Join(dirHref, name))
	}
	sort.Strings(out)
	return out
}

// isRenderable renvoie vrai si le lien pointe vers un fichier Markdown à rendre
// (et non un dossier de projet ou un lien externe).
func isRenderable(href string) bool {
	if strings.Contains(href, "://") {
		return false
	}
	// On ignore l'ancre éventuelle pour le test d'extension.
	if i := strings.IndexByte(href, '#'); i >= 0 {
		href = href[:i]
	}
	return strings.HasSuffix(href, ".md")
}

// mdToHTML transforme un chemin .md en .html en préservant une ancre éventuelle.
func mdToHTML(href string) string {
	anchor := ""
	if i := strings.IndexByte(href, '#'); i >= 0 {
		anchor = href[i:]
		href = href[:i]
	}
	return strings.TrimSuffix(href, ".md") + ".html" + anchor
}

// linkPrevNext chaîne les pages dans l'ordre du SOMMAIRE (préc./suiv.).
func linkPrevNext(book *model.Book) {
	for i, p := range book.Pages {
		if i > 0 {
			p.Prev = book.Pages[i-1]
		}
		if i < len(book.Pages)-1 {
			p.Next = book.Pages[i+1]
		}
	}
}
