# 32 — Maps

> **Objectif** — Ouvrir la table de hachage de Go : les **Swiss Tables** (groupes de 8 slots, mot de
> contrôle, recherche parallèle h2), le **facteur de charge**, la **croissance incrémentale**, la
> **randomisation** de l'itération et sa raison, la **non-sûreté concurrente**, et le choix
> `sync.Map` vs `map`+`Mutex`.
>
> **Prérequis** — [Ch. 7](07-maps-strings.md), [Ch. 21](21-synchronisation.md)

---

## Introduction

Une `map` Go est une **table de hachage** : insertion, lecture et suppression en **O(1) amorti**. Le
[Ch. 7](07-maps-strings.md) a montré l'usage ; ce chapitre ouvre le capot. Depuis **Go 1.24**,
l'implémentation a changé en profondeur — les **Swiss Tables** — pour gagner en mémoire et en vitesse.
Comprendre leur structure explique la randomisation d'itération, l'intérêt de préallouer, et pourquoi un
accès concurrent **crashe**. Code dans [`code/ch32-maps-hachage/`](../code/ch32-maps-hachage/).

---

## La structure : groupes de 8 slots

Une Swiss Table range les entrées dans des **groupes** de **8 slots**. Chaque groupe porte un **mot de
contrôle** de 8 octets — **un octet par slot** — qui résume l'état du slot : **vide**, **supprimé**
(tombstone), ou les **7 bits bas du hash** (`h2`) si le slot est plein.

```
  Un GROUPE = mot de contrôle (8 octets) + 8 slots (clé/valeur)

  control word (64 bits)            slots
  +----+----+----+----+----+        [0] k,v
  | h2 | E  | h2 | T  | h2 | ...     [1] vide
  +----+----+----+----+----+        [2] k,v   ...
    E = vide   T = supprimé   h2 = 7 bits bas du hash de la clé
```

## La recherche : `h2` comparé en parallèle

Le hash d'une clé (64 bits) se scinde en deux : **`h1`** (les **57 bits hauts**) choisit la **table** et
le **groupe** de départ ; **`h2`** (les **7 bits bas** restants) sert à filtrer les slots du groupe.

```
  hash(key) = [   h1  (bits hauts)   |  h2 (7 bits)  ]
                       |                     |
                       v                     v
              choisit table + groupe   diffusé puis comparé
                                        aux 8 octets de contrôle
                                        EN UNE opération 64 bits
                                             |
                                             v
                                   masque des slots candidats
                                   -> compare la clé COMPLÈTE
```

L'astuce : comparer `h2` aux **8 octets de contrôle d'un coup** (une seule instruction sur un registre,
technique **SWAR** — pas besoin d'instructions SIMD matérielles spécifiques à une architecture) donne un
**masque** des slots candidats. On ne compare la clé entière que pour ces quelques candidats. Avec
seulement **7 bits**, `h2` ne distingue que 128 valeurs : statistiquement, **1 candidat sur 128** est une
fausse alerte (deux clés différentes qui partagent le même `h2`) — la comparaison complète de la clé
reste donc nécessaire, mais devient **rare**.

Pourquoi ce découpage bat-il un **chaînage classique** (chaque collision ajoute un nœud alloué à part,
relié par pointeur) ? Parce que le mot de contrôle d'un groupe — **8 octets contigus** — tient dans un
fragment d'une seule ligne de cache : une unique lecture mémoire suffit pour écarter la quasi-totalité
des slots, sans jamais toucher aux clés/valeurs elles-mêmes ni suivre le moindre pointeur. Les 8 entrées
du groupe sont elles aussi **contiguës**, donc la comparaison de clé qui suit reste locale. Le chaînage,
lui, force un saut mémoire (souvent un défaut de cache) par nœud de la liste — c'est ce coût-là, pas le
nombre brut de comparaisons, qui domine sur des maps trop grandes pour le cache L1/L2. Si le groupe est
plein sans correspondance, on **sonde** le groupe suivant.

## Facteur de charge & croissance incrémentale

Les Swiss Tables tournent **denses** : elles grossissent vers **~87,5 %** de remplissage (7 slots sur 8)
avant de croître — contre ~81 % pour l'ancienne map. Plus dense = moins de mémoire, meilleure localité.

Une table individuelle est plafonnée à **128 groupes, soit 1024 entrées**. Quand elle atteint ce seuil,
elle ne grossit pas sur place : elle **scinde en deux** nouvelles tables de 1024 entrées, départagées par
un bit supplémentaire emprunté à `h1`. Pour les **grandes** maps, un **annuaire** (directory) regroupe
ainsi plusieurs tables, chacune adressée par les bits hauts du hash :

```
  annuaire (directory) — route chaque clé via les bits hauts de h1
  +------------+------------+------------+------------+
  |  table 00  |  table 01  |  table 10  |  table 11  |   <- jusqu'à 1024 entrées chacune
  +------------+------------+------------+------------+
                      |
                      v  table 01 atteint 1024 entrées
              scinde en DEUX tables ; l'annuaire gagne un bit
              les 3 AUTRES tables ne sont pas touchées
```

Ne **scinder qu'une seule table à la fois** borne le coût d'une insertion qui déclenche une croissance :
au pire, copier 1024 entrées — quelle que soit la taille totale de la map, même à plusieurs millions de
clés. C'est la **croissance incrémentale** : pas de pause proportionnelle à la taille globale.
C'est pourquoi **préallouer** paie :

```go
// code/ch32-maps-hachage/maps.go
func WordCount(words []string) map[string]int {
	counts := make(map[string]int, len(words)) // dimensionne d'emblée -> pas de croissance
	for _, w := range words {
		counts[w]++
	}
	return counts
}
```

Mesuré (`-benchmem`, 1000 insertions) :

| Variante           | ns/op      | B/op   | allocs/op |
| ------------------ | ---------- | ------ | --------- |
| sans préallocation | **45 285** | 74 264 | **20**    |
| `make(map, 1000)`  | **12 054** | 36 944 | **5**     |

`make(map, n)` : **3,8× plus rapide**, **2× moins de mémoire**, **20 → 5** allocations.

## L'itération est randomisée — exprès

Chaque `range` démarre à un **groupe et un offset aléatoires**. Deux parcours de la **même** map donnent
donc des ordres **différents** :

```go
// code/ch32-maps-hachage/maps.go
orders := IterationOrders(m, 2) // 2 parcours de la même map
```

```
$ go run ./ch32-maps-hachage
parcours 1 : 6,10,11,1,2,3,4,5,7,8,9,0,
parcours 2 : 7,8,9,0,6,10,11,1,2,3,4,5,
différents ? true
```

Pourquoi ? Pour **empêcher** tout code de dépendre d'un ordre d'insertion (qui n'a aucune raison d'être
stable) et comme **durcissement** anti-DoS par collisions de hash. 📌 Une map est **non ordonnée** :
pour un ordre déterministe, extrayez les clés et triez (`slices.Sorted(maps.Keys(m))`,
[Ch. 18](18-iterateurs.md)).

## Non-sûreté concurrente

Une map **n'est pas** sûre en accès concurrent. Le runtime **détecte** les écritures simultanées et
**tue le programme** : `fatal error: concurrent map writes`. Ce n'est **pas** une panique — `recover`
n'y peut rien. La parade : un **verrou**.

```go
// code/ch32-maps-hachage/maps.go
type SafeCounter struct {
	mu sync.Mutex
	m  map[string]int
}

func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	c.m[key]++
	c.mu.Unlock()
}
```

Testé avec **100 × 100 incréments concurrents** : total exact `10000`, et `go test -race` **propre**.

### `sync.Map` vs `map` + `Mutex`

`sync.Map` ([Ch. 21](21-synchronisation.md)) est une map concurrente prête à l'emploi, mais **spécialisée** :
elle n'excelle que dans deux cas — clés **écrites une fois puis surtout lues**, ou jeux de clés
**disjoints** entre goroutines. Dans le cas général, **`map` + `Mutex`** (ou `RWMutex`) est **plus simple
et souvent plus rapide**. Mesurez avant de choisir `sync.Map`.

---

## 🆕 Go 1.2x

- **1.24** — l'implémentation des maps passe aux **Swiss Tables** : groupes de 8 slots, mot de contrôle,
  recherche `h2` parallèle (SWAR), annuaire de tables pour une croissance incrémentale. Gains de mémoire
  et de CPU **sans changement d'API** : votre code en profite gratuitement.
- **continuité** — l'**ordre d'itération reste randomisé** (ce n'est pas un effet de bord, c'est
  garanti) ; ne vous y fiez jamais.

## ⚠️ Pièges

- **Écriture concurrente** — `fatal error: concurrent map writes`, **non rattrapable**. Protégez par
  `Mutex`/`RWMutex` ou utilisez `sync.Map` (cas adaptés).
- **Dépendre de l'ordre d'itération** — il est volontairement aléatoire. Triez les clés si besoin d'ordre.
- **Prendre l'adresse d'un élément** (`&m[k]`) — **interdit** : la croissance **déplace** les entrées.
  Stockez un pointeur **comme valeur** (`map[K]*V`) si vous devez muter en place.
- **Muter un champ d'une valeur struct** — `m[k].champ = x` ne **compile pas** (valeur non adressable) ;
  réassignez la struct entière, ou utilisez `map[K]*V`.
- **Oublier `comma-ok`** — `m[k]` renvoie le **zéro** si absent ; `v, ok := m[k]` distingue absent de zéro.

## ⚡ Performance

- **Préallouez** `make(map, n)` dès que la taille est estimable : 20 → 5 allocations ci-dessus.
- **Clés coûteuses** — une clé `string` ou struct longue est **hachée et comparée** à chaque accès.
  Internez les chaînes répétées (`unique`, [Ch. 31](31-strings-profondeur.md)) ou utilisez une clé plus
  compacte.
- **Itérer puis supprimer** est sûr (`delete` pendant `range` est autorisé) ; **ajouter** pendant l'itération
  donne un résultat **indéterminé** pour les nouvelles clés.
- Les Swiss Tables améliorent la **localité de cache** : sur de grosses maps, les lectures sont sensiblement
  plus rapides qu'avant 1.24.

## 🧪 À tester soi-même

```bash
cd code
go run ./ch32-maps-hachage
go test -race ./ch32-maps-hachage/...
go test -bench=. -benchmem -run=^$ ./ch32-maps-hachage/...
```

À essayer :

1. Retirez le `Mutex` de `SafeCounter`, lancez `go test -race` : observez `concurrent map writes`.
2. Mesurez `WordCount` avec et sans le `len(words)` dans `make` (`-benchmem`).
3. Triez les clés avec `slices.Sorted(maps.Keys(m))` pour obtenir un parcours **déterministe**.

---

## 📌 À retenir

- Une map est une **table de hachage** ; depuis **1.24**, des **Swiss Tables** : groupes de **8 slots**,
  **mot de contrôle**, recherche **`h2` parallèle** (SWAR) — plus denses, plus rapides, **même API**.
- `h1` choisit table/groupe, `h2` filtre les 8 slots d'un coup ; remplissage **~87,5 %** avant croissance
  **incrémentale** (annuaire de tables).
- L'**ordre d'itération est aléatoire** par conception ; une map est **non ordonnée** — triez les clés
  pour un ordre stable.
- Accès concurrent = **`fatal error: concurrent map writes`** (non rattrapable). Protégez par `Mutex`, ou
  `sync.Map` dans ses cas d'usage.
- **Préallouez** (`make(map, n)`), évitez `&m[k]` (entrées déplacées), utilisez `map[K]*V` pour muter
  en place.

## 🔁 Pour aller plus loin

- [Ch. 7 — Maps & strings (usage)](07-maps-strings.md) : l'API de base, `comma-ok`, `delete`.
- [Ch. 21 — Synchronisation](21-synchronisation.md) : `Mutex`, `RWMutex`, `sync.Map` en détail.
- [Ch. 31 — Strings en profondeur](31-strings-profondeur.md) : `unique` pour des clés string compactes.
- [Ch. 26 — Allocation & escape](26-allocation-escape.md) : ce que coûte la croissance d'une table.
- Doc : `go doc builtin.delete` ; `go doc maps` ; `go doc sync.Map`.
