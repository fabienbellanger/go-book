# 54 — Architecture & organisation d'une application Go

> **Objectif** — Savoir **organiser** une application Go qui grossit : quand
> découper en paquets, quel layout adopter (`cmd/`, `internal/`), comment
> **injecter les dépendances** sans framework, où placer les **interfaces** pour
> garder un cœur métier testable, et comment éviter les cycles d'import. Le fil
> rouge est une petite application de notes, dans [`code/ch54-architecture/`](../code/ch54-architecture/).
>
> **Prérequis** — [Ch. 9 — Interfaces](09-interfaces.md),
> [Ch. 12 — Packages & modules](12-packages-modules.md),
> [Ch. 10 — Erreurs](10-erreurs.md), [Ch. 13 — Tests](13-tests-outillage.md).

---

## Introduction

Le langage vous a tout appris sur les briques : types, fonctions, interfaces,
paquets. Reste la question qui fait trébucher la plupart des développeurs
intermédiaires : **où mettre quoi ?** Un `main.go` de 2 000 lignes fonctionne,
jusqu'au jour où l'on veut tester la logique métier sans lancer la base de
données, réutiliser un bout de code ailleurs, ou simplement retrouver où une
règle est appliquée.

Go n'impose **aucune** architecture — pas de convention « MVC », pas de dossier
`src/`. À la place, il fournit deux garde-fous discrets mais puissants (le
dossier `internal/` et l'**interdiction des cycles d'import**) et une culture :
**commencer plat, découper quand la douleur apparaît.** Ce chapitre montre la
progression naturelle, du fichier unique à une application en couches testable.

> 💡 **La bonne architecture est celle qu'on peut défaire.** Le vrai risque n'est
> pas de sous-découper (facile à corriger) mais de **sur-découper** trop tôt :
> cinq paquets et trois interfaces pour cent lignes de logique, c'est du coût sans
> bénéfice.

## Étape 0 — plat tant que ça ne fait pas mal

Un outil ou un service qui débute tient dans **un seul paquet `main`**, quitte à
le répartir sur plusieurs fichiers (`main.go`, `store.go`, `handlers.go`). Le
compilateur les voit comme un tout ; découper en fichiers n'a aucun coût et aide
déjà à s'y retrouver.

On ne crée un **paquet** séparé que lorsqu'un besoin concret l'exige :

- **tester** une logique isolément, sans démarrer tout le programme ;
- **réutiliser** du code entre plusieurs binaires (`cmd/serveur`, `cmd/cli`) ;
- **cacher** des détails d'implémentation derrière une frontière stable ;
- **casser** un fichier devenu illisible dont les morceaux ne se parlent plus.

Si aucune de ces raisons ne s'applique, rester plat est le **bon** choix, pas un
aveu de paresse.

## Le layout usuel

Quand l'application grandit, la communauté Go converge vers une poignée de
dossiers **conventionnels** (aucun n'est imposé par l'outillage, sauf `internal/`) :

```
  monapp/
  |-- go.mod                  module example.com/monapp
  |-- cmd/                     un sous-dossier par BINAIRE
  |   |-- monapp/
  |   |   `-- main.go          point d'entrée + CÂBLAGE, presque pas de logique
  |   `-- monapp-cli/
  |       `-- main.go          un second binaire partageant le même internal/
  |-- internal/                code PRIVÉ au module (imposé par le compilateur)
  |   |-- domain/              types métier, aucune dépendance technique
  |   |-- service/            logique applicative + interfaces consommées
  |   |-- store/               adaptateurs de persistance (SQL, mémoire...)
  |   `-- httpapi/             adaptateur d'entrée HTTP (handlers)
  `-- pkg/                     (optionnel) code EXPORTÉ, réutilisable hors module
```

- **`cmd/<app>/main.go`** — le point d'entrée. Son rôle idéal : lire la config,
  construire les dépendances concrètes, les **câbler**, lancer, gérer l'arrêt.
  Presque aucune logique métier (🔁 [Ch. 48](48-processus-signaux-cli.md) pour
  les signaux et la CLI).
- **`internal/`** — le compilateur **interdit** d'importer un paquet situé sous
  `internal/` depuis l'extérieur du module (ou du sous-arbre) qui le contient
  (🔁 [Ch. 12](12-packages-modules.md)). C'est le seul contrôle d'accès natif de
  Go : tout ce qui n'est pas une API publique volontaire va là.
- **`pkg/`** — dossier **optionnel** et souvent inutile. Il ne signale qu'une
  intention (« ce code est fait pour être importé par d'autres modules »).
  Dans le doute, préférez `internal/` : on peut toujours exporter plus tard, on
  peut rarement dé-publier une API.

> ⚠️ Ne recréez pas un `src/` à la Java, ni un paquet unique géant nommé comme le
> projet. Le nom d'un paquet Go décrit **ce qu'il fournit** (`store`, `token`),
> pas où il se trouve.

## Découper par couche ou par feature ?

Deux façons de tailler `internal/` :

| Découpage        | `internal/` contient…                          | Force                                     | Faiblesse                                             |
| ---------------- | ---------------------------------------------- | ----------------------------------------- | ----------------------------------------------------- |
| **par couche**   | `domain/`, `service/`, `store/`, `httpapi/`    | rôles clairs, frontières techniques nettes | une feature s'éparpille dans 4 paquets                |
| **par feature**  | `notes/`, `users/`, `billing/` (chacun complet) | tout ce qui touche « notes » au même endroit | risque de dupliquer la plomberie entre features        |

Il n'y a pas de gagnant universel. Une heuristique qui marche bien : **par
feature au premier niveau, par couche à l'intérieur** d'une feature quand elle
grossit. Ce chapitre illustre le découpage par couche car il rend les rôles
explicites, mais le raisonnement sur les interfaces vaut dans les deux cas.

> ⚠️ **Le piège du paquet fourre-tout.** `utils`, `common`, `helpers`, `base` :
> ces noms ne décrivent rien, attirent tout, et deviennent des aimants à cycles
> d'import. Si une fonction manipule des chaînes, elle va dans un paquet `text` ;
> si elle formate des dates, dans `dateformat`. Un paquet doit avoir **un thème**.

## L'injection de dépendances, sans framework

En Go, « injecter une dépendance » ne demande **aucune bibliothèque** : on passe
la dépendance au **constructeur**. Le service ne fabrique pas son store ni son
logger, il les **reçoit**.

```go
// code/ch54-architecture/service/service.go
type Service struct {
	store NoteStore    // une INTERFACE, pas un type concret
	log   *slog.Logger
}

// New construit un Service. Les dépendances sont INJECTÉES : aucun accès à un
// état global, tout est explicite et donc remplaçable en test.
func New(store NoteStore, log *slog.Logger) *Service {
	return &Service{store: store, log: log}
}
```

Le câblage — brancher les implémentations concrètes les unes dans les autres —
se fait **une seule fois**, dans `main`. On appelle cet endroit le
*composition root* : c'est le seul qui connaît à la fois tous les types concrets
et la façon de les assembler.

```go
// code/ch54-architecture/main.go
st := store.NewMem()          // implémentation concrète (périphérie)
svc := service.New(st, log)   // injectée dans le cœur, vu comme NoteStore
```

Changer de store (mémoire → SQL) ne toucherait **que cette ligne**. C'est tout
le bénéfice : les dépendances remontent vers `main`, le cœur reste ignorant.

> 💡 Pas besoin de Wire, Fx ou d'un conteneur d'injection pour commencer. Le
> câblage manuel dans `main` reste lisible longtemps ; on n'ajoute un générateur
> que si le graphe de dépendances devient vraiment gros.

## Les frontières par interfaces

C'est le cœur du chapitre. Une bonne architecture Go tient à **une règle sur les
interfaces** et à **un choix sur leur emplacement**.

### Définir l'interface côté consommateur

En Go, l'interface appartient au **paquet qui l'utilise**, pas à celui qui
l'implémente. Le `service` a besoin de persister des notes : c'est **lui** qui
déclare l'interface minimale dont il a besoin.

```go
// code/ch54-architecture/service/service.go
// NoteStore est définie CÔTÉ CONSOMMATEUR : le service déclare exactement les
// opérations dont il a besoin, ni plus ni moins.
type NoteStore interface {
	Create(ctx context.Context, title, body string) (domain.Note, error)
	Get(ctx context.Context, id string) (domain.Note, error)
	List(ctx context.Context) ([]domain.Note, error)
}
```

Le paquet `store`, lui, ne déclare **rien** : il expose un type concret `*Mem`
qui, *par hasard heureux du typage structurel* ([Ch. 9](09-interfaces.md)),
satisfait `NoteStore`.

```go
// code/ch54-architecture/store/memory.go
// NewMem renvoie le TYPE CONCRET *Mem, pas une interface : « accepter des
// interfaces, renvoyer des structs ».
func NewMem() *Mem { return &Mem{notes: make(map[string]domain.Note)} }
```

```
  service (cœur)                          store (périphérie)
  +---------------------+                 +---------------------+
  | déclare l'interface |  <-- satisfait  |  *Mem (type concret)|
  |   NoteStore         |     (implicite) |  aucune interface   |
  +---------------------+                 +---------------------+
        ^  dépend de domain                     |  dépend de domain
        |                                        |
        +-------------- domain (feuille) --------+
             la dépendance pointe TOUJOURS vers le cœur
```

Pourquoi côté consommateur ? Parce que le besoin est **local et petit**. Si le
`store` définissait une grosse interface `Repository` avec vingt méthodes, chaque
consommateur en dépendrait tout entier. En laissant chaque consommateur déclarer
les deux ou trois méthodes qu'il utilise, on obtient des interfaces **petites**,
faciles à implémenter et à simuler.

> 💡 « The bigger the interface, the weaker the abstraction. » (Rob Pike). Les
> interfaces les plus utiles de la stdlib ont **une** méthode (`io.Reader`,
> `io.Writer`, `fmt.Stringer`). Visez la même sobriété.

### Le bénéfice : un cœur testable avec des fakes

Comme le service ne dépend que de `NoteStore`, on le teste en lui injectant une
implémentation **factice** — sans base de données, sans I/O, en microsecondes :

```go
// code/ch54-architecture/service/service_test.go
type fakeStore struct {
	created []domain.Note
	getErr  error
}

func (f *fakeStore) Create(_ context.Context, title, body string) (domain.Note, error) {
	n := domain.Note{ID: "fake1", Title: title, Body: body}
	f.created = append(f.created, n)
	return n, nil
}
// ... Get, List ...

func TestServiceCreateRejectsEmptyTitle(t *testing.T) {
	svc := New(&fakeStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := svc.Create(context.Background(), "   ", "corps")
	if !errors.Is(err, ErrEmptyTitle) {
		t.Fatalf("attendu ErrEmptyTitle, obtenu %v", err)
	}
}
```

L'implémentation concrète (SQL, HTTP, fichier) vit en **périphérie** ; le cœur
métier ne dépend que d'interfaces qu'il maîtrise (🔁 [Ch. 13](13-tests-outillage.md)).

## Éviter les cycles d'import

Go **interdit** les cycles d'import : si `a` importe `b`, alors `b` ne peut pas
importer `a`, ni directement ni via un tiers. Le compilateur refuse net. C'est une
contrainte salutaire — un cycle traduit presque toujours deux responsabilités
mal séparées — mais il faut savoir la contourner.

Trois techniques, de la plus simple à la plus structurante :

1. **Extraire un paquet feuille.** Les types partagés (ici `domain.Note`,
   `domain.ErrNotFound`) vont dans un paquet **sans dépendance**, que tout le
   monde peut importer sans créer de cycle. C'est le rôle de `domain/`.
2. **Inverser la dépendance par une interface.** Si `service` a besoin de
   `store` mais qu'on ne veut pas que le cœur dépende de la périphérie, on place
   l'interface dans `service` (côté consommateur) : c'est `store` qui « dépend »
   du contrat, pas l'inverse.
3. **Fusionner** deux paquets qui s'appellent sans cesse : leur séparation était
   peut-être artificielle.

```
  cycle interdit :   service  <---->  store        (refusé par le compilateur)

  solution 1 :       service  --->  domain  <---  store
  solution 2 :       store  ---(satisfait l'interface de)-->  service
```

## Faire circuler `context` et les erreurs

À travers les couches, deux valeurs voyagent toujours ensemble :

- **`context.Context`** — premier paramètre de toute méthode qui fait de l'I/O
  ou peut être annulée. Il traverse `httpapi → service → store` sans être stocké
  dans un struct (🔁 [Ch. 22](22-context.md)).
- **L'erreur** — enveloppée à chaque étage avec `%w` pour garder la **cause**
  identifiable via `errors.Is`/`errors.As` (🔁 [Ch. 10](10-erreurs.md)) :

  ```go
  // code/ch54-architecture/service/service.go
  n, err := s.store.Get(ctx, id)
  if err != nil {
      return domain.Note{}, fmt.Errorf("service.Get %q: %w", id, err)
  }
  ```

  La sentinelle `domain.ErrNotFound` remonte ainsi du store jusqu'au handler
  HTTP, qui peut la traduire en `404` sans connaître le store.

> ⚠️ **Ne loggez pas ET ne renvoyez pas la même erreur.** Chaque couche qui logge
> puis propage produit la même erreur cinq fois dans les journaux. Choisissez :
> soit on **gère** l'erreur (on la logge, on la mange), soit on la **renvoie**
> enrichie. En pratique, seul le point le plus haut (souvent `main` ou le
> middleware HTTP) logge.

## La configuration

La config suit le même principe que les dépendances : **lue au bord, injectée
vers le centre**. Une struct `Config` remplie dans `main` à partir des flags
([Ch. 48](48-processus-signaux-cli.md)) et de l'environnement, puis passée aux
constructeurs. Aucun paquet profond ne lit `os.Getenv` de lui-même — sinon il
devient impossible à configurer autrement, donc à tester.

```go
type Config struct {
	Addr string
	DSN  string
}

func main() {
	var cfg Config
	flag.StringVar(&cfg.Addr, "addr", ":8080", "adresse d'écoute")
	flag.StringVar(&cfg.DSN, "dsn", "notes.db", "source de données")
	flag.Parse()
	// ... cfg est ensuite injectée dans les constructeurs ...
}
```

## Et les « architectures » avec un grand A ?

Vous croiserez les termes *hexagonal*, *ports & adapters*, *clean architecture*.
Traduits en Go, ils disent tous la même chose que ce chapitre : **le cœur métier
au centre, ignorant de la technique ; les adaptateurs (base, HTTP, fichiers) en
périphérie ; les dépendances qui pointent vers le centre via des interfaces.**

Adoptez l'**idée** (dépendances dirigées vers le domaine), pas le **cérémonial**
(une interface pour chaque struct, des dossiers `ports/` et `adapters/` dès le
premier jour). En Go, l'idiomatique est de rester léger et d'ajouter une
frontière quand un besoin réel — un test difficile, une seconde implémentation —
la justifie.

## ⚠️ Pièges

- **Paquets fourre-tout** (`utils`, `common`, `helpers`) : sans thème, ils
  aspirent tout et deviennent des nœuds de cycles. Nommez par le domaine fourni.
- **Interfaces définies côté producteur et trop grosses** : elles forcent chaque
  consommateur à dépendre de méthodes qu'il n'utilise pas et compliquent les
  fakes. Interface **côté consommateur**, la plus petite possible.
- **État global mutable** (variables de paquet, singletons) : invisible dans les
  signatures, il rend les tests non déterministes et le câblage implicite.
  Injectez plutôt.
- **Sur-abstraction prématurée** : une interface avec une seule implémentation,
  jamais simulée, n'apporte qu'une indirection. Introduisez l'interface **quand**
  le second usage (ou le test) arrive, pas « au cas où ».
- **Cycles d'import** : signal d'un mauvais découpage. Extrayez un paquet feuille
  ou inversez la dépendance par interface, plutôt que de tricher.
- **`main` qui contient la logique métier** : le point d'entrée câble et lance ;
  la logique testable vit dans `internal/`.

## ⚡ Coût & compromis

- **L'indirection par interface a un prix.** Un appel via interface est un appel
  dynamique : il empêche l'*inlining* et coûte un peu plus qu'un appel direct
  (🔁 [Ch. 33](33-interfaces-profondeur.md), [Ch. 39](39-compilation-inlining-pgo.md)).
  Négligeable pour de la logique applicative ; à surveiller seulement dans une
  boucle chaude où le même type concret est appelé des millions de fois.
- **Lisibilité contre flexibilité.** Chaque frontière (paquet, interface) ajoute
  un point d'indirection à suivre pour comprendre le flux. Une frontière ne se
  paie que si elle vous rend un service concret : testabilité, seconde
  implémentation, isolation d'un choix technique. Sinon, elle coûte sans rendre.
- **Le bon moment.** Refactorer un paquet plat vers plusieurs paquets est
  mécanique et sûr (le compilateur guide). Défaire une sur-architecture est plus
  pénible. En cas de doute, **attendez** que la douleur soit réelle.

## 🧪 À tester soi-même

Dans [`code/ch54-architecture/`](../code/ch54-architecture/) :

```bash
cd code && go test ./ch54-architecture/...
```

Exercice : ajoutez une seconde implémentation de `NoteStore` (par exemple un
store qui écrit dans un fichier JSON, 🔁 [Ch. 50](50-fichiers-fs.md)) et vérifiez
que **ni `service` ni ses tests** ne changent — seule la ligne de câblage dans
`main` bascule d'une implémentation à l'autre.

---

## 📌 À retenir

- **Commencez plat.** Un paquet, plusieurs fichiers. On découpe pour tester,
  réutiliser, cacher ou clarifier — jamais « au cas où ».
- **Layout** : `cmd/<app>/main.go` pour le câblage, `internal/` pour le code
  privé (frontière imposée par le compilateur), `pkg/` rarement utile.
- **Injection par constructeur** : les dépendances (store, logger, config)
  remontent vers `main`, le *composition root* ; le cœur ne fabrique rien.
- **Interfaces côté consommateur, petites** : le cœur définit le contrat minimal
  dont il a besoin ; la périphérie fournit un type concret qui le satisfait
  implicitement → cœur testable avec des fakes.
- **Dépendances vers le centre** : `domain` est une feuille sans dépendance ;
  extraire un paquet feuille ou inverser par interface casse les cycles.
- **`context` et erreurs enveloppées (`%w`)** circulent à travers les couches ;
  on logge **ou** on renvoie, pas les deux.

## 🔁 Pour aller plus loin

- [Ch. 12 — Packages & modules](12-packages-modules.md) : `internal/`, chemins
  d'import, visibilité.
- [Ch. 9 — Interfaces](09-interfaces.md) : typage structurel, « accepter des
  interfaces, renvoyer des structs ».
- [Ch. 10 — Erreurs](10-erreurs.md) et [Ch. 22 — `context`](22-context.md) : ce
  qui circule à travers les couches.
- [Ch. 13 — Tests](13-tests-outillage.md) : fakes, tests par paquet.
- [Projet 2 — API REST](../projets/2-api-rest/) : ce découpage appliqué à une
  vraie API (`domain`, `service`, `store`, `httpapi`).
- [Projet 4 — Bibliothèque générique](../projets/4-lib-generique/) : concevoir
  une API publique réutilisable.
