# Annexe F — Idiomes & style

> **Objectif** — Condenser l'esprit d'« Effective Go » : les conventions qui font
> qu'un code Go se lit comme du Go, et les pièges qui guettent ceux qui arrivent
> d'un autre langage. À relire avant une revue de code.

---

## Formatage & nommage

Le formatage n'est **pas une opinion** : `gofmt` (intégré à `go fmt`) tranche
tout — indentation par tabulation, alignement, espaces. On ne discute pas, on
lance l'outil. `gofmt -l .` en CI échoue si un fichier n'est pas formaté.

Conventions de nommage :

- **`MixedCaps`**, jamais `snake_case` ni `SCREAMING_CASE` pour les identifiants.
- **La casse contrôle la visibilité** : un identifiant qui commence par une
  **majuscule** est exporté hors du paquet ; minuscule = privé. Pas de mot-clé
  `public`/`private`.
- **Pas de getters préfixés `Get`** : on écrit `u.Name()`, pas `u.GetName()`. Le
  setter, lui, garde le verbe : `u.SetName(...)`.
- **Interfaces à une méthode en `-er`** : `Reader`, `Writer`, `Stringer`,
  `Formatter`.
- **Noms de paquets courts, en minuscules, sans souligné ni mixedCaps** : `http`,
  `strconv`, `io`. Le nom du paquet préfixe ses symboles à l'usage — évitez donc
  la redondance (`http.Server`, pas `http.HTTPServer`).
- **Acronymes en casse uniforme** : `URL`, `ID`, `HTTP` → `userID`, `parseURL`,
  `ServeHTTP` (pas `userId` ni `parseUrl`).

```go
package user

// User est exporté (Majuscule). Son champ id ne l'est pas (minuscule).
type User struct {
	id   int
	Name string
}

// ID est l'accesseur idiomatique : pas de préfixe « Get ».
func (u User) ID() int { return u.id }
```

> 💡 `go vet` et `staticcheck` repèrent beaucoup d'écarts de style ; intégrez-les
> tôt plutôt que de débattre en revue.

---

## Erreurs

🔁 Voir Ch. 10. Les règles d'or :

- **L'erreur est une valeur** renvoyée **en dernière position**, gérée
  explicitement — pas d'exceptions.
- **Envelopper avec `%w`** pour conserver la chaîne de causes, puis interroger
  avec `errors.Is` (valeur sentinelle) et `errors.As` (type concret).
- **Erreurs sentinelles** : `var ErrNotFound = errors.New("...")`, comparées via
  `errors.Is`, jamais avec `==` à travers des couches d'enveloppement.
- **Ne pas paniquer dans une bibliothèque** : `panic` est pour les invariants
  rompus (bug du programmeur), pas pour un fichier absent ou une entrée invalide.

```go
// On enrichit le contexte sans perdre la cause d'origine.
data, err := os.ReadFile(path)
if err != nil {
	return fmt.Errorf("lecture de la config %s : %w", path, err)
}

// Côté appelant, on teste la cause, pas le message.
if errors.Is(err, os.ErrNotExist) {
	// fichier manquant : créer une config par défaut
}
```

> ⚠️ Ne décorez pas une erreur deux fois sur le même chemin (« lecture : lecture :
> … ») : enveloppez une fois, à la frontière utile.

---

## Interfaces

- **« Accepter des interfaces, renvoyer des structs »** : une fonction prend le
  comportement minimal dont elle a besoin (souple à l'appel) et renvoie un type
  concret (riche à l'usage).
- **Petites interfaces** : une ou deux méthodes. On compose `io.Reader` +
  `io.Writer` en `io.ReadWriter`, on ne déclare pas une interface géante par
  avance.
- **Définir l'interface côté consommateur**, pas côté producteur : c'est celui
  qui appelle qui sait de quel sous-ensemble il dépend.
- **La valeur zéro doit être utile** : `bytes.Buffer`, `sync.Mutex`,
  `sync.WaitGroup` s'emploient sans constructeur. Visez la même chose pour vos
  types.

```go
// On dépend du strict nécessaire : n'importe quelle source de lecture convient.
func countLines(r io.Reader) (int, error) { /* ... */ }

// Et on renvoie un type concret, pas une interface fourre-tout.
func NewBuffer() *bytes.Buffer { return &bytes.Buffer{} }
```

---

## Slices & maps

🔁 Voir Ch. 6 et 30 (slices), Ch. 7 et 32 (maps). Les pièges récurrents :

- **`append` peut réallouer _ou_ aliaser** : si la capacité suffit, il écrit dans
  le tableau sous-jacent partagé ; sinon il copie ailleurs. Ne supposez jamais
  qu'un `append` est isolé d'une autre slice qui partage le tableau.
- **`nil` vs vide** : une slice/map `nil` se lit comme vide (`len == 0`, `range`
  ne fait rien), mais **écrire dans une map `nil` panique**. Initialisez avec
  `make` avant d'insérer.
- **Passage par valeur** : structs et arrays sont **copiés** à l'appel ; slices,
  maps et channels copient un petit en-tête qui **partage** les données sous-jacentes.

```go
var m map[string]int
m["x"] = 1            // ⚠️ panique : assignment to entry in nil map
m = make(map[string]int)
m["x"] = 1            // OK

// append : la slice résultat peut ou non partager le tableau d'origine.
a := []int{1, 2, 3}
b := append(a[:1], 99) // écrase a[1] si la capacité le permet — surprise garantie
```

> 💡 Pour copier réellement une slice, utilisez `slices.Clone` (ou `copy` dans une
> slice neuve), pas un simple `=`.

---

## Concurrence

🔁 Voir Ch. 19 à 23. Les principes qui évitent 90 % des bugs :

- **Qui possède le channel le ferme** — et c'est **l'émetteur**, jamais le
  récepteur. Fermer côté lecture, ou fermer deux fois, panique.
- **`context.Context` en premier paramètre**, nommé `ctx`, jamais stocké dans une
  struct : il se propage le long de la pile d'appels et porte l'annulation.
- **Ne lancez pas une goroutine sans savoir comment elle s'arrête** : toute
  goroutine doit avoir une condition de sortie (channel fermé, `ctx.Done()`,
  travail épuisé), sinon c'est une fuite.
- **Partager la mémoire en communiquant** : préférez passer une valeur par un
  channel plutôt que la protéger par un mutex, quand le design s'y prête.

```go
// L'émetteur ferme ; le récepteur boucle jusqu'à fermeture.
func produce(ctx context.Context, out chan<- int) {
	defer close(out)              // un seul fermeur : le producteur
	for i := 0; ; i++ {
		select {
		case out <- i:
		case <-ctx.Done():        // sortie propre : pas de fuite
			return
		}
	}
}
```

> ⚠️ Lancez `go test -race` régulièrement : beaucoup de bugs de concurrence sont
> invisibles tant qu'ils ne corrompent pas les données « pour de vrai ».

---

## Tests

🔁 Voir Ch. 13 et 36.

- **Tests pilotés par table** : un slice de cas `{nom, entrée, attendu}`, une
  boucle avec `t.Run(nom, ...)` pour des sous-tests nommés et isolés.
- **`t.Helper()`** dans les fonctions d'assertion : les erreurs pointent la ligne
  de l'appelant, pas celle du helper.
- **`t.Cleanup(fn)`** pour libérer les ressources (serveur, fichier temporaire)
  en fin de test, dans l'ordre inverse — plus robuste qu'un `defer` éparpillé.
- **Exemples testables** : une fonction `ExampleXxx` avec un commentaire
  `// Output:` sert à la fois de documentation (godoc) et de test.

```go
func TestAdd(t *testing.T) {
	cases := []struct {
		name     string
		a, b, want int
	}{
		{"positifs", 2, 3, 5},
		{"avec zéro", 0, 7, 7},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Add(c.a, c.b); got != c.want {
				t.Errorf("Add(%d,%d) = %d, voulu %d", c.a, c.b, got, c.want)
			}
		})
	}
}
```

---

## Composition plutôt qu'héritage

Go n'a **ni classes ni héritage**. On réutilise par **composition** :

- **Incorporation de struct** (`embedding`) : un type contient un autre type
  anonymement et **promeut** ses champs et méthodes.
- **Incorporation d'interface** pour composer des contrats (`io.ReadWriter`).
- Le polymorphisme passe par les **interfaces**, satisfaites implicitement : un
  type qui a les bonnes méthodes implémente l'interface, sans le déclarer.

```go
// Logger est « incorporé » : Service hérite de ses méthodes par promotion,
// sans relation de classe.
type Logger struct{ prefix string }

func (l Logger) Log(msg string) { fmt.Println(l.prefix, msg) }

type Service struct {
	Logger          // incorporation : s.Log(...) est disponible directement
	name string
}
```

> 💡 L'incorporation n'est pas de l'héritage : il n'y a pas de « super », pas de
> redéfinition virtuelle. La méthode promue opère sur le champ incorporé.

---

## ⚠️ Pièges classiques du débutant

- **Variable de boucle capturée** — 🆕 **corrigé en Go 1.22** : chaque itération a
  désormais sa propre variable, donc capturer `i`/`v` dans une closure ou une
  goroutine fait ce qu'on attend. En Go ≤ 1.21, il fallait `v := v`. 🔁 Voir Ch. 15.

  ```go
  for _, v := range items {
      go func() { fmt.Println(v) }() // ✅ OK depuis 1.22 ; bug avant
  }
  ```

- **Shadowing avec `:=`** — redéclarer accidentellement une variable dans un bloc
  interne masque l'externe ; l'`err` que vous croyez tester reste `nil`.

  ```go
  x, err := f()
  if cond {
      y, err := g() // ⚠️ NOUVEAU err local : celui du dehors n'est pas mis à jour
      _ = y
  }
  _ = err           // toujours celui de f()
  ```

- **Comparer une erreur avec `==`** au lieu d'`errors.Is` : casse dès qu'une couche
  a enveloppé l'erreur avec `%w`.

- **Oublier de vérifier `err`** — `go vet`/`errcheck` aident, mais l'habitude est
  de traiter l'erreur **immédiatement**, sur la ligne suivante.

- **Fuite de goroutine** — une goroutine bloquée à jamais sur un channel sans
  lecteur, ou sans écoute de `ctx.Done()`. Elle ne sera jamais ramassée.

  ```go
  ch := make(chan int)   // non bufférisé, personne ne lira
  go func() { ch <- 1 }() // ⚠️ bloque pour toujours → fuite
  ```

- **`range` copie l'élément** — `for _, v := range gros` copie chaque `v`. Pour
  muter en place ou éviter la copie d'un gros struct, indexez : `for i := range s { s[i].X = ... }`.

- **Conversion `[]byte` ↔ `string` qui alloue** — chaque `string(b)` / `[]byte(s)`
  recopie les octets. Sur un chemin chaud, évitez-les (cas spécial sans allocation :
  `m[string(b)]` **en lecture seule** d'une map). 🔁 Voir Ch. 31 et Projet 7.

- **`nil` d'interface ≠ `nil` de pointeur** (le « typed nil ») — une interface qui
  contient un pointeur nil n'est **pas** une interface nil : elle a un type. D'où
  des `if err != nil` qui passent alors que la valeur est « vide ».

  ```go
  func do() error {
      var p *MyError = nil
      return p          // ⚠️ l'interface error n'est PAS nil (elle a le type *MyError)
  }
  if do() != nil {       // ... donc ceci est vrai : bug courant
      // ...
  }
  ```

  Règle : renvoyez `nil` littéral en cas de succès, pas une variable typée nil.

---

## 📌 À retenir

- `gofmt` n'est pas négociable ; la casse de la première lettre décide de
  l'export ; on nomme court et en `MixedCaps`.
- L'erreur est une valeur en dernière position : envelopper avec `%w`, tester avec
  `errors.Is`/`As`, ne pas paniquer dans une bibliothèque.
- Accepter des interfaces (petites, côté consommateur), renvoyer des structs ; la
  valeur zéro doit être utile.
- En concurrence : l'émetteur ferme le channel, `ctx` en premier paramètre, toute
  goroutine a une porte de sortie.
- Les pièges récurrents (shadowing, typed nil, map nil, fuite de goroutine) se
  détectent vite avec `go vet` + `-race` : faites-en un réflexe.
