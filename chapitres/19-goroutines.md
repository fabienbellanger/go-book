# 19 — Goroutines

> **Objectif** — Lancer des tâches concurrentes avec `go`, comprendre **ce qu'est** une goroutine
> (et son coût réel), la différence **concurrence ≠ parallélisme**, le **cycle de vie** d'une
> goroutine, et surtout comment éviter les **fuites** par un arrêt propre.
>
> **Prérequis** — [Ch. 5 — Fonctions](05-fonctions.md), [Ch. 15 — Closures](15-closures.md) (capture)

---

## Introduction

La concurrence est un **pilier** de Go : elle est dans le langage, pas dans une bibliothèque. L'unité
de base est la **goroutine** — une fonction qui s'exécute **indépendamment**, gérée par le **runtime**
Go et non directement par le système d'exploitation. On en lance des **centaines de milliers** sans y
penser, là où autant de threads OS écrouleraient la machine.

Ce chapitre pose le **modèle** : comment lancer une goroutine, ce qu'elle coûte, et comment garantir
qu'elle **s'arrête** quand il le faut. La **communication** entre goroutines (canaux) vient au
[Ch. 20](20-channels-select.md), la **protection de l'état partagé** au [Ch. 21](21-synchronisation.md).
L'exemple est dans [`code/ch19-goroutines/`](../code/ch19-goroutines/).

---

## Lancer une goroutine : `go`

Le mot-clé `go` devant un appel de fonction **démarre une nouvelle goroutine** et **n'attend pas** :
l'exécution continue immédiatement à la ligne suivante.

```go
go doWork()              // lance doWork() en concurrence ; ne bloque pas
go func() { ... }()      // souvent une fonction littérale (closure)
fmt.Println("continue")  // s'exécute sans attendre doWork
```

> 💡 Les arguments d'un appel `go f(x)` sont évalués **immédiatement**, à l'instruction `go` —
> exactement comme pour `defer` ([Ch. 16](16-defer.md)). C'est différent d'une **closure** qui
> capture une variable sans la passer en paramètre : `go func() { use(x) }()` lit `x` **au moment
> de l'exécution réelle** de la goroutine, pas à son lancement. Cette distinction est au cœur du
> piège historique de capture de boucle (voir plus bas) : passer la variable en paramètre
> (`go func(x int) { ... }(x)`) l'évitait déjà avant Go 1.22.

> ⚠️ `main` est elle-même une goroutine. Quand `main` **retourne**, le programme s'arrête **sans
> attendre** les autres goroutines. Lancer `go work()` puis sortir de `main` peut ne **rien** exécuter.
> Il faut donc une **synchronisation** explicite pour attendre la fin.

## Attendre la fin : `sync.WaitGroup`

Pour attendre qu'un groupe de goroutines termine, on utilise un **`sync.WaitGroup`**
([Ch. 21](21-synchronisation.md)) : un compteur qu'on **incrémente** avant de lancer (`Add`),
**décrémente** à la fin de chaque goroutine (`Done`), et sur lequel on **bloque** (`Wait`).

```go
// code/ch19-goroutines/goroutines.go
func parallelMap[T, U any](items []T, f func(T) U) []U {
	out := make([]U, len(items))
	var wg sync.WaitGroup
	for i, item := range items { // Go 1.22+ : i et item sont propres à l'itération
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = f(item) // index distinct par goroutine : aucune course
		}()
	}
	wg.Wait() // bloque jusqu'à ce que les len(items) goroutines aient terminé
	return out
}
```

Chaque goroutine écrit à un **index distinct** de `out` : il n'y a pas d'état partagé en écriture,
donc **pas de course** — `go test -race` le confirme. Et grâce à la **portée par itération** de Go
1.22 ([Ch. 15](15-closures.md)), capturer `i` et `item` dans la closure est **sûr** ; avant 1.22, ce
code était le bug de concurrence n°1.

> ⚠️ Le **deuxième** piège classique du trio `Add`/`Done`/`Wait` : appeler `Add(1)` **depuis** la
> goroutine plutôt qu'**avant** le `go`, comme dans `parallelMap` ci-dessus. `Wait()` ignore les
> `Add` à venir : si l'ordonnanceur fait tourner `wg.Wait()` avant que la goroutine ait eu la main
> pour s'enregistrer, il peut **retourner trop tôt**, compteur encore à zéro. Go 1.25 supprime le
> risque avec **`WaitGroup.Go`**, qui lance la goroutine et l'enregistre **atomiquement**
> ([Ch. 21](21-synchronisation.md)) ; l'analyzer `go vet waitgroup` détecte aussi l'erreur.

## Goroutine vs thread OS

Une goroutine **n'est pas** un thread du système. Le runtime **multiplexe** des milliers de
goroutines sur une poignée de threads OS (le modèle **G-M-P**, [Ch. 28](28-ordonnanceur-gmp.md)) :

```
  des centaines de milliers de goroutines (G)      quelques threads OS (M, ~ GOMAXPROCS)

   G G G G G G G G G G G G G G G G G G G G G G
    \   \   \    \    |    |    |    /   /   /
     \   \   \    \   |    |    |   /   /   /
      v   v   v    v  v    v    v  v   v   v
         +-------+       +-------+       +-------+
         |  M 0  |       |  M 1  |       |  M 2  |     <- exécutent du code Go
         +-------+       +-------+       +-------+
             |               |               |
          cœur 0          cœur 1          cœur 2

  Le runtime choisit à tout instant QUELLES goroutines tournent sur QUELS threads, suspend
  celles qui bloquent, en réveille d'autres à leur place. Mécanique complète (files
  d'exécution locales, work-stealing, préemption) : Ch. 28.
```

C'est ce multiplexage **M:N** (M threads pour N goroutines, au lieu d'un thread OS par tâche comme
dans le modèle 1:1 traditionnel) qui rend les goroutines abondantes : le nombre de threads M reste
proche de `GOMAXPROCS`, **quel que soit** le nombre de goroutines lancées.

| Critère                | Goroutine                                      | Thread OS                   |
| ---------------------- | ---------------------------------------------- | --------------------------- |
| Création               | `go f()` — quelques ns                         | appel système — quelques µs |
| Pile initiale          | **~2 Ko**, **agrandie à la demande**           | 1–8 **Mo**, fixe            |
| Ordonnancement         | par le **runtime Go** (coopératif + préemptif) | par le **noyau**            |
| Nombre réaliste        | **centaines de milliers**                      | quelques milliers           |
| Changement de contexte | très bon marché (en espace user)               | coûteux (passage noyau)     |

Cet écart tient à un choix structurel, pas à un simple réglage. Créer un thread est un **appel
système** (`clone`/`pthread_create`) : le noyau lui réserve une pile dont la taille est figée **une
fois pour toutes**, car cette pile ne peut **jamais être déplacée** — un pointeur brut vers son
intérieur peut exister ailleurs (y compris en C via cgo). Le système la dimensionne donc largement
(1 à 8 Mo) par prudence. Une goroutine, elle, est une structure gérée entièrement par le runtime Go
en espace utilisateur : sa pile peut être **copiée** vers un nouvel emplacement plus grand quand
elle déborde, car le compilateur connaît précisément chaque pointeur qui y pointe et peut tous les
réajuster ([Ch. 26](26-allocation-escape.md)) — d'où une pile de départ minuscule, sans gaspillage.

Mesuré sur go1.26.4 (arm64), lancer **100 000** goroutines bloquées :

```
   runtime.NumGoroutine() : 1  ->  100001
   StackInuse             : +205 Mo   (~2049 octets / goroutine)
```

Une goroutine démarre donc avec **2 Ko de pile**, que le runtime **agrandit et rétrécit**
automatiquement ([Ch. 26](26-allocation-escape.md)). C'est ce qui rend leur multitude abordable.

## Concurrence ≠ parallélisme

Deux notions **distinctes**, souvent confondues :

- **Concurrence** : _structurer_ un programme en tâches indépendantes qui **progressent** sur des
  périodes qui se chevauchent. C'est un modèle de **conception**.
- **Parallélisme** : _exécuter_ littéralement plusieurs calculs **au même instant**, sur plusieurs
  cœurs. C'est un fait d'**exécution**.

```
  CONCURRENCE (1 cœur) : les tâches s'entrelacent dans le temps
     cœur 0 : [A][B][A][B][A][B]      A et B progressent tous les deux

  PARALLÉLISME (2 cœurs) : les tâches s'exécutent en même temps
     cœur 0 : [A][A][A][A]
     cœur 1 : [B][B][B][B]            A et B au même instant
```

Un programme concurrent **bien écrit** devient parallèle **automatiquement** si `GOMAXPROCS` > 1 (le
nombre de cœurs utilisables, [Ch. 28](28-ordonnanceur-gmp.md)) — sans changer une ligne. La
concurrence est **votre** affaire (le design) ; le parallélisme est celle du **runtime**.

## Cycle de vie & fuites de goroutines

Une goroutine vit tant que sa fonction n'a pas **retourné**. Elle disparaît à son retour. Le danger :
une goroutine qui **se bloque pour toujours** (sur un canal, un verrou, un I/O) ne retourne **jamais**
— c'est une **fuite de goroutine**. Sa pile reste allouée, et ce qu'elle capture aussi.

```go
// FUITE : personne n'enverra jamais sur ch ; la goroutine reste bloquée à vie.
func leak() {
	ch := make(chan int)
	go func() {
		<-ch // bloqué pour toujours : fuite
	}()
	// la fonction retourne, mais la goroutine survit, inutile et invisible
}
```

La fuite la plus fréquente en production prend la forme **inverse** : un **envoi** bloqué parce que
plus personne ne lit.

```go
// FUITE : si l'appelant a abandonné (timeout, erreur ailleurs), l'envoi reste bloqué à vie.
func produce(results chan<- int) {
	v := compute()
	results <- v // si plus personne ne lit results, cette ligne ne retourne JAMAIS
}
```

Le correctif est le même dans les deux sens : donner à la goroutine un moyen de **renoncer** — un
`select` avec un second cas sur un canal d'arrêt ou `ctx.Done()` ([Ch. 22](22-context.md)), ou un
canal **bufferisé** d'une capacité suffisante pour que l'envoi n'ait jamais besoin d'attendre.

Les fuites ne plantent pas le programme : elles le font **enfler** lentement (mémoire, et parfois
descripteurs). On les traque avec `runtime.NumGoroutine()` qui **monte sans redescendre**, ou avec le
profil **`goroutineleak`** (voir 🆕).

## Arrêt propre : le canal d'arrêt

La parade : **donner à chaque goroutine un moyen de s'arrêter**. Le patron canonique est un **canal
d'arrêt** (`chan struct{}`) que l'on **ferme** pour signaler « termine-toi ». La goroutine le
surveille et **retourne** dès qu'il est fermé.

```go
// code/ch19-goroutines/goroutines.go
func tickUntilStop(stop <-chan struct{}) (count *atomic.Int64, done <-chan struct{}) {
	count = &atomic.Int64{}
	d := make(chan struct{})
	go func() {
		defer close(d) // signale « j'ai terminé »
		for {
			select {
			case <-stop:
				return // arrêt demandé : la goroutine se termine (pas de fuite)
			default:
				count.Add(1)
			}
		}
	}()
	return count, d
}

// Côté appelant : on demande l'arrêt, puis on ATTEND la confirmation.
close(stop) // demande d'arrêt
<-done      // la goroutine a bel et bien fini
```

Fermer un canal est le signal de diffusion idéal : **toutes** les goroutines qui lisent `stop` le
voient. On verra la mécanique de `select` et des canaux au [Ch. 20](20-channels-select.md), et le
`context.Context` qui **standardise** ce signal d'annulation au [Ch. 22](22-context.md).

---

## 🆕 Go 1.2x

- **1.26** — profil **`goroutineleak`** (expérimental, `GOEXPERIMENT=goroutineleakprofile`) :
  `pprof.Lookup("goroutineleak")` (ou `/debug/pprof/goroutineleak`) liste les goroutines **bloquées à
  jamais et inaccessibles**. Vérifié sur go1.26.4 sur le `leak()` ci-dessus :

```
$ GOEXPERIMENT=goroutineleakprofile go run .
goroutineleak profile: total 1
1 @ ...
#  main.leak.func1+0x23  .../main.go:14   (<-ch)
```

Sans l'expérience, `pprof.Lookup("goroutineleak")` renvoie `nil`. 🔁 [Ch. 29](29-observabilite-runtime.md).

- **1.26** — nouvelles métriques d'ordonnanceur dans `runtime/metrics` :
  `/sched/goroutines-created`, et la ventilation `/sched/goroutines/{running,runnable,waiting}`,
  `/sched/threads/total`.
- **1.25** — `GOMAXPROCS` devient **conscient des limites cgroups** (conteneurs) et se réajuste à
  chaud ([Ch. 28](28-ordonnanceur-gmp.md)).

## ⚠️ Pièges

- **Sortir de `main` sans attendre** : les goroutines lancées peuvent ne jamais s'exécuter. Synchronisez
  (`WaitGroup`, canal, `context`).
- **Fuite de goroutine** : toute goroutine bloquée sur un canal/verrou sans issue **fuit**, que ce
  soit en réception ou en **envoi**. Donnez toujours un chemin de sortie (canal d'arrêt, `context`,
  timeout).
- **Capturer une variable de boucle** (avant Go 1.22) : toutes les goroutines lisaient la **même**
  variable, souvent sa valeur finale. Réglé en 1.22 (chaque itération a sa propre variable,
  [Ch. 15](15-closures.md)), mais vérifiez la ligne `go` du `go.mod` — un module resté en `go 1.21`
  garde l'ancien comportement même compilé avec Go 1.26. La parade pré-1.22, toujours valable :
  passer la variable en **paramètre** plutôt que la capturer (voir le 💡 plus haut).
- **Panique non rattrapée dans une goroutine** : elle fait planter **tout le programme**, même si
  `main` a son propre `recover`. `recover` ne fonctionne que dans la **même** chaîne d'appel ; un
  `recover` qui protège `main` ne protège **pas** les goroutines qu'elle lance. Chaque goroutine
  susceptible de paniquer doit avoir son **propre** `defer`/`recover` ([Ch. 17](17-panic-recover.md)).
- **Croire que `go` = parallèle** : avec `GOMAXPROCS=1`, tout est concurrent mais **pas** parallèle.
- **Supposer un ordre d'exécution** : l'ordonnancement n'est **pas** déterministe. Ne dépendez jamais
  de « quelle goroutine part en premier ».

## ⚡ Performance

- Lancer une goroutine coûte **quelques nanosecondes** et **~2 Ko** de pile initiale (contre des
  micro-secondes et des méga-octets pour un thread). C'est conçu pour être **abondant**.
- `GOMAXPROCS` borne le nombre de threads qui **exécutent** du code Go en parallèle, pas le nombre de
  goroutines qu'on peut **créer** : rien n'empêche de lancer un million de goroutines avec
  `GOMAXPROCS=1` — elles s'entrelacent simplement sur un seul cœur, sans la moindre erreur.
- La pile **croît et décroît** par copie à la demande ([Ch. 26](26-allocation-escape.md)) : pas besoin
  de la dimensionner.
- Mais une goroutine n'est **pas gratuite** : 100 k goroutines, c'est ~200 Mo rien qu'en piles.
  Pour des tâches **nombreuses et courtes**, bornez le parallélisme avec un **pool de workers**
  ([Ch. 23](23-patterns-concurrence.md)) plutôt qu'une goroutine par tâche.
- 🔁 Ordonnancement, work-stealing et préemption au [Ch. 28](28-ordonnanceur-gmp.md).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch19-goroutines
go test -race ./ch19-goroutines/...   # -race : prouve l'absence de course dans parallelMap
```

À essayer :

1. Lancez 100 000 goroutines bloquées et affichez `runtime.NumGoroutine()` avant/après.
2. Écrivez la version **qui fuit** (goroutine bloquée sur un canal) et observez le compteur monter ;
   réparez-la avec un canal d'arrêt.
3. Compilez avec `GOEXPERIMENT=goroutineleakprofile` et dumpez le profil `goroutineleak`.
4. Écrivez la variante **qui fuit par envoi** (`produce` ci-dessus, appelée sans jamais lire
   `results`) ; ajoutez un `select` avec un canal d'arrêt pour la réparer.

---

## 📌 À retenir

- `go f()` lance une goroutine et **ne bloque pas** ; `main` qui retourne **tue** tout — synchronisez.
- Une goroutine est **légère** : ~2 Ko de pile, multiplexée sur les threads OS par le runtime ; on en
  lance des **centaines de milliers**.
- **Concurrence** (design : tâches qui se chevauchent) **≠ parallélisme** (exécution simultanée sur
  plusieurs cœurs, piloté par `GOMAXPROCS`).
- Une goroutine bloquée sans issue **fuit** : donnez-lui toujours un **chemin d'arrêt** (canal fermé,
  `context`).
- Ne dépendez **jamais** de l'ordre d'ordonnancement.

## 🔁 Pour aller plus loin

- [Ch. 20 — Channels & `select`](20-channels-select.md) : faire **communiquer** les goroutines.
- [Ch. 21 — Synchronisation](21-synchronisation.md) : `WaitGroup`, mutex, atomics pour l'état partagé.
- [Ch. 22 — `context`](22-context.md) : standardiser l'annulation et les délais.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : pools, pipelines, détection de fuites.
- [Ch. 28 — L'ordonnanceur (G-M-P)](28-ordonnanceur-gmp.md) : comment les goroutines tournent vraiment.
