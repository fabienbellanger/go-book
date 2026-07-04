# Annexe K — Venir de Python, JavaScript/TypeScript, Rust ou PHP

> **Objectif** — Transposer vos réflexes d'un autre langage vers Go sans trébucher
> sur les **faux amis** : la syntaxe qui ressemble mais ne fait pas la même chose,
> et les mécanismes que Go remplace par autre chose (ou supprime purement et
> simplement). Une section par langage source, à lire en diagonale selon d'où
> vous venez.

---

## Ce qui surprend tout le monde

Go fait des choix radicaux de **simplicité** et d'**explicite**. Quatre d'entre eux
déroutent, quelle que soit votre origine :

- **Pas d'exceptions.** Une fonction qui peut échouer renvoie une `error` que
  l'appelant doit traiter sur place. Le fameux `if err != nil { return err }` est
  la contrepartie assumée d'un flot de contrôle sans surprise (🔁 [Ch. 10](../chapitres/10-erreurs.md)).
- **Pas de classes ni d'héritage.** On a des `struct` (données), des **méthodes**
  (attachées à un type), de la **composition** (embedding) et des **interfaces
  implicites** (🔁 [Ch. 8](../chapitres/08-structs-methodes.md), [Ch. 9](../chapitres/09-interfaces.md)).
- **La casse remplace `public`/`private`.** Un identifiant qui commence par une
  **majuscule** est exporté hors de son paquet ; minuscule = privé. Il n'existe
  aucun mot-clé de visibilité (🔁 [Ch. 12](../chapitres/12-packages-modules.md)).
- **Le style ne se discute pas.** `gofmt` impose le formatage (tabulations,
  alignement, position des accolades). Aucune configuration, aucun débat.

Et une chose qu'on cherche en vain au début : **il n'y a pas d'opérateur
ternaire** (`a ? b : c`), pas de surcharge d'opérateurs, pas de valeurs par
défaut d'arguments, pas de paramètres nommés. Ces absences sont voulues.

---

## Venir de Python

| Concept | Python | Go | ⚠️ Faux ami / piège |
|---------|--------|----|----------------------|
| Déclaration | `x = 3` (dynamique) | `x := 3` (statique, inféré) / `var x int` | `:=` déclare **et** affecte ; réservé à l'intérieur d'une fonction. `=` seul suppose une variable déjà déclarée |
| Types | dynamique, *duck typing* à l'exécution | statique, vérifié à la compilation | le *duck typing* de Go (interfaces) est **vérifié au compile-time**, pas au runtime |
| Absence de valeur | `None` | valeur **zéro** (`0`, `""`, `nil`, struct à champs nuls) | pas de `None` universel : un `int` non initialisé vaut `0`, jamais `nil` (🔁 [Ch. 3](../chapitres/03-variables-constantes-types.md)) |
| Erreurs | `try` / `except` / `raise` | valeur `error` renvoyée et testée | pas d'exceptions. `panic` existe mais n'est **pas** un `raise` du quotidien (🔁 [Ch. 17](../chapitres/17-panic-recover.md)) |
| Classes | `class C:` + héritage + `self` | `struct` + méthodes + composition, récepteur nommé (pas `self`) | pas d'héritage : on **compose** (🔁 [Ch. 8](../chapitres/08-structs-methodes.md)) |
| `self` | premier paramètre explicite | récepteur : `func (c Counter) Inc()` | le récepteur est court (`c`), constant, et **valeur ou pointeur** change tout (copie vs mutation) |
| Listes / dicts | `list`, `dict` | `slice` (`[]T`), `map[K]V` | une `map` Go n'a **pas d'ordre** garanti ; lire une clé absente renvoie la **valeur zéro**, pas une `KeyError` (🔁 [Ch. 7](../chapitres/07-maps-strings.md)) |
| Compréhensions | `[f(x) for x in xs]` | boucle `for` explicite | pas de *list comprehension* ; la boucle est l'idiome (🔁 [Ch. 6](../chapitres/06-arrays-slices.md)) |
| Ternaire | `a if cond else b` | `if`/`else` classique | **aucun** ternaire ; on écrit le `if` |
| Décorateurs | `@decorator` | fonction qui prend/renvoie une fonction | pas de sucre `@` ; les closures suffisent (🔁 [Ch. 15](../chapitres/15-closures.md)) |
| Ressources | `with open(...) as f:` | `defer f.Close()` | `defer` s'exécute à la **sortie de la fonction**, pas à la fin d'un bloc (🔁 [Ch. 16](../chapitres/16-defer.md)) |
| Concurrence | `threading` (GIL), `asyncio` | goroutines + channels | pas de GIL : les goroutines tournent en **vrai parallèle** ; la synchronisation est à votre charge (🔁 [Ch. 19](../chapitres/19-goroutines.md), [Ch. 20](../chapitres/20-channels-select.md)) |
| Paquets | `pip`, `venv`, `import` | modules Go, `go.mod`, `import` | versions gérées par le module ; pas d'environnement virtuel (🔁 [Ch. 12](../chapitres/12-packages-modules.md)) |
| Généricité | tout est dynamique | paramètres de type `[T any]` | générique **statique**, contraint (🔁 [Ch. 11](../chapitres/11-genericite.md)) |
| Visibilité | `_prefixe` par convention | Majuscule = exporté, minuscule = privé | c'est **imposé** par le compilateur, pas une convention |

**Le réflexe à désapprendre :**

- Ne cherchez pas à envelopper chaque appel dans un `try/except` : testez la valeur `error` renvoyée.
- Ne comptez pas sur `None` : pensez « valeur zéro ». Une `map`, un `slice`, un pointeur peuvent être `nil`, mais pas un `int` ou une `string`.
- Oubliez le typage dynamique : le compilateur exige des types cohérents, et c'est un filet, pas une contrainte.

---

## Venir de JavaScript / TypeScript

| Concept | JS / TS | Go | ⚠️ Faux ami / piège |
|---------|---------|----|----------------------|
| Déclaration | `let` / `const` | `:=`, `var`, `const` | `const` Go ne vaut que pour des **constantes de compilation** (nombres, chaînes, booléens), pas pour « une variable non réassignée » |
| Absence | `null` **et** `undefined` | `nil` (pour pointeurs, maps, slices, channels, interfaces, fonctions) | un `int`/`string` ne peut **pas** être `nil` ; il a une valeur zéro. Pas de distinction null/undefined |
| Égalité | `==` (coercition) vs `===` | `==` (typé, pas de coercition) | jamais de coercition implicite ; `==` entre types différents ne **compile pas** |
| Erreurs | `throw` / `try` / `catch` | valeur `error` | pas de `throw`. `panic`/`recover` ≠ try/catch du quotidien (🔁 [Ch. 17](../chapitres/17-panic-recover.md)) |
| Objets | objets littéraux, classes, prototypes | `struct` typées + interfaces | pas de propriétés dynamiques : les champs sont **fixés** à la compilation |
| Tableaux | `Array` (dynamique) | `array` (taille fixe) **vs** `slice` (dynamique) | le mot « array » désigne en Go un tableau de **taille fixe** ; le quotidien, c'est le `slice` (🔁 [Ch. 6](../chapitres/06-arrays-slices.md)) |
| Objets clé/valeur | `{}` / `Map` | `map[K]V` | clé absente → **valeur zéro**, pas `undefined` ; pas d'ordre d'itération (🔁 [Ch. 7](../chapitres/07-maps-strings.md)) |
| `this` | dépend de l'appel, `bind` | récepteur explicite, lié statiquement | pas de surprise de `this` : le récepteur est un paramètre |
| Async | `Promise`, `async`/`await` | goroutines + channels (synchrones) | Go est **bloquant par conception** : une goroutine bloquée n'immobilise pas le programme, l'ordonnanceur bascule (🔁 [Ch. 19](../chapitres/19-goroutines.md), [Ch. 28](../chapitres/28-ordonnanceur-gmp.md)) |
| Callbacks | `arr.map(...)`, `filter` | boucle `for` explicite | pas de `map`/`filter` intégrés sur les slices ; on écrit la boucle ou on utilise la généricité |
| Ternaire | `cond ? a : b` | `if`/`else` | **aucun** ternaire |
| Modules | `npm`, `import`/`require` | modules Go, `go.mod` | pas de `node_modules` : le cache modules est global et versionné (🔁 [Ch. 12](../chapitres/12-packages-modules.md)) |
| Types (TS) | interfaces **structurelles**, optionnelles | interfaces **structurelles**, vérifiées, implicites | proche de TS ! Mais l'implémentation est **implicite** : aucun `implements` (🔁 [Ch. 9](../chapitres/09-interfaces.md)) |
| Généricité (TS) | `<T>` effacé au runtime | `[T any]`, conservé et contraint | pas d'*type erasure* : les contraintes sont réelles (🔁 [Ch. 11](../chapitres/11-genericite.md)) |
| Visibilité | `export`, `#private` | Majuscule = exporté | pas de mot-clé ; la **casse** décide |

**Le réflexe à désapprendre :**

- N'attendez pas de valeurs `undefined` : une clé de map absente renvoie la valeur zéro, indiscernable d'une valeur zéro stockée (testez avec la forme `v, ok := m[k]`).
- Ne cherchez pas `async`/`await` : lancez une goroutine et communiquez par channel. Le code « asynchrone » se lit comme du code séquentiel.
- Méfiez-vous de `array` vs `slice` : 99 % du temps vous voulez un `slice`.

---

## Venir de Rust

| Concept | Rust | Go | ⚠️ Faux ami / piège |
|---------|------|----|----------------------|
| Déclaration | `let` / `let mut` | `:=` / `var` | **tout est mutable** en Go ; pas de `mut`, pas d'immutabilité par défaut |
| Absence / erreurs | `Option<T>`, `Result<T, E>`, `?` | `nil` + valeur zéro ; `(T, error)` | pas de `Option`/`Result` ni de `?` : on renvoie **deux valeurs** et on teste `err` à la main (🔁 [Ch. 10](../chapitres/10-erreurs.md)) |
| Mémoire | *ownership*, *borrow checker*, durées de vie | *garbage collector* | pas de propriété ni d'emprunt : le GC gère la mémoire (🔁 [Ch. 27](../chapitres/27-garbage-collector.md)) |
| Références | `&T`, `&mut T`, règles d'aliasing | `*T` (pointeur), pas de règle d'emprunt | un `*T` Go peut être copié/aliasé librement ; les *data races* sont **votre** responsabilité (🔁 [Annexe H](H-concurrence-sure.md)) |
| Traits | `trait` + `impl ... for` explicite | interfaces **implicites** | aucun `impl` : un type satisfait une interface s'il en a les méthodes, sans le déclarer (🔁 [Ch. 9](../chapitres/09-interfaces.md)) |
| Enums / pattern | `enum` + `match` exhaustif | pas d'enums algébriques ; `switch` non exhaustif | pas de sommes de types ni de `match` exhaustif vérifié ; on émule avec des interfaces + `type switch` (🔁 [Ch. 14](../chapitres/14-switch.md), [Ch. 33](../chapitres/33-interfaces-profondeur.md)) |
| Généricité | `<T: Trait>`, monomorphisation | `[T Constraint]` | contraintes plus simples ; pas de spécialisation ni de trait bounds riches (🔁 [Ch. 11](../chapitres/11-genericite.md)) |
| Concurrence | `async`/`.await`, `tokio`, `Send`/`Sync` | goroutines + channels, runtime intégré | ordonnanceur **intégré au langage**, pas une bibliothèque ; pas de marqueurs `Send`/`Sync` — la sûreté n'est **pas** garantie par le compilateur (🔁 [Ch. 19](../chapitres/19-goroutines.md)) |
| Ressources | *Drop* (RAII, fin de portée) | `defer` (fin de **fonction**) | `defer` ne se déclenche pas en fin de bloc mais en fin de fonction (🔁 [Ch. 16](../chapitres/16-defer.md)) |
| Macros | `macro_rules!`, dérive | `go generate` + génération de code | pas de macros ; on génère du code au build (🔁 [Projet 6](../projets/6-codegen/)) |
| Paquets | `cargo`, `crates` | `go`, modules | proche dans l'esprit ; `go.mod` ≈ `Cargo.toml` (🔁 [Ch. 12](../chapitres/12-packages-modules.md)) |
| Visibilité | `pub` | Majuscule = exporté | pas de `pub` ; la casse décide |

**Le réflexe à désapprendre :**

- Lâchez le *borrow checker* : rien ne vous empêche de partager un pointeur entre goroutines — donc rien ne vous protège d'une *data race*. Utilisez channels/mutex et `go test -race` (🔁 [Ch. 23](../chapitres/23-patterns-concurrence.md)).
- N'attendez pas de `match` exhaustif : un `switch` sur un type incomplet compile sans broncher. Prévoyez le `default`.
- Renoncez à `Option`/`Result` : le couple `(valeur, error)` et la valeur zéro les remplacent, avec moins de garanties mais moins de cérémonie.

---

## Venir de PHP

| Concept | PHP | Go | ⚠️ Faux ami / piège |
|---------|-----|----|----------------------|
| Typage | dynamique (types stricts optionnels) | statique, obligatoire | tout est typé et vérifié à la compilation |
| Variables | `$var` | `var` / `:=`, sans `$` | pas de sigil `$` ; le nom seul suffit |
| Absence | `null` | `nil` (pointeurs, maps, slices…) + valeur zéro | un `int`/`string` a une valeur zéro, jamais `null` |
| Tableaux | `array` (liste **et** dictionnaire) | `slice` **ou** `map`, distincts | le `array` fourre-tout de PHP se scinde en deux : `[]T` **ou** `map[K]V` (🔁 [Ch. 6](../chapitres/06-arrays-slices.md), [Ch. 7](../chapitres/07-maps-strings.md)) |
| Erreurs | `Exception` + `try`/`catch` | valeur `error` | pas d'exceptions dans le flot normal (🔁 [Ch. 10](../chapitres/10-erreurs.md)) |
| Classes | `class`, héritage, `interface` explicite | `struct` + méthodes + interfaces implicites | pas d'héritage ni de `implements` ; composition + interfaces (🔁 [Ch. 8](../chapitres/08-structs-methodes.md), [Ch. 9](../chapitres/09-interfaces.md)) |
| `$this` | `$this` | récepteur nommé | le récepteur est un paramètre explicite, souvent une seule lettre |
| Concaténation | `.` (point) | `+` sur les `string`, ou `strings.Builder` | le `.` de PHP devient `+` ; pour concaténer en boucle, `strings.Builder` (🔁 [Ch. 31](../chapitres/31-strings-profondeur.md)) |
| Exécution | requête → script rejoué, état perdu | processus **long** en mémoire | un serveur Go est un processus persistant : l'état vit entre les requêtes (attention aux données partagées, 🔁 [Ch. 45](../chapitres/45-net-http.md)) |
| Concurrence | rare (par requête) | goroutines natives | un serveur Go gère la concurrence **dans un seul processus** (🔁 [Ch. 19](../chapitres/19-goroutines.md)) |
| Paquets | `composer`, `namespace`, autoload | modules Go, `import` | pas d'autoload : chaque import est explicite (🔁 [Ch. 12](../chapitres/12-packages-modules.md)) |
| Interpolation | `"Bonjour $name"` | `fmt.Sprintf("Bonjour %s", name)` | pas d'interpolation dans les chaînes ; `fmt` ou concaténation (🔁 [Annexe I](I-formatage-fmt.md)) |
| Visibilité | `public`/`private`/`protected` | Majuscule = exporté | pas de mot-clé ni de `protected` ; la casse décide |

**Le réflexe à désapprendre :**

- Pensez « processus long » : un binaire Go n'est pas rejoué à chaque requête. Les variables globales et caches survivent — et se partagent entre goroutines (danger de *data race*).
- Distinguez `slice` et `map` : le `array` universel de PHP n'existe pas.
- Remplacez les exceptions par le test de `err`, et l'interpolation `$` par `fmt.Sprintf` ou `+`.

---

## Synthèse : les faux amis les plus coûteux

| Piège | Ce que vous croyez | La réalité Go |
|-------|--------------------|---------------|
| Clé de map absente | erreur / `None` / `undefined` | renvoie la **valeur zéro** sans erreur — utilisez `v, ok := m[k]` |
| `array` | tableau dynamique | tableau de **taille fixe** ; le dynamique, c'est `slice` |
| `defer` | fin de bloc (RAII, `with`, `finally`) | fin de **fonction** |
| Absence de valeur | `null`/`None`/`undefined` universel | `nil` seulement pour pointeurs, maps, slices, channels, interfaces, fonctions ; sinon **valeur zéro** |
| Concurrence sûre | garantie par le langage (GIL, `Send`/`Sync`) | **non garantie** : *data races* possibles, à traquer avec `go test -race` (🔁 [Annexe H](H-concurrence-sure.md)) |
| Interface | à déclarer (`implements`, `impl`) | **implicite** : satisfaite par la seule présence des méthodes |
| Erreur | exception qui remonte la pile | valeur `error` à tester **à chaque appel** |
| Comparaison `==` | coercition de types (JS) | typée et stricte ; types différents ne compilent pas |
| Visibilité | mot-clé `public`/`private` | **casse** de la première lettre |
| Ternaire / valeurs par défaut / surcharge | disponibles | **absents**, volontairement |

## 🔁 Pour aller plus loin

- [Annexe F — Idiomes & style](F-idiomes-style.md) : écrire du Go qui se lit comme du Go.
- [Ch. 9 — Interfaces](../chapitres/09-interfaces.md) et [Ch. 8 — Structs & méthodes](../chapitres/08-structs-methodes.md) : le remplacement des classes.
- [Ch. 10 — Gestion des erreurs](../chapitres/10-erreurs.md) : la vie sans exceptions.
- [Annexe H — Concurrence sûre](H-concurrence-sure.md) : ce que le compilateur ne garantit pas.
