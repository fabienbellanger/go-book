# 14 — `switch` & sélection de cas

> **Objectif** — Exploiter toute la puissance de `switch` : switch d'expression, **sans
> condition**, `fallthrough`, cas multiples, et comprendre **ce que le compilateur génère**
> (cascade, recherche binaire, _jump table_).
>
> **Prérequis** — [Ch. 4 — Flux de contrôle](04-flux-controle.md), [Ch. 9 — Interfaces](09-interfaces.md) (type switch)

---

## Introduction

Le `switch` de Go est plus **sûr** et plus **expressif** que celui de C. Trois différences
fondamentales :

- **pas de `fallthrough` implicite** : chaque cas se termine **automatiquement** (pas de `break`
  oublié qui cause un bug) ;
- les **cas peuvent être des expressions** non constantes, pas seulement des constantes ;
- il remplace élégamment une **cascade de `if/else if`**.

Le [Ch. 4](04-flux-controle.md) l'a introduit ; ce chapitre en fait le tour complet, jusqu'au code
machine généré. L'exemple est dans [`code/ch14-switch/`](../code/ch14-switch/).

---

## Switch d'expression

On compare une valeur à une série de cas. Le **premier cas qui correspond** gagne (évaluation de
**haut en bas**) ; le corps exécuté, on **sort** du switch — pas de `break` nécessaire.

```go
switch day {
case "samedi", "dimanche": // PLUSIEURS valeurs par cas
	return "week-end"
case "vendredi":
	return "presque le week-end"
default: // optionnel, n'importe où, exécuté si aucun cas ne correspond
	return "semaine"
}
```

> 💡 Plusieurs valeurs dans un même cas se séparent par des **virgules** — c'est l'équivalent d'un
> « OU ». Pas besoin d'empiler des cas vides comme en C.

## Switch sans condition (_tagless_)

Un `switch` **sans expression** équivaut à `switch true` : chaque cas est une **condition
booléenne**, évaluée dans l'ordre. C'est la forme idiomatique pour des **intervalles** ou des
conditions composées — plus lisible qu'une cascade de `if/else if`.

```go
func grade(score int) string {
	switch { // pas d'expression : on teste des conditions
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	default:
		return "F"
	}
}
```

## Switch avec instruction d'initialisation

Comme `if`, `switch` accepte un **`init;`** dont la portée est limitée au switch — pratique pour
ne pas polluer l'extérieur :

```go
switch n := len(name); { // n n'existe que dans ce switch
case n == 0:
	return "vide"
case n > 64:
	return "trop long"
default:
	return "ok"
}
```

## `fallthrough` : enchaîner explicitement

Quand on **veut** tomber dans le cas suivant, on l'écrit : **`fallthrough`**. Il transfère
l'exécution au **corps du cas suivant** — **sans réévaluer sa condition**.

```go
// Chaque rôle hérite des droits des rôles inférieurs grâce à fallthrough.
func capabilities(role string) []string {
	var caps []string
	switch role {
	case "admin":
		caps = append(caps, "delete")
		fallthrough
	case "editor":
		caps = append(caps, "write")
		fallthrough
	case "viewer":
		caps = append(caps, "read")
	}
	return caps
}
// admin  -> [delete write read]
// editor -> [write read]
// viewer -> [read]
```

> ⚠️ `fallthrough` doit être la **dernière instruction** d'un cas ; il est **interdit** dans le
> **dernier** cas et dans un **type switch**. Surtout : il **n'évalue pas** la condition du cas
> suivant — il y saute **inconditionnellement**. À utiliser avec parcimonie ; il est rare.

## Type switch (rappel du Ch. 9)

Le **type switch** branche sur le **type dynamique** d'une interface ([Ch. 9](09-interfaces.md)).
Point avancé : quand un cas liste **plusieurs types**, la variable garde le type de l'**interface**
(on ne peut pas utiliser d'opération spécifique à l'un des types) :

```go
func describe(v any) string {
	switch x := v.(type) {
	case int, int64: // plusieurs types -> x est de type `any` ici
		return fmt.Sprintf("entier : %v", x)
	case string: // un seul type -> x est un string, len(x) est permis
		return fmt.Sprintf("texte de %d octets", len(x))
	default:
		return fmt.Sprintf("autre : %T", x)
	}
}
```

> 💡 Mettez les cas **concrets** avant les cas **interface** : le premier qui correspond gagne, et
> un type concret peut satisfaire plusieurs interfaces ([Ch. 9](09-interfaces.md)).

## Ce que le compilateur génère

Un `switch` n'est **pas** forcément une cascade de `if`. Selon le **nombre**, le **type** et la
**densité** des cas, le compilateur choisit la stratégie la plus rapide :

```
   peu de cas            entiers DENSES (0,1,2,…)        beaucoup de chaînes
   ----------            -----------------------         -------------------
   comparaisons          JUMP TABLE                      regroupe par LONGUEUR
   en cascade            1 borne + 1 saut indirect       puis recherche BINAIRE
   (linéaire)            -> O(1)                          -> O(log n)

   levelFromInt(n) compilé (arm64) :
       CMP  $7, R0        ; n <= 7 ?            (une seule borne)
       JMP  (R27)         ; saute droit au bon cas via une table
```

Vérifié sur go1.26.4 : un `switch` entier dense ne fait **qu'une** comparaison (la borne) suivie
d'un **saut indirect**, là où un `if/else if` ferait jusqu'à 8 comparaisons. Inutile donc de
« déplier » un switch à la main pour la performance. Détails du backend au
[Ch. 39](39-compilation-inlining-pgo.md).

---

## 🆕 Go 1.2x

- **1.19** — le compilateur génère des **_jump tables_** pour les `switch` d'entiers et de chaînes
  suffisamment denses : la sélection devient O(1) au lieu d'un balayage linéaire.
- `switch` n'a pas évolué syntaxiquement depuis Go 1 (stabilité de la « Go 1 promise ») ; son
  cousin pour les canaux, **`select`**, est traité au [Ch. 20](20-channels-select.md).

## ⚠️ Pièges

- **Attendre le `fallthrough` du C** : en Go, chaque cas **s'arrête** seul. Le comportement par
  défaut est l'inverse de C.
- **`fallthrough` qui saute la condition suivante** : il exécute le corps du cas suivant **sans le
  tester**. Source de surprises ; préférez souvent une condition explicite.
- **Cas constant en double** (`case 1: … case 1:`) → **erreur de compilation**
  (`duplicate case`). Pratique : le compilateur attrape la faute.
- **Pas de vérification d'exhaustivité** sur un type énuméré (`iota`, [Ch. 3](03-variables-constantes-types.md)) :
  ajouter une constante ne provoque **aucune** erreur si vous oubliez son cas. Mettez un `default`
  qui panique sur l'inattendu, ou utilisez un linter d'exhaustivité.
- **Oublier que `default` peut être n'importe où** : sa position n'a pas d'importance, il n'est
  choisi que si **aucun** cas ne correspond.

## ⚡ Performance

- Un `switch` dense d'entiers = **jump table** (O(1)) ; un `switch` de chaînes = regroupement par
  longueur + recherche binaire (O(log n)). Le compilateur fait mieux qu'un `if/else if` manuel.
- **`switch` vs `map`** pour un petit ensemble fixe : le `switch` gagne (pas de hachage, pas
  d'allocation). Mesure indicative (7 accès, chaînes) :

```
   BenchmarkSwitch    20.8 ns/op    0 allocs/op
   BenchmarkMap       92.6 ns/op    0 allocs/op   (~4x plus lent : hachage)
```

- Préférez une **`map`** quand l'ensemble est **grand**, **dynamique** (construit à l'exécution),
  ou quand les clés ne sont pas connues à la compilation. Le `switch` est pour les cas **fixes**
  connus à l'écriture ([Ch. 39](39-compilation-inlining-pgo.md)).

## 🧪 À tester soi-même

```bash
cd code
go run ./ch14-switch
go test ./ch14-switch/...
go test -bench=. -benchmem ./ch14-switch/...
# Observer le code généré (jump table). -o /dev/null évite la collision entre le
# binaire et le dossier du même nom :
go build -gcflags='-S' -o /dev/null ./ch14-switch 2>&1 | grep -A12 'levelFromInt'
```

À essayer :

1. Transformez `grade` (tagless) en cascade de `if/else if` : le code est-il plus clair ?
2. Ajoutez un `case 1:` en double dans un switch entier et observez l'erreur de compilation.
3. Comparez `BenchmarkSwitch` et `BenchmarkMap` sur **votre** machine.

---

## 📌 À retenir

- Pas de **`fallthrough` implicite** : chaque cas s'arrête seul ; plusieurs valeurs par cas se
  séparent par des virgules.
- Le `switch` **sans condition** (= `switch true`) remplace une cascade de `if/else if` ; le
  `switch init;` limite la portée d'une variable.
- **`fallthrough`** enchaîne explicitement (sans tester la condition suivante) ; rare, interdit en
  dernier cas et en type switch.
- Le compilateur génère du code **efficace** (jump table O(1), recherche binaire) : ne dépliez pas
  un switch « pour la perf ».
- `switch` pour un ensemble **fixe** connu à la compilation ; `map` pour un ensemble **grand ou
  dynamique**.

## 🔁 Pour aller plus loin

- [Ch. 9 — Interfaces](09-interfaces.md) : le **type switch** en détail (assertions, `(type)`).
- [Ch. 20 — Channels & `select`](20-channels-select.md) : `select`, le « switch » des canaux.
- [Ch. 39 — Compilation & inlining](39-compilation-inlining-pgo.md) : génération des jump tables,
  inspection avec `-gcflags=-S`.
- [Ch. 3 — Constantes](03-variables-constantes-types.md) : `iota` et les types énumérés.
