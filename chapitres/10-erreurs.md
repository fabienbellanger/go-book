# Ch. 10 — Gestion des erreurs

> **Objectif** — Maîtriser le modèle d'erreur de Go : les erreurs sont des **valeurs** que l'on
> **enrichit** (wrapping) et **inspecte** (`Is`/`As`), jamais des exceptions que l'on masque.
>
> **Prérequis** — [Ch. 9 — Interfaces](09-interfaces.md)

---

## Introduction

Go n'a **pas d'exceptions** pour le contrôle de flux. Une fonction qui peut échouer renvoie une
**erreur** comme **dernière valeur de retour**, et l'appelant la **traite explicitement**. Ce
choix rend les chemins d'échec visibles dans le code — verbeux, mais robuste et lisible. La
devise : _les erreurs sont des valeurs_.

L'exemple complet est dans [`code/ch10-errors/`](../code/ch10-errors/).

---

## Le type `error`

`error` est une simple **interface** de la bibliothèque standard ([Ch. 9](09-interfaces.md)) :

```go
type error interface {
	Error() string
}
```

Tout type ayant une méthode `Error() string` est une erreur. La valeur `nil` signifie « pas
d'erreur ».

## Créer une erreur

```go
err := errors.New("disque plein")                 // erreur simple
err = fmt.Errorf("port invalide : %d", 99999)     // erreur formatée
```

> ⚡ Depuis **Go 1.26**, `fmt.Errorf("…")` **sans** verbe `%w` alloue autant qu'`errors.New`
> (1 allocation), il n'y a donc plus de raison de préférer l'un à l'autre pour une erreur
> formatée simple.

## Le motif idiomatique

L'erreur est le **dernier** retour ; on la teste **immédiatement** (rappel
[Ch. 5](05-fonctions.md)) :

```go
f, err := os.Open(name)
if err != nil {
	return err // ou : return fmt.Errorf("ouverture %q: %w", name, err)
}
defer f.Close() // nettoyage déterministe (intro ; détail Ch. 16)
```

> ⚠️ **Ne jamais ignorer une erreur** silencieusement (`f, _ := os.Open(...)`). Si vous l'ignorez
> vraiment sciemment, rendez-le explicite et commentez pourquoi.

## Envelopper (`wrap`) avec `%w`

Le verbe `%w` de `fmt.Errorf` **enveloppe** une erreur : il ajoute du contexte **tout en
conservant** l'erreur d'origine, accessible via `errors.Unwrap`. On construit ainsi une **chaîne
d'erreurs** :

```go
base := errors.New("disque plein")
err := fmt.Errorf("écriture de config.txt: %w", base)
// err.Error() == "écriture de config.txt: disque plein"
errors.Unwrap(err) == base // true
```

```
   chaîne d'erreurs (du plus externe au plus interne)
   --------------------------------------------------
   fmt.Errorf("écriture: %w", ...)      (wrapError)
              |  Unwrap()
              v
   *ParseError{ Line: 3 }               (type d'erreur, Unwrap -> cause)
              |  Unwrap()
              v
   ErrEmptyKey                          (sentinelle = feuille)

   errors.Is(err, ErrEmptyKey)  parcourt la chaîne          -> true
   errors.As(err, &pe)          y trouve le *ParseError      -> pe.Line == 3
```

> 💡 `%w` vs `%v` : `%w` **garde** le lien (la chaîne reste inspectable) ; `%v` n'insère que le
> **texte** et **rompt** la chaîne. Utilisez `%w` pour propager, `%v` quand vous voulez
> délibérément masquer le type interne.

## Inspecter la chaîne : `errors.Is` et `errors.As`

- **`errors.Is(err, cible)`** — l'un des maillons est-il **égal** à la sentinelle `cible` ?

```go
if errors.Is(err, ErrEmptyKey) { /* ... */ }
```

- **`errors.As(err, &cible)`** — l'un des maillons est-il **du type** de `cible` ? Si oui, il y
  est **affecté** :

```go
var pe *ParseError
if errors.As(err, &pe) {
	fmt.Println("erreur à la ligne", pe.Line)
}
```

> ⚠️ Comparez **toujours** avec `errors.Is`, pas avec `==` : `err == ErrEmptyKey` échoue dès que
> l'erreur a été enveloppée. `errors.Is` traverse la chaîne.

## 🆕 `errors.AsType[E]` (Go 1.26)

`errors.AsType` est la variante **générique** d'`errors.As` : plus besoin de déclarer une
variable cible et de passer son adresse, le type est un **paramètre de type**
([Ch. 11](11-genericite.md)).

```go
// Avant (errors.As) :
var pe *ParseError
if errors.As(err, &pe) { use(pe) }

// 🆕 1.26 (errors.AsType) :
if pe, ok := errors.AsType[*ParseError](err); ok { use(pe) }
```

Signature : `func AsType[E error](err error) (E, bool)`. Plus typée (pas de `any`), plus directe,
et le compilateur `go fix` peut migrer `As` → `AsType` automatiquement.

## Sentinelles **vs** types d'erreur

Deux façons de signaler une condition reconnaissable :

| Approche          | Définition                          | Test        | Quand                                  |
| ----------------- | ----------------------------------- | ----------- | -------------------------------------- |
| **Sentinelle**    | `var ErrNotFound = errors.New(...)` | `errors.Is` | condition simple, sans données         |
| **Type d'erreur** | `type ParseError struct{ ... }`     | `errors.As` | besoin de **contexte** (ligne, champ…) |

Un **type d'erreur** porte des champs (numéro de ligne, nom de champ) et **enveloppe** sa cause
via une méthode `Unwrap` :

```go
type ParseError struct {
	Line int
	Err  error
}

func (e *ParseError) Error() string { return fmt.Sprintf("ligne %d: %v", e.Line, e.Err) }
func (e *ParseError) Unwrap() error { return e.Err } // insère la cause dans la chaîne
```

> 💡 On peut aussi définir `Is(target error) bool` ou `As(any) bool` sur son type pour
> personnaliser la correspondance (ex. deux erreurs « égales » selon un code).

## Agréger plusieurs erreurs : `errors.Join` (🆕 1.20)

Pour ne **pas s'arrêter à la première** erreur (validation de formulaire, parsing de plusieurs
lignes), `errors.Join` combine plusieurs erreurs en une seule. `errors.Is`/`As` traversent
**toutes** les branches.

```go
err := errors.Join(err1, err2, err3) // ignore les nil ; renvoie nil si tout est nil
// err.Error() = les messages, un par ligne
```

`fmt.Errorf` accepte d'ailleurs **plusieurs `%w`** (1.20) : `fmt.Errorf("%w / %w", a, b)`.

## `panic` n'est **pas** une erreur

`panic` sert aux **bugs** et **invariants violés** (état impossible), pas aux échecs attendus
(fichier absent, entrée invalide) — ceux-là sont des **valeurs `error`**. On détaille `panic`/
`recover` au [Ch. 17](17-panic-recover.md), et `defer` (le nettoyage) au
[Ch. 16](16-defer.md).

```
   entrée invalide / I/O / réseau   -> renvoyer une error   (cas NORMAL, attendu)
   index hors bornes / nil map en   -> panic                (BUG du programme)
   écriture / invariant cassé
```

---

## 🆕 Go 1.2x

- **1.13** — `errors.Is`, `errors.As`, `errors.Unwrap` et le verbe `%w` : naissance du wrapping
  standard.
- **1.20** — `errors.Join` et **plusieurs `%w`** dans un même `fmt.Errorf` (arbres d'erreurs,
  `Unwrap() []error`).
- **1.26** — `errors.AsType[E]` (variante générique typée) ; `fmt.Errorf` sans `%w` alloue comme
  `errors.New`.

## ⚠️ Pièges

- **Ignorer l'erreur** (`_`) → bugs silencieux. Traitez ou propagez.
- **`==` au lieu d'`errors.Is`** → faux négatif dès qu'il y a wrapping.
- **`%v` au lieu de `%w`** quand on voulait propager → la chaîne est rompue, `Is`/`As`
  échouent.
- **Exposer une sentinelle** (`var ErrFoo`) en fait une **partie de l'API publique** : les
  appelants vont l'utiliser avec `Is`. Changez-la avec prudence.
- **`errors.As` avec une mauvaise cible** (pas un pointeur vers un type implémentant `error`) →
  panique. La cible doit être `*T`.

## ⚡ Performance

- `errors.New` et `fmt.Errorf("…")` (sans `%w`) : **1 allocation** chacun (≈16 o). `fmt.Errorf`
  **avec `%w`** : **2 allocations** (≈48 o) — le surcoût du maillon de chaîne.
- **Préallouez les sentinelles** au niveau package (`var ErrX = errors.New(...)`) : une seule
  allocation pour tout le programme.
- N'enveloppez pas dans une **boucle chaude** par réflexe : ajoutez du contexte aux
  **frontières** (entrée/sortie d'une couche), pas à chaque appel.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch10-errors
go test ./ch10-errors/...
```

À essayer :

1. Remplacez `%w` par `%v` dans `parseLine` et observez `errors.Is(err, ErrEmptyKey)` passer à
   `false` (la chaîne est rompue).
2. Ajoutez une méthode `Is(target error) bool` à `*ParseError` qui considère deux erreurs égales
   si elles concernent la même ligne.
3. Migrez le `errors.As` du `main.go` vers `errors.AsType` (ou lancez `go fix ./...`).

---

## 📌 À retenir

- Une erreur est une **valeur** (`interface{ Error() string }`), renvoyée en **dernier** et
  testée tout de suite.
- `%w` **enveloppe** (chaîne inspectable) ; `errors.Is` reconnaît une **sentinelle**,
  `errors.As`/`AsType` un **type**.
- **Sentinelle** (`Is`) pour une condition simple ; **type d'erreur** (`As`) pour porter du
  contexte.
- `errors.Join` agrège plusieurs erreurs ; `errors.AsType` (🆕 1.26) remplace `As` en plus typé.
- `panic` = **bug**, pas erreur attendue.

## 🔁 Pour aller plus loin

- [Ch. 16 — `defer`](16-defer.md) : nettoyage déterministe, et erreurs depuis un `defer`.
- [Ch. 17 — `panic` & `recover`](17-panic-recover.md) : la frontière entre bug et erreur.
- [Ch. 11 — Généricité](11-genericite.md) : ce qui rend `errors.AsType[E]` possible.
- Projet 2 — API REST : erreurs structurées aux frontières HTTP.
