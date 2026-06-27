# Ch. 17 — `panic` & `recover`

> **Objectif** — Savoir **quand** paniquer (et quand ne pas le faire), rattraper une panique avec
> `recover` dans un `defer`, re-déclencher une panique inattendue, et poser une **frontière de
> récupération** (serveur HTTP). Comprendre pourquoi une panique de goroutine est **fatale**.
>
> **Prérequis** — [Ch. 10 — Erreurs](10-erreurs.md), [Ch. 16 — `defer`](16-defer.md)

---

## Introduction

En Go, le chemin normal de gestion des problèmes, ce sont les **erreurs** ([Ch. 10](10-erreurs.md)) :
des valeurs qu'on renvoie et qu'on inspecte. `panic` est l'**exception** au sens propre : un arrêt
brutal du flot normal, réservé aux **bugs** et aux **invariants violés** — pas au contrôle de flux
ordinaire.

Une panique **déroule la pile** (exécute les `defer` de chaque fonction en remontant) et, si rien ne
l'arrête, **termine le programme**. `recover`, appelé **dans un `defer`**, peut stopper ce
déroulement. L'exemple est dans [`code/ch17-panic-recover/`](../code/ch17-panic-recover/).

---

## Quand paniquer (et quand ne pas)

| Situation                                             | Outil                    |
| ----------------------------------------------------- | ------------------------ |
| Entrée invalide, fichier absent, réseau coupé...      | **erreur**               |
| Bug du programmeur (invariant cassé, état impossible) | **panic**                |
| Échec d'initialisation **fatal** (config de package)  | **panic** (`Must…`)      |
| Cas « ça ne peut pas arriver » dans un `switch`       | **panic** dans `default` |

> 📌 Règle : si l'appelant peut **raisonnablement réagir**, renvoyez une **erreur**. Paniquez
> seulement quand continuer n'a **aucun sens** (le programme est dans un état corrompu).

La bibliothèque standard suit cette ligne : `regexp.Compile` renvoie une erreur, `regexp.MustCompile`
**panique** — cette dernière est faite pour les **variables de package**, où un motif invalide est
un bug à corriger, pas une condition d'exécution.

```go
// Pattern "Must" : paniquer plutôt que renvoyer une erreur (init/invariants).
func mustPositive(n int) int {
	if n <= 0 {
		panic(fmt.Sprintf("valeur attendue > 0, reçu %d", n))
	}
	return n
}
```

## `recover` : toujours dans un `defer`

`recover` n'a d'effet **que** s'il est appelé **directement dans une fonction différée**, pendant
qu'une panique est en cours. Il renvoie la valeur passée à `panic` (ou `nil` s'il n'y a pas de
panique). L'idiome de base convertit une panique en **erreur** via un **retour nommé**
([Ch. 16](16-defer.md)) :

```go
// code/ch17-panic-recover/recover.go
func safeCall(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panique rattrapée : %v", r)
		}
	}()
	fn()
	return nil
}

safeCall(func() { _ = divide(10, 0) })
// -> "panique rattrapée : runtime error: integer divide by zero"
```

Même les **paniques du runtime** (déréférencement de pointeur nil, index hors bornes, division
entière par zéro) sont des paniques ordinaires : `recover` les rattrape aussi.

### Déroulement de pile

```
  main
   |__ handle()      defer recover()   <- rattrape ici
        |__ process()   defer cleanup()
             |__ parse()   defer close()
                  panic("x")   <- déclenche le déroulement

  On REMONTE la pile en exécutant les defers de chaque niveau :

     parse   : close()    exécuté
     process : cleanup()  exécuté
     handle  : recover()  -> STOPPE le déroulement ; handle() retourne normalement
```

Si aucun `recover` n'intercepte, le déroulement atteint le sommet et le programme **s'arrête** (code
de sortie 2) en imprimant le message et la pile.

## Re-déclencher une panique inattendue

Un `recover` trop large masque les vrais bugs. Le bon réflexe : ne rattraper que ce qu'on
**reconnaît**, et **re-paniquer** le reste.

```go
type validationPanic struct{ field string }

func validate(age, score int) (err error) {
	defer func() {
		switch r := recover().(type) {
		case nil:
			// aucune panique
		case validationPanic:
			err = fmt.Errorf("champ %q invalide", r.field) // attendu -> erreur
		default:
			panic(r) // INATTENDU -> on laisse remonter (vrai bug)
		}
	}()
	checkPositive("age", age)
	checkPositive("score", score)
	return nil
}
```

## Frontière de récupération (serveur)

Le cas d'usage le plus légitime de `recover` : une **frontière** qui empêche la panique d'**une**
tâche de tuer **tout** le processus. Dans un serveur, on enveloppe chaque requête ; une panique
devient un **500**, et le serveur **continue** :

```go
func recoverMiddleware(next Handler) Handler {
	return func(req Request) (resp Response) {
		defer func() {
			if r := recover(); r != nil {
				// En vrai : logguer r + debug.Stack().
				resp = Response{status: 500, body: fmt.Sprintf("internal error: %v", r)}
			}
		}()
		return next(req)
	}
}
// GET /home -> 200 ;  GET /boom (panique) -> 500 ;  le serveur tourne toujours
```

C'est exactement ce que fait le middleware `Recoverer` des frameworks HTTP (projet 2). `net/http`
lui-même rattrape déjà les paniques de handler pour ne pas tomber.

## ⚠️ Une panique de goroutine est fatale

Point **critique** : `recover` ne fonctionne que dans la goroutine qui panique. Une panique dans une
**autre** goroutine **n'est pas rattrapable** depuis l'extérieur — elle fait planter **tout le
programme**, même s'il existe un `recover` dans `main` :

```go
func main() {
	defer func() { recover() }() // N'AURA AUCUN EFFET sur la goroutine ci-dessous
	go func() {
		panic("boom") // fait planter TOUT le programme
	}()
	time.Sleep(time.Second)
}
// panic: boom  ->  exit status 2
```

> 📌 Chaque goroutine qui peut paniquer doit gérer **sa propre** frontière de recover. C'est pourquoi
> les pools de workers ([Ch. 23](23-patterns-concurrence.md)) enveloppent la fonction de tâche.

---

## 🆕 Go 1.2x

- **1.25** — le format d'une panique **re-déclenchée avec la même valeur** (`panic(recover())`) est
  désormais condensé en **`[recovered, repanicked]`** sur une seule ligne, au lieu d'afficher deux
  fois la même panique. Vérifié sur go1.26.4 :

```
$ go run .   # defer { panic(recover()) } ; panic("boom")
panic: boom [recovered, repanicked]
```

Re-paniquer avec une valeur **différente** donne toujours la chaîne `panic: … [recovered]` suivie
de la nouvelle panique.

## ⚠️ Pièges

- **`recover` hors d'un `defer`** : sans effet (renvoie `nil`). Il doit être appelé **directement**
  dans la fonction différée, pas dans une fonction qu'elle appelle.
- **Recover trop large** : avaler toutes les paniques masque les bugs. Reconnaissez et
  **re-paniquez** l'inattendu.
- **Paniquer pour le contrôle de flux** : remplacer les erreurs par des paniques rend le code
  imprévisible et lent. Réservez `panic` aux bugs.
- **Panique de goroutine** : non rattrapable de l'extérieur → crash global. Posez une frontière
  **dans** chaque goroutine.
- **`recover()` dont on ignore la valeur** : `if recover() != nil` perd l'information. Récupérez la
  valeur pour la logguer.

## ⚡ Performance

- Le chemin **sans panique** est gratuit : un `defer` avec `recover` profite des _open-coded defers_
  ([Ch. 16](16-defer.md)) — pas de surcoût tant que rien ne panique.
- **Paniquer/dérouler est coûteux** (parcours de pile, exécution des defers) : ce n'est pas un
  mécanisme à emprunter sur un chemin chaud. Une boucle qui « contrôle » par panic/recover est bien
  plus lente que des retours d'erreur.
- 🔁 Capture de la pile (`runtime/debug.Stack`) et observabilité au [Ch. 29](29-observabilite-runtime.md).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch17-panic-recover
go test ./ch17-panic-recover/...
```

À essayer :

1. Écrivez un programme `defer func(){ panic(recover()) }()` + `panic("x")` et observez
   `[recovered, repanicked]`.
2. Lancez une `panic` dans une goroutine avec un `recover` dans `main` : constatez le crash.
3. Ajoutez à `recoverMiddleware` la capture de `debug.Stack()` dans le corps de 500.

---

## 📌 À retenir

- `panic` est pour les **bugs/invariants**, pas le contrôle de flux ; si l'appelant peut réagir,
  renvoyez une **erreur**.
- `recover` ne marche que **dans un `defer`** et seulement dans la **goroutine** qui panique.
- Une panique **déroule la pile** en exécutant les `defer` ; sans `recover`, le programme s'arrête.
- Ne rattrapez que l'**attendu** ; **re-paniquez** le reste pour ne pas masquer les vrais bugs.
- Une panique de **goroutine** est **fatale** : chaque goroutine pose sa propre frontière.

## 🔁 Pour aller plus loin

- [Ch. 16 — `defer`](16-defer.md) : `recover` repose sur les defers et les retours nommés.
- [Ch. 10 — Erreurs](10-erreurs.md) : la voie normale, à préférer à `panic`.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : protéger les goroutines d'un pool.
- [Ch. 19 — Goroutines](19-goroutines.md) : pourquoi une panique de goroutine est fatale.
