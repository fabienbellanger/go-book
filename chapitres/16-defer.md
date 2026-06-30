# 16 — `defer`

> **Objectif** — Maîtriser le nettoyage déterministe avec `defer` : ordre **LIFO**, **moment
> d'évaluation** des arguments, interaction avec les **retours nommés**, le piège du `defer` en
> boucle, et le coût réel (_open-coded defers_).
>
> **Prérequis** — [Ch. 5 — Fonctions](05-fonctions.md) (retours nommés), [Ch. 15 — Closures](15-closures.md)

---

## Introduction

`defer` **diffère** l'exécution d'un appel jusqu'au **retour** de la fonction englobante — quel que
soit le chemin de sortie (retour normal, retour anticipé, ou `panic`). C'est l'outil de Go pour le
**nettoyage déterministe** : fermer un fichier, libérer un verrou, restaurer un état, juste à côté
du code qui l'a acquis.

```go
f, err := os.Open(name)
if err != nil {
	return err
}
defer f.Close() // garanti, qu'on sorte par erreur ou normalement
// ... utiliser f ...
```

Le mécanisme rappelle le `try`/`finally` de Java ou Python, ou le RAII de C++ — en plus léger :
pas de bloc dédié, `defer` est une instruction ordinaire, posée à l'endroit même où la ressource est
acquise. Différence importante avec un bloc `finally` : un `defer` est rattaché à la **fonction
englobante tout entière**, jamais à un bloc (`if`, `for`...) qui le contient. Il ne se déclenche
qu'au retour de **cette fonction**, pas à la sortie d'une boucle ou d'un `if` — point central du
piège vu plus bas.

L'exemple est dans [`code/ch16-defer/`](../code/ch16-defer/).

---

## Ordre LIFO

Plusieurs `defer` dans une fonction s'exécutent en ordre **inverse** d'enregistrement (_Last In,
First Out_) : le dernier différé part en premier. C'est logique pour défaire des acquisitions
imbriquées (on libère dans l'ordre inverse où l'on a pris) : si une ressource B dépend d'une
ressource A acquise avant elle (un verrou qui protège un fichier déjà ouvert, par exemple), B doit
être libérée avant A. C'est exactement l'ordre que produit le LIFO des `defer`, sans que vous ayez à
y penser — il suffit d'empiler les `defer` dans l'ordre naturel d'acquisition.

```go
// code/ch16-defer/defer.go
func lifoOrder() (out []int) {
	for i := range 3 {
		defer func() { out = append(out, i) }() // enregistre 0, 1, 2
	}
	return // exécute 2, 1, 0
}
// lifoOrder() == [2 1 0]
```

```
  func f() {
      defer A()        (1) empile A
      defer B()        (2) empile B
      defer C()        (3) empile C
      ... corps ...
  }                    au retour : dépile en LIFO

      +-------+  <- exécuté en 1er
      |  C()  |
      +-------+
      |  B()  |
      +-------+
      |  A()  |  <- exécuté en dernier
      +-------+
```

## Moment d'évaluation des arguments

Point **crucial** : les **arguments** d'un `defer` sont évalués **à l'enregistrement** (quand la
ligne `defer` est atteinte), pas à l'exécution. En revanche, une **closure** différée lit les
variables **à l'exécution** (capture par référence, [Ch. 15](15-closures.md)). C'est le contraste à
bien avoir en tête :

```go
func evalContrast() (snapshot, live int) {
	x := 1
	defer func(v int) { snapshot = v }(x) // v = 1 : ARGUMENT figé maintenant
	defer func() { live = x }()           // lit x au RETOUR
	x = 99
	return
}
// snapshot == 1 (valeur figée), live == 99 (valeur finale)
```

Ce comportement n'a rien d'arbitraire : un `defer` reste, au fond, un **appel de fonction normal**,
simplement **exécuté plus tard**. Comme pour tout appel `f(x)`, les arguments sont calculés au
moment où l'appel est **préparé** — ici, à la ligne `defer`. Seule l'**invocation** elle-même est
repoussée au retour :

```
  x := 1
   |
   +-- defer func(v int){snapshot=v}(x)   <- x est LU ICI (argument) : v fige à 1
   +-- defer func()     {live=x}()        <- aucune lecture ici, juste l'enregistrement
   |
  x = 99                                     (v=1 ne voit plus ce changement, mais la
   |                                          closure le verra)
  return                                  -> exécute les defers, en LIFO :
                                              func(){live=x}()      -> live = 99 (lu maintenant)
                                              func(v int){...}(v=1) -> snapshot = 1 (figé avant)
```

> 💡 Pour photographier une valeur au moment du `defer`, **passez-la en argument**. Pour observer sa
> valeur finale, **capturez-la** dans la closure.

> ⚠️ La règle s'applique aussi au **récepteur** d'une méthode différée. `defer obj.Method()` évalue
> `obj` immédiatement, comme n'importe quel argument — mais ce qui est figé dépend du type de
> récepteur :
>
> ```go
> type counter struct{ n int }
>
> func (c counter) showValue()  { fmt.Println(c.n) } // récepteur valeur : copié au defer
> func (c *counter) showByRef() { fmt.Println(c.n) }  // récepteur pointeur : lu à l'exécution
>
> c := counter{n: 1}
> defer c.showValue()  // copie c MAINTENANT : affichera 1
> defer c.showByRef()  // capture &c MAINTENANT, mais lit *c plus tard : affichera 2
> c.n = 2
> ```

## Interaction avec les retours nommés

Un `defer` s'exécute **après** l'évaluation de l'expression `return` mais **avant** que la fonction
ne rende vraiment la main. S'il modifie une **variable de retour nommée**, le changement est
**visible par l'appelant**. C'est exactement le mécanisme du pattern `recover` → erreur
([Ch. 17](17-panic-recover.md)).

```go
func doubleViaDefer() (result int) {
	defer func() { result *= 2 }()
	result = 21
	return result // pose result = 21, puis le defer le double -> 42
}
```

```
  return result   ->   result = 21   ->   defers s'exécutent (result *= 2)   ->   rend 42
                       (affecte le         (peuvent lire/écrire result)
                        retour nommé)
```

> ⚠️ Cela ne marche **qu'avec un retour nommé**. Avec un `return` non nommé, la valeur est **copiée**
> avant les defers ; les modifier n'a plus d'effet sur ce qui est renvoyé (voir le piège suivant).

## Le piège du `defer` en boucle

`defer` se déclenche au retour de la **fonction**, **pas** à la fin de l'itération. Empiler des
`defer` dans une boucle **repousse** toutes les libérations à la toute fin — les ressources
s'accumulent (descripteurs de fichiers, verrous...). Sur une boucle qui parcourt des milliers de
fichiers, c'est l'erreur classique `too many open files` (limite `ulimit -n` du système
d'exploitation) : chaque `os.Open` réussit, mais aucun `Close` ne s'exécute avant la fin de la
fonction qui les a tous accumulés.

```go
// PIÈGE : tous les Close() à la fin, en LIFO. Les ressources restent ouvertes.
func processDeferInLoop(names []string) (log []string) {
	for _, n := range names {
		r := acquire(n, &log)
		defer r.Close() // s'empile : rien n'est fermé avant le retour
		r.use()
	}
	return
}
// [a b] -> [open:a use:a open:b use:b close:b close:a]
```

La correction : **une portée par itération** via une closure (ou une fonction nommée), pour que le
`defer` se déclenche à **chaque tour** :

```go
// BON : Close() à chaque itération, juste après use.
func processScoped(names []string) []string {
	var log []string
	for _, n := range names {
		func() {
			r := acquire(n, &log)
			defer r.Close()
			r.use()
		}()
	}
	return log
}
// [a b] -> [open:a use:a close:a open:b use:b close:b]
```

> 💡 Le `log` est un retour **nommé** dans `processDeferInLoop` exprès : sans cela, les `Close()`
> différés s'exécuteraient **après** la copie de la valeur de retour et seraient **invisibles** — le
> piège des retours non nommés, vu juste au-dessus.

## Patterns courants

### Trace d'entrée/sortie

Une fonction qui **journalise l'entrée maintenant** et **renvoie** la closure de sortie à différer :

```go
func trace(name string, log *[]string) func() {
	*log = append(*log, "enter:"+name)
	return func() { *log = append(*log, "exit:"+name) }
}

defer trace("work", &log)() // noter le () final : on appelle trace, on diffère son résultat
```

Ce pattern exploite directement la règle vue plus haut : `trace("work", &log)` est **appelée
immédiatement** (elle journalise `"enter:work"` tout de suite et renvoie une closure), seul le
**résultat de cet appel** — la closure renvoyée — est différé par le `defer` qui l'entoure.

### Verrou

L'idiome le plus répandu — _lock_ suivi immédiatement de `defer unlock` :

```go
func withLock(mu *sync.Mutex, fn func()) {
	mu.Lock()
	defer mu.Unlock() // libéré même si fn panique
	fn()
}
```

Placer le `defer mu.Unlock()` **juste après** `mu.Lock()` — plutôt qu'à la fin de la fonction — est
la convention : la libération reste **visuellement collée** à l'acquisition, donc impossible à
oublier en ajoutant un retour anticipé plus loin dans le corps de la fonction.

---

## 🆕 Go 1.2x

- `defer` est stable depuis Go 1 (aucun changement de sémantique). L'évolution majeure est interne :
  les **_open-coded defers_** (depuis Go 1.14) rendent les `defer` à position fixe **quasi gratuits**
  (voir ⚡ Performance).
- 🔁 Le format de panique re-déclenchée a changé en 1.25 — c'est `recover`/`panic` qui sont concernés
  ([Ch. 17](17-panic-recover.md)).

## ⚠️ Pièges

- **`defer` en boucle** : libérations repoussées à la fin de la fonction. Encapsulez le corps dans
  une closure (portée par itération).
- **Argument vs capture** : l'argument est figé à l'enregistrement, la closure lit la valeur finale.
  Confondre les deux donne des bugs subtils.
- **Récepteur de méthode** : `defer obj.Method()` fige `obj` à l'enregistrement si `Method` a un
  récepteur **valeur** (copie), mais lit l'état **au retour** si le récepteur est un **pointeur**
  (la closure implicite ne fait que déréférencer plus tard).
- **Modifier un retour non nommé** depuis un `defer` n'a **aucun effet** : la valeur est déjà copiée.
- **Ignorer l'erreur de `Close()`** : sur un fichier en **écriture**, `defer f.Close()` peut masquer
  une erreur d'écriture finale. Pour ces cas, fermez explicitement et vérifiez l'erreur
  ([Ch. 10](10-erreurs.md)), ou affectez-la à un retour nommé dans le `defer`.
- **`defer` dans une fonction longue** : la ressource vit jusqu'au bout. Si elle peut être libérée
  plus tôt, faites-le.

## ⚡ Performance

Avant Go 1.13, chaque `defer` allouait un enregistrement sur le **tas** (_defer record_) : coût
notable même hors boucle. Go 1.13 a ajouté une allocation sur la **pile** pour les cas simples ;
Go 1.14 va plus loin avec les _**open-coded defers**_ : le compilateur **« déplie »** les appels
différés directement à chaque point de sortie de la fonction (un peu comme s'il avait écrit
l'appel à la main avant chaque `return`) — plus aucun enregistrement à l'exécution, le coût devient
**celui d'un appel direct**.

Cette optimisation a une condition précise : la fonction doit compter **8 `defer` au plus, comptés
textuellement**, et **aucun d'entre eux ne doit être à l'intérieur d'une boucle**. C'est une
condition **syntaxique**, pas dynamique : un seul `defer` écrit dans un `for` désactive l'open-coding
pour cette fonction, **même si la boucle ne tourne qu'une fois** — c'est exactement le cas de
`deferLoop` ci-dessous, qui retombe donc sur le mécanisme runtime malgré ses 8 appels. Mesuré
(go1.26.4, `b.Loop`) :

```
   BenchmarkWithDefer       3.24 ns/op   0 allocs   (open-coded : ~ appel direct)
   BenchmarkWithoutDefer    3.24 ns/op   0 allocs
   BenchmarkDeferInLoop   132.9  ns/op   0 allocs   (8 defers : ~16,6 ns / defer)
```

- Un `defer` **à position fixe** est gratuit : ne l'évitez **jamais** pour « gagner » des
  nanosecondes.
- Un `defer` **en boucle** retombe sur le mécanisme runtime (chaîne de _defer records_) : ~5× plus
  cher **par defer**, et la ressource fuit en attendant. Double raison de l'éviter.
- 🔁 Mécanique des defers et allocations au [Ch. 26](26-allocation-escape.md).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch16-defer
go test ./ch16-defer/...
go test -bench=. -benchmem ./ch16-defer/...
```

À essayer :

1. Retirez le retour nommé de `processDeferInLoop` : les `Close()` disparaissent du résultat.
   Pourquoi ?
2. Transformez `defer func(v int){...}(x)` en `defer func(){... x ...}()` et observez le changement
   de valeur.
3. Mesurez `BenchmarkDeferInLoop` avec 2, 8 puis 64 itérations : le coût croît linéairement.

---

## 📌 À retenir

- `defer` s'exécute au **retour** de la fonction (y compris sur `panic`), en ordre **LIFO**.
- Les **arguments** d'un `defer` sont évalués **à l'enregistrement** ; une **closure** différée lit
  les variables **à l'exécution**.
- Un `defer` peut **modifier un retour nommé** (base du `recover` → erreur) — mais **pas** un retour
  non nommé.
- **Jamais de `defer` en boucle** pour une ressource : encapsulez le corps dans une closure.
- Les **open-coded defers** rendent un `defer` à position fixe **gratuit** ; ne l'évitez pas « pour
  la perf ».

## 🔁 Pour aller plus loin

- [Ch. 17 — `panic` & `recover`](17-panic-recover.md) : `recover` se place **toujours** dans un
  `defer` ; les defers s'exécutent pendant le déroulement de pile.
- [Ch. 15 — Closures](15-closures.md) : capture par référence, à opposer aux arguments de `defer`.
- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : open-coded defers, _defer records_.
- [Ch. 21 — Synchronisation](21-synchronisation.md) : `defer mu.Unlock()`, l'idiome du verrou.
