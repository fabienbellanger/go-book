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
plusieurs le sont, il en choisit une **au hasard** (équité).

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
- **1.25** — ce qui change, c'est **comment on les teste** : `testing/synctest` (GA) permet de tester
  un timeout `select`/`time.After` **sans attente réelle** grâce à une horloge virtuelle
  ([Ch. 23](23-patterns-concurrence.md)).

## ⚠️ Pièges

- **Envoyer sur un canal fermé** → **panique** `send on closed channel`. Idem `close` d'un canal déjà
  fermé (`close of closed channel`) ou `nil` (`close of nil channel`). Seul **l'envoyeur** ferme, une
  fois.
- **Deadlock** : toutes les goroutines bloquées → `fatal error: all goroutines are asleep - deadlock!`.
  Cause typique : un `range` sur un canal **jamais fermé**, ou un envoi non bufferisé sans récepteur.
- **Fuite par canal non drainé** : une goroutine bloquée sur `out <- v` parce que personne ne lit
  **fuit** ([Ch. 19](19-goroutines.md)). Garantissez un récepteur, ou un `select` avec sortie.
- **Lire la valeur nulle sans tester `ok`** : après fermeture, `<-ch` renvoie `0`/`""`/`nil`. Utilisez
  `v, ok := <-ch` pour distinguer « zéro reçu » de « canal fermé ».

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

---

## 📌 À retenir

- Un canal est un tuyau **typé** qui **transfère** une valeur **et synchronise** ; `<-` pointe dans le
  sens du flux.
- **Non bufferisé** = rendez-vous (synchronisation forte) ; **bufferisé** = file bornée. Non bufferisé
  par défaut.
- `range` consomme jusqu'à la **fermeture** ; **l'envoyeur** ferme, une seule fois, jamais le récepteur.
- `select` attend la **première** opération prête ; `default` le rend non bloquant, `time.After` le
  borne dans le temps ; un canal `nil` **désactive** une branche.
- Un canal **coordonne** (plus cher) ; un `mutex`/`atomic` **protège** (moins cher) : choisissez selon
  l'intention.

## 🔁 Pour aller plus loin

- [Ch. 21 — Synchronisation](21-synchronisation.md) : quand les canaux ne suffisent pas (état partagé).
- [Ch. 22 — `context`](22-context.md) : l'annulation propagée, bâtie sur `select`/`Done()`.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : pipelines, worker pools, fan-out.
- [Ch. 25 — Modèle mémoire](25-modele-memoire.md) : les garanties _happens-before_ d'un canal.
