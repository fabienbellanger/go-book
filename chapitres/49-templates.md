# 49 — Templates (`text/template` & `html/template`)

> **Objectif** — Générer du texte et du HTML paramétrés (rapports, e-mails, fichiers
> de configuration, code source, pages web) avec les moteurs de gabarits de la
> bibliothèque standard : le langage d'actions, les fonctions personnalisées
> (`FuncMap`), la composition de gabarits, l'échappement contextuel de
> `html/template`, et les pièges à connaître.

> **Prérequis** — [Ch. 5](05-fonctions.md) (fonctions, valeurs de fonction),
> [Ch. 8](08-structs-methodes.md) (structs, champs exportés), [Ch. 41](41-io-flux.md)
> (`io.Writer`).

---

## Introduction

Un **moteur de gabarits** sépare la **structure** d'un texte de ses **données** :
on écrit une fois la forme (`Facture — {{ .Customer }}`), on l'exécute autant de
fois qu'on a de jeux de données. C'est l'outil de choix pour produire des e-mails,
des rapports, des fichiers de configuration ou du code source (c'est ce que fait le
[Projet 6](../projets/6-codegen/)).

Go fournit **deux** packages jumeaux, d'API identique mais au comportement opposé :

```
  text/template   ->  n'échappe RIEN   (texte brut, config, code, e-mails texte)
  html/template   ->  échappe selon le CONTEXTE HTML (anti-XSS, pages web)
```

⚠️ Dans une page HTML, `text/template` est une faille XSS : du contenu utilisateur
y passe tel quel. Pour tout rendu HTML, utilisez `html/template` (section dédiée
plus bas). Le **langage** décrit ci-dessous est **identique** pour les deux packages ;
on l'introduit avec `text/template`, puis on montre ce que `html/template` ajoute :
l'échappement contextuel automatique.

---

## Le langage des actions

Tout ce qui est entre `{{` et `}}` est une **action** ; le reste est copié
littéralement. Le **point** `{{.}}` désigne la **donnée courante** passée à
`Execute`.

```
  {{ . }}                  la donnée courante (racine, ou élément d'une boucle)
  {{ .Name }}              le champ (exporté) Name du struct courant
  {{ .User.Email }}        accès en chaîne
  {{ index .Items 0 }}     indexation d'un slice/map
  {{ /* commentaire */ }}   ignoré (n'apparaît pas dans la sortie)
```

⚠️ Seuls les **champs exportés** (majuscule initiale) sont accessibles : `{{ .name }}`
sur un champ non exporté échoue à l'exécution.

### Conditions et boucles

```
  {{ if .Paid }}payée{{ else }}à régler{{ end }}
  {{ range .Lines }}- {{ .Label }}
  {{ end }}
  {{ with .Address }}{{ .City }}{{ end }}   redéfinit . sur .Address si non vide
```

Dans un `{{ range }}`, le point `.` devient **l'élément courant** de l'itération.
`{{ range $i, $v := .Items }}` expose l'index et la valeur dans des variables.

### Variables, pipelines, fonctions

Une **variable** se déclare avec `$nom := ...`. Un **pipeline** enchaîne des
appels avec `|`, la valeur de gauche devenant le **dernier** argument de droite :

```
  {{ $name := .Customer }}
  {{ .Customer | upper }}            équivaut à {{ upper .Customer }}
  {{ .Price | printf "%.2f €" }}     chaîne printf en bout de pipeline
```

Le moteur fournit des fonctions **prédéfinies** : `printf`, `print`, `len`, `index`,
`slice`, `and`, `or`, `not`, et les comparaisons `eq`, `ne`, `lt`, `le`, `gt`, `ge`.

### Contrôle des espaces

Les sauts de ligne autour des actions se retrouvent dans la sortie. `{{-` supprime
les espaces (et retours) **avant** l'action, `-}}` ceux **après** :

```
  {{ range .Lines -}}     le "-}}" évite une ligne vide après chaque itération
  - {{ .Label }}
  {{ end -}}
```

---

## Fonctions personnalisées : `FuncMap`

`Funcs` enregistre vos propres fonctions **avant** `Parse`. Chaque clé devient un
mot utilisable en appel direct ou en aval d'un pipeline.

```go
// code/ch49-templates/main.go
var funcs = template.FuncMap{
	"upper":   strings.ToUpper,
	"frMoney": frMoney,        // int64 (centimes) -> "12,50 €"
	"total":   totalCents,     // []Line -> int64
}

var invoiceTmpl = template.Must(
	template.New("invoice").Funcs(funcs).Parse(invoiceText))
```

```
Facture — {{ .Customer | upper }}
{{ range .Lines -}}
- {{ .Label }}: {{ .Cents | frMoney }}
{{ end -}}
Total: {{ total .Lines | frMoney }}
Statut: {{ if .Paid }}payée{{ else }}à régler{{ end }}
```

💡 Une fonction de `FuncMap` peut renvoyer `(T, error)` : si l'erreur est non nulle,
`Execute` **s'arrête** et la remonte. Pratique pour valider une donnée en plein rendu.

---

## Composer des gabarits

Un template peut en **appeler** un autre. `{{ define "nom" }}...{{ end }}` déclare un
sous-gabarit ; `{{ template "nom" . }}` l'invoque en lui passant une donnée (souvent
`.`, l'élément courant) :

```go
const menuText = `{{define "item"}}- {{.}}{{end}}Menu:
{{range .}}{{template "item" .}}
{{end -}}`
```

Rendu pour `["Entrée", "Plat", "Dessert"]` :

```
  Menu:
  - Entrée
  - Plat
  - Dessert
```

`{{ block "nom" . }}...{{ end }}` combine `define` + `template` : il définit un bloc
**et** l'exécute, ce qui permet de le **surcharger** ailleurs (gabarits de mise en
page avec sections remplaçables).

En pratique, on ne code pas les gabarits en dur : on les charge depuis des fichiers.

```go
tmpl := template.Must(template.ParseGlob("templates/*.tmpl"))
//go:embed templates/*.tmpl
var tfs embed.FS
tmpl := template.Must(template.ParseFS(tfs, "templates/*.tmpl"))
```

🔁 `ParseFS` + `//go:embed` embarque les gabarits dans le binaire : voir
[Ch. 46](46-embed-build-deploiement.md).

---

## `html/template` : le même langage, échappement contextuel

Pour générer du **HTML**, on change un seul mot dans l'import — `text/template`
devient `html/template` — et **tout ce qui précède reste vrai** : mêmes actions,
mêmes `FuncMap`, mêmes `{{define}}`/`{{block}}`, mêmes `ParseFS`. La seule
différence, décisive, est que `html/template` **échappe automatiquement** chaque
action **selon le contexte** où elle atterrit dans le document.

```go
// code/ch49-templates/main.go — la même valeur {{.Name}} dans deux contextes.
const pageText = `<p>Bonjour {{.Name}}</p>
<a href="/u/{{.Name}}">profil</a>`
```

Rendu pour une entrée hostile `"<script>alert(1)</script>"` :

```
  <p>Bonjour &lt;script&gt;alert(1)&lt;/script&gt;</p>
  <a href="/u/%3cscript%3ealert%281%29%3c/script%3e">profil</a>
```

La **même** valeur est encodée en **entités HTML** dans le corps, mais en
**URL** dans l'attribut `href` : `html/template` choisit l'échappeur adapté à
chaque emplacement (corps, attribut, URL, bloc `<script>`, bloc `<style>`).

| Contexte de l'action      | Échappement appliqué        |
| ------------------------- | --------------------------- |
| Corps HTML `<p>{{.}}</p>` | entités HTML (`<` → `&lt;`) |
| Valeur d'attribut         | échappement d'attribut      |
| URL (`href`, `src`)       | encodage d'URL (`%XX`)      |
| Bloc `<script>`           | littéral JavaScript         |
| Bloc `<style>`            | valeur CSS                  |

⚠️ Cet échappement est **contextuel**, pas un simple filtrage de caractères :
`html/template` analyse la **structure** du gabarit dès `Parse`. Le _pourquoi_
sécurité (anti-XSS) et le détail de cette analyse sont traités au
[Ch. 47](47-securite-supply-chain.md) ; ici, retenez surtout que l'échappement est
**automatique et gratuit** côté rédacteur de gabarits.

### Marquer un fragment comme sûr

Parfois on veut insérer du HTML **déjà** fiable (un fragment produit par le
programme, pas par l'utilisateur). Les types nommés de `html/template` disent au
moteur « cette valeur est sûre, ne l'échappe pas » :

```go
data := map[string]any{
    "Bio": template.HTML("<em>déjà nettoyé</em>"), // inséré tel quel
}
```

`template.HTML`, `template.JS`, `template.URL`, `template.CSS`,
`template.HTMLAttr`, `template.JSStr` couvrent chaque contexte.

⚠️ N'enveloppez **jamais** une entrée externe dans un de ces types : c'est
exactement rouvrir la faille XSS que `html/template` ferme. Réservez-les au
contenu que **vous** produisez (🔁 [Ch. 47](47-securite-supply-chain.md)).

> 💡 Comme l'API est identique, un template HTML se compose et se charge de la même
> façon (`Funcs`, `{{block}}`, `ParseFS` + `//go:embed`) — idéal pour un layout de
> page avec des sections remplaçables, servi depuis un `http.Handler`
> ([Ch. 45](45-net-http.md)).

---

## ⚠️ Pièges

- **Deux temps, deux erreurs.** `Parse` échoue sur une faute de **syntaxe** ;
  `Execute` échoue sur une faute de **donnée** (mauvais type, fonction en erreur).
  Ne supposez jamais qu'un gabarit qui _compile_ s'exécutera sans faute.
- **Clé manquante silencieuse.** Sur une `map`, une clé absente rend `<no value>`
  **sans erreur** par défaut — un bug qui passe inaperçu. Activez
  `Option("missingkey=error")` pour transformer ce cas en erreur d'exécution :

```go
t, _ := template.New("strict").Option("missingkey=error").Parse(`Bonjour {{.name}}`)
err := t.Execute(io.Discard, map[string]any{}) // err != nil : clé "name" absente
```

- **`Must` panique.** `template.Must(...)` transforme une erreur de `Parse` en
  **panique** : réservez-le aux gabarits **constants** compilés à l'initialisation
  (variable de paquet), jamais à un gabarit issu d'une entrée dynamique.
- **Ne parsez pas en boucle.** Comme pour `regexp.MustCompile`
  ([Ch. 42](42-encodages-serialisation.md)), compilez le gabarit **une fois** et
  réutilisez-le : re-parser à chaque rendu jette tout le travail d'analyse.

---

## ⚡ Performance

Le coût dominant est le **parsing** (analyse lexicale + construction de l'arbre),
pas l'exécution. La règle est donc simple :

- **Compiler une fois** : gabarit en **variable de paquet** (`var t = template.Must(...)`)
  ou dans `init`, jamais dans le chemin chaud.
- **Exécuter vers un `io.Writer`** : `Execute(w, data)` **streame** sa sortie. Écrire
  directement dans le `http.ResponseWriter` ou un `bufio.Writer` évite de bâtir une
  grande chaîne en mémoire ; réservez `strings.Builder`/`bytes.Buffer` aux cas où
  vous avez vraiment besoin du texte complet ([Ch. 41](41-io-flux.md)).

Un `*template.Template` compilé est **sûr en usage concurrent** pour `Execute` : un
seul gabarit de paquet sert toutes les goroutines, sans copie ni verrou.

---

## 🧪 À tester soi-même

Le code du chapitre (`code/ch49-templates/`) rend une facture (`render`), une liste
composée (`renderMenu`), illustre `missingkey=error` (`renderStrict`) et
l'échappement contextuel de `html/template` sur une entrée hostile (`renderPage`).

```bash
cd code && go test ./ch49-templates/...
```

**À essayer :**

1. Retirez le `-}}` après `{{ range .Lines` dans `invoiceText` et relancez
   `go run ./ch49-templates/` : observez les lignes vides parasites réintroduites.
2. Ajoutez une fonction `frDate` à la `FuncMap` (formatage d'une `time.Time`) et
   utilisez-la dans le gabarit via un pipeline `{{ .When | frDate }}`.
3. Passez un mauvais type à `render` (un champ `Cents` en `string`) et constatez que
   l'erreur remonte à l'**exécution**, pas à la compilation.
4. Remplacez `renderStrict` par un `Option("missingkey=zero")` : la clé absente rend
   alors la **valeur zéro** du type au lieu d'une erreur — comparez les trois modes
   (`default`, `zero`, `error`).
5. Faites rendre `pageText` par `text/template` au lieu de `html/template` : la balise
   `<script>` ressort **intacte** (faille XSS). Puis, dans `renderPage`, enveloppez le
   nom dans `template.HTML(...)` et constatez que l'échappement est désactivé pour cette
   valeur — à ne jamais faire sur une entrée externe.

---

## 📌 À retenir

- `text/template` **n'échappe rien** (texte, config, code) ; `html/template` a la
  **même API** mais **échappe selon le contexte** (corps, attribut, URL, JS, CSS) —
  automatique et anti-XSS ([Ch. 47](47-securite-supply-chain.md)). Les types
  `template.HTML`/`JS`/`URL`… marquent un fragment sûr, jamais une entrée externe.
- Le langage : `{{.}}`, champs `{{.Name}}` (exportés seulement), `{{if}}`, `{{range}}`,
  `{{with}}`, variables `{{$x := ...}}`, pipelines `{{.X | fn}}`, trim `{{- -}}`.
- **`FuncMap` avant `Parse`** ajoute vos fonctions ; une fonction peut renvoyer une
  erreur qui interrompt le rendu.
- **Composer** avec `{{define}}`/`{{template}}`/`{{block}}` ; **charger** avec
  `ParseFS` + `//go:embed`.
- **Compiler une fois** (variable de paquet), **exécuter vers un `io.Writer`** ;
  un gabarit compilé est sûr en concurrence.
- Pièges : `Parse` vs `Execute`, `missingkey`, `Must` panique, ne pas parser en boucle.

## 🔁 Pour aller plus loin

- [Ch. 47 — Sécurité](47-securite-supply-chain.md) : `html/template` et l'anti-XSS.
- [Ch. 46 — Embarquer & déployer](46-embed-build-deploiement.md) : `ParseFS` + `embed.FS`.
- [Ch. 42 — Encodages](42-encodages-serialisation.md) : le piège « compiler en boucle » (regexp).
- [Projet 6 — Générateur de code](../projets/6-codegen/) : `text/template` pour produire du Go.
- Doc : [`pkg.go.dev/text/template`](https://pkg.go.dev/text/template),
  [`pkg.go.dev/html/template`](https://pkg.go.dev/html/template).
