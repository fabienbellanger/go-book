# 20 — Channels & `select`

> **Objectif** — Faire **communiquer** des goroutines avec des canaux : envoi/réception, fermeture,
> `range`, canaux **bufferisés ou non**, **directions**, et multiplexage avec `select` (`default`,
> timeout). Reconnaître les pièges (envoi sur canal fermé, deadlock, fuite).
>
> **Prérequis** — [Ch. 19 — Goroutines](19-goroutines.md), [Ch. 4 — Flux de contrôle](04-flux-controle.md) (`range`)

---

## Introduction

La devise de la concurrence en Go : **« Ne communiquez pas en partageant la mémoire ; partagez la
mémoire en communiquant. »** Un **canal** (`chan`) est un tuyau **typé** et **sûr** par lequel une
goroutine **envoie** une valeur qu'une autre **reçoit**. Le canal **synchronise** au passage : la
réception ne peut pas voir une valeur avant qu'elle soit envoyée (une arête _happens-before_,
[Ch. 25](25-modele-memoire.md)).

Là où le [Ch. 19](19-goroutines.md) lançait des goroutines, ce chapitre les fait **dialoguer**.
L'exemple est dans [`code/ch20-channels-select/`](../code/ch20-channels-select/).

---

## Envoi, réception, fermeture

Trois opérations, l'opérateur `<-` pointant **dans le sens du flux** :

```go
ch := make(chan int) // canal de int
ch <- 42             // ENVOI : pousse 42 (l'opérateur pointe VERS le canal)
v := <-ch            // RÉCEPTION : tire une valeur (l'opérateur vient DU canal)
close(ch)            // FERMETURE : « plus aucune valeur ne sera envoyée »
```

Après `close`, toute **réception** renvoie immédiatement les valeurs restantes, puis la **valeur
nulle** du type. La forme à deux résultats distingue les deux cas :

```go
v, ok := <-ch // ok == false quand le canal est fermé ET vidé
```

> ⚠️ L'**asymétrie** entre envoi et réception sur un canal fermé n'est pas arbitraire. `close`
> est une promesse à sens unique de **l'envoyeur** : « plus rien n'arrivera ». Cette promesse doit
> rester valable indéfiniment, pour **tous** les récepteurs, y compris ceux qui regardent le canal
> bien après sa fermeture — une réception sur un canal fermé renvoie donc toujours la valeur zéro
> plutôt que de paniquer, ce qui permet à un `range` ou un `select` de continuer à l'interroger
> sans risque. Un **envoi**, lui, viole directement cette promesse : il n'existe pas de
> comportement « sûr » à choisir à sa place (bloquer pour toujours ? perdre la valeur en
> silence ?), donc Go **panique immédiatement** pour signaler l'erreur de programmation au plus
> tôt plutôt que de la masquer.

## Non bufferisé vs bufferisé

`make(chan T)` crée un canal **non bufferisé** ; `make(chan T, n)` un canal **bufferisé** de capacité
`n`. La différence est fondamentale :

- **Non bufferisé** : envoi et réception sont un **rendez-vous**. L'envoyeur **bloque** jusqu'à ce
  qu'un récepteur soit prêt (et inversement). C'est une **synchronisation** forte.
- **Bufferisé** : l'envoi réussit **sans attendre** tant que le tampon n'est pas plein ; la réception
  réussit sans attendre tant qu'il n'est pas vide. C'est une **file** d'attente bornée.

```
  NON BUFFERISÉ make(chan T)            BUFFERISÉ make(chan T, 3)
  rendez-vous : l'un attend l'autre     file : envoi tant qu'il reste de la place

    G1 --envoie--> [ ] <--reçoit-- G2     G1 --> [x][x][ ] --> G2
        (bloque jusqu'au récepteur)             (bloque seulement si plein)
```

> 💡 Règle de pouce : non bufferisé **par défaut** (la synchronisation est explicite et sûre).
> N'ajoutez un tampon que pour une **raison mesurée** (découpler producteur/consommateur, absorber des
> rafales) — pas « pour aller plus vite » au hasard.

> ⚠️ Un canal bufferisé n'est **pas** un outil de synchronisation fiable au-delà de sa capacité.
> Tant que le tampon n'est pas plein, `ch <- v` **réussit sans qu'aucun récepteur n'ait rien lu** :
> l'envoi garantit seulement qu'il restait de la place, pas qu'un message précédent a été traité.
> Avec `make(chan T, 3)`, les trois premiers envois peuvent tous réussir avant même que la goroutine
> réceptrice ait démarré. Le modèle mémoire formalise cette limite précisément : la _k_-ième
> réception happens-before la fin du _(k+capacité)_-ième envoi seulement
> ([Ch. 25](25-modele-memoire.md)). Pour une garantie de synchronisation à **chaque** message,
> restez en non bufferisé, ou utilisez un `sync.WaitGroup` ([Ch. 21](21-synchronisation.md)).

## `range` sur un canal

`for v := range ch` reçoit en boucle **jusqu'à la fermeture** du canal. C'est l'idiome pour consommer
un flux. La fermeture est ce qui **termine** la boucle : sans elle, le `range` bloque pour toujours.

```go
// code/ch20-channels-select/channels.go
func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // sans ce close, le range de l'appelant ne finirait jamais
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

for v := range gen(1, 2, 3) { ... } // 1, 2, 3 puis la boucle se termine
```

> 📌 Convention : c'est **l'envoyeur** qui ferme un canal, **jamais** le récepteur — et **une seule
> fois**. Le récepteur n'a aucun moyen de savoir si d'autres valeurs viendront.

## Directions : `chan<-` et `<-chan`

Un paramètre ou un retour peut **restreindre** le sens d'un canal. C'est une garantie **vérifiée à la
compilation**, qui documente le rôle de chaque partie :

```go
func produce(out chan<- int) { out <- 1 } // chan<- : envoi SEULEMENT
func consume(in <-chan int)  { <-in }      // <-chan : réception SEULEMENT
```

`gen` renvoie un `<-chan int` : l'appelant peut **recevoir** mais ni **envoyer** ni **fermer** — le
contrat est protégé par le typage.

## `select` : multiplexer plusieurs canaux

`select` attend sur **plusieurs** opérations de canal à la fois et exécute la **première prête**. Si
plusieurs le sont, il en choisit une par **sélection aléatoire uniforme** — c'est une garantie du
**langage** (spécification Go), pas un détail d'implémentation. Cette équité évite qu'un cas
toujours prêt n'**affame** les autres en étant systématiquement préféré (l'ordre d'écriture des
`case` n'a **aucune** influence sur la priorité) :

```go
select {
case v := <-in1:
	use(v)
case in2 <- x:
	sent()
case <-done:
	return // signal d'arrêt
}
```

```go
// code/ch20-channels-select/channels.go — deux cas TOUJOURS prêts : aucun ne doit dominer
func selectFairness(n int) (a, b int) {
	chA := make(chan struct{}, 1)
	chB := make(chan struct{}, 1)
	chA <- struct{}{}
	chB <- struct{}{}
	for range n {
		select {
		case <-chA:
			a++
			chA <- struct{}{} // remis aussitôt : les deux cas restent prêts au tour suivant
		case <-chB:
			b++
			chB <- struct{}{}
		}
	}
	return a, b
}
```

Mesuré (go1.26.4) sur 10 000 tirages : `case A=5104, case B=4896` — répartition proche de 50/50,
sans qu'aucune branche ne soit jamais privée d'exécution.

### `default` : non bloquant

Une branche `default` s'exécute **si aucune autre n'est prête** — `select` ne bloque alors jamais :

```go
// code/ch20-channels-select/channels.go
func trySend(ch chan<- int, v int) bool {
	select {
	case ch <- v:
		return true
	default:
		return false // l'envoi bloquerait : on renonce
	}
}
```

> ⚠️ Ne placez **jamais** un `select`+`default` seul dans une boucle serrée
> (`for { select { ...; default: } }`). Rien n'y bloque : la boucle tourne en **busy-wait** et
> sature un cœur CPU en testant le canal des millions de fois par seconde, pour rien. Si un nouvel
> essai est nécessaire, espacez les tentatives (`time.Sleep`) ou repassez en `select` **bloquant**
> (sans `default`), qui **dort** réellement tant qu'aucun cas n'est prêt.

### Timeout : `time.After`

`time.After(d)` renvoie un canal qui délivre une valeur après `d`. En branche de `select`, il borne
l'attente :

```go
func recvWithTimeout(ch <-chan int, d time.Duration) (int, bool) {
	select {
	case v := <-ch:
		return v, true
	case <-time.After(d):
		return 0, false // délai dépassé
	}
}
```

> 💡 `time.After` convient à un timeout **ponctuel**, comme ici. Dans une boucle qui réarme un
> délai à **chaque** itération (un `select` répété à haute fréquence), un nouveau minuteur est créé
> à chaque appel : préférez un seul `time.NewTimer` réinitialisé par `Reset` (voir le 🆕 Go 1.23
> ci-dessous sur le sort de ces minuteurs côté GC).

> 🔁 Pour une annulation **propagée** à travers tout un arbre d'appels (plutôt qu'un timeout local),
> on utilise `context` ([Ch. 22](22-context.md)), bâti **au-dessus** de `select` et `Done()`.

## Le canal `nil`

Un canal **nil** (jamais initialisé) **bloque pour toujours**, en envoi comme en réception. Loin
d'être un bug, c'est un **outil** : mettre une variable de canal à `nil` **désactive** sa branche dans
un `select` (la branche n'est plus jamais prête).

```go
for in != nil || other != nil {
	select {
	case v, ok := <-in:
		if !ok { in = nil; continue } // entrée tarie : on désactive sa branche
		use(v)
	case v, ok := <-other:
		if !ok { other = nil; continue }
		use(v)
	}
}
```

## Tableau récapitulatif : état du canal et opérations

Synthèse de tout ce qui précède — la **même** opération (`send`, `receive`, `close`) se comporte
différemment selon l'état du canal :

| État du canal                             | `ch <- v` (envoi)                         | `<-ch` (réception)                                               | `close(ch)`                           |
| ----------------------------------------- | ----------------------------------------- | ---------------------------------------------------------------- | ------------------------------------- |
| `nil` (jamais initialisé)                 | bloque **pour toujours**                  | bloque **pour toujours**                                         | **panique** `close of nil channel`    |
| ouvert, **non bufferisé**                 | bloque jusqu'à un récepteur prêt          | bloque jusqu'à un envoyeur prêt                                  | ferme normalement                     |
| ouvert, **bufferisé**, tampon non plein   | réussit **sans attendre** (place ajoutée) | réussit si le tampon contient une valeur, sinon bloque           | ferme normalement                     |
| ouvert, **bufferisé**, tampon plein       | bloque jusqu'à une place libre            | réussit **sans attendre** (dépile)                               | ferme normalement                     |
| **fermé**, tampon encore non vide         | **panique** `send on closed channel`      | réussit, renvoie une valeur restante, `ok == true`               | **panique** `close of closed channel` |
| **fermé**, tampon vide (ou non bufferisé) | **panique** `send on closed channel`      | réussit **sans attendre**, renvoie la valeur zéro, `ok == false` | **panique** `close of closed channel` |

Un canal non bufferisé fermé n'a jamais de valeurs « en attente » : il bascule directement sur la
dernière ligne. Cette table explique aussi le patron du canal `nil` ci-dessus : mettre une branche
à `nil` la fait **bloquer pour toujours**, donc `select` ne la choisit plus jamais — exactement
l'effet recherché pour la « désactiver ».

## Patron : signal d'arrêt + fan-in

Le canal `chan struct{}` **fermé** sert de **signal de diffusion** (`done`), déjà vu au
[Ch. 19](19-goroutines.md). Combiné au `range`, il permet le **fan-in** : fusionner plusieurs sources
en une.

```go
// code/ch20-channels-select/channels.go — fusionne N canaux, ferme à la fin
func fanIn(inputs ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, in := range inputs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range in { // draine l'entrée jusqu'à sa fermeture
				out <- v
			}
		}()
	}
	go func() { wg.Wait(); close(out) }() // fermer APRÈS que toutes les entrées sont taries
	return out
}
```

---

## 🆕 Go 1.2x

- Les canaux et `select` sont **stables depuis Go 1** : aucune évolution de sémantique. Ils restent la
  **fondation** de tout le reste — `context` ([Ch. 22](22-context.md)) et les patterns du
  [Ch. 23](23-patterns-concurrence.md) sont bâtis dessus.
- **1.23** — les `time.Timer`/`time.Ticker` (donc le minuteur sous-jacent à `time.After`) sont
  désormais **récupérables par le GC** dès qu'ils ne sont plus référencés, même sans appel à
  `Stop()`. Avant 1.23, un `time.After` créé dans une boucle à haute fréquence et jamais déclenché
  restait en mémoire jusqu'à son expiration — une fuite discrète mais réelle. Le correctif réduit le
  risque ; il ne supprime pas le coût d'**allouer** un minuteur par itération (voir 💡 plus haut).
- **1.25** — ce qui change, c'est **comment on les teste** : `testing/synctest` (GA) permet de tester
  un timeout `select`/`time.After` **sans attente réelle** grâce à une horloge virtuelle
  ([Ch. 23](23-patterns-concurrence.md)).

## ⚠️ Pièges

- **Envoyer sur un canal fermé** → **panique** `send on closed channel`. Idem `close` d'un canal déjà
  fermé (`close of closed channel`) ou `nil` (`close of nil channel`). Seul **l'envoyeur** ferme, une
  fois :

  ```go
  ch := make(chan int)
  close(ch)
  ch <- 1   // panic: send on closed channel
  close(ch) // panic: close of closed channel
  ```

- **Plusieurs envoyeurs, une seule fermeture** : si **N** goroutines envoient sur le même canal,
  **aucune** ne doit le fermer de sa propre initiative — celle qui termine en premier ne sait pas si
  les autres ont encore quelque chose à envoyer, et fermer pendant qu'une autre envoie déclenche le
  panic ci-dessus. Centralisez la fermeture chez un **coordinateur unique** qui attend que toutes les
  sources soient taries avant de fermer : c'est exactement le rôle de la goroutine
  `wg.Wait(); close(out)` dans `fanIn` plus haut.
- **Deadlock** : toutes les goroutines bloquées → `fatal error: all goroutines are asleep - deadlock!`.
  Cause typique : un `range` sur un canal **jamais fermé**, ou un envoi non bufferisé sans récepteur.
- **Fuite par canal non drainé** : une goroutine bloquée sur `out <- v` parce que personne ne lit
  **fuit** ([Ch. 19](19-goroutines.md)). Garantissez un récepteur, ou un `select` avec sortie.
- **Lire la valeur nulle sans tester `ok`** : après fermeture, `<-ch` renvoie `0`/`""`/`nil`. Utilisez
  `v, ok := <-ch` pour distinguer « zéro reçu » de « canal fermé ».
- **`select`+`default` en boucle serrée** : transforme une attente en **busy-wait** qui sature un
  cœur CPU pour rien (détail plus haut). Espacez les essais ou repassez en `select` bloquant.

## ⚡ Performance

Un canal **coordonne** des goroutines : il fait plus qu'un verrou, donc il **coûte** plus. Mesuré
(go1.26.4, Apple M3) :

```
   BenchmarkChanUnbuffered   185.2 ns/op   0 allocs   (rendez-vous : handoff de goroutine)
   BenchmarkChanBuffered      43.1 ns/op   0 allocs   (tampon 64 : pas de rendez-vous à chaque envoi)
```

- Un canal **non bufferisé** implique un **rendez-vous** (réveil d'une goroutine) : ~185 ns, bien plus
  qu'un `mutex` (~135 ns) ou un `atomic` (~40 ns, [Ch. 21](21-synchronisation.md)).
- Un **tampon** amortit le rendez-vous (~4× ici) quand producteur et consommateur vont à des rythmes
  différents.
- **Choisissez l'outil** : un canal pour **transférer la propriété** d'une donnée ou **signaler** ; un
  `mutex`/`atomic` pour **protéger** un état partagé en place. Un canal n'est pas un mutex plus lent.
- Le coût d'un `select` croît avec son **nombre de cas** : le runtime évalue chacun et tire un ordre
  aléatoire avant de bloquer ou de réessayer. Négligeable pour deux ou trois cas ; au-delà d'une
  poignée, un `select` tentaculaire devient un signe qu'un **fan-in** centralisé
  ([Ch. 23](23-patterns-concurrence.md)) serait plus clair et plus rapide.
- 🔁 Garanties mémoire d'un canal au [Ch. 25](25-modele-memoire.md).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch20-channels-select
go test -race ./ch20-channels-select/...
```

À essayer :

1. Faites paniquer un programme avec `send on closed channel`, puis réparez l'ordre des opérations.
2. Provoquez un `deadlock` en oubliant le `close` dans `gen`, lisez le message du runtime.
3. Ajoutez à `fanIn` un canal `done` pour **arrêter** la fusion avant épuisement des entrées.
4. Étendez `selectFairness` à **trois** cas toujours prêts : la répartition reste-t-elle proche d'un
   tiers chacun ?

---

## 📌 À retenir

- Un canal est un tuyau **typé** qui **transfère** une valeur **et synchronise** ; `<-` pointe dans le
  sens du flux.
- **Non bufferisé** = rendez-vous (synchronisation forte) ; **bufferisé** = file bornée. Non bufferisé
  par défaut.
- `range` consomme jusqu'à la **fermeture** ; **l'envoyeur** ferme, une seule fois, jamais le récepteur
  — et jamais depuis plusieurs goroutines productrices indépendantes sans coordinateur unique.
- Envoyer sur un canal fermé **panique** (promesse violée) ; recevoir renvoie la valeur zéro avec
  `ok == false` (aucun risque à interroger un canal déjà fermé). Le tableau récapitulatif ci-dessus
  couvre les six combinaisons nil/ouvert/fermé × bufferisé/non bufferisé.
- `select` attend la **première** opération prête, choisie par **tirage aléatoire uniforme** s'il y en
  a plusieurs ; `default` le rend non bloquant (jamais en boucle serrée), `time.After` le borne dans
  le temps ; un canal `nil` **désactive** une branche.
- Un canal **coordonne** (plus cher) ; un `mutex`/`atomic` **protège** (moins cher) : choisissez selon
  l'intention.

## 🔁 Pour aller plus loin

- [Ch. 21 — Synchronisation](21-synchronisation.md) : quand les canaux ne suffisent pas (état partagé).
- [Ch. 22 — `context`](22-context.md) : l'annulation propagée, bâtie sur `select`/`Done()`.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : pipelines, worker pools, fan-out.
- [Ch. 25 — Modèle mémoire](25-modele-memoire.md) : les garanties _happens-before_ d'un canal.
