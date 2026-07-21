# AUDIT — « Comprendre et maîtriser Go 1.26 »

> Audit de complétude, cohérence et exactitude des chapitres, annexes et code d'appui.
> Date : 2026-07-21. Cible : Go 1.26 (toolchain `go1.26.5` vérifiée).

## Méthodologie

1. **Reconnaissance globale** (automatisée, dépôt entier).
2. **Validation du code** : `go build`, `go vet`, `go test`, `gofmt` sur le module `code/`.
3. **Audit de fond en 10 passes parallèles** (8 sur les chapitres, 2 sur les annexes), chacune
   croisant le Markdown avec son dossier `code/` et compilant/exécutant au besoin, pour vérifier :
   - complétude (promesses de l'intro tenues, sections standard présentes, pas de sections vides) ;
   - cohérence (renvois `Ch. NN` / `Annexe X` valides, terminologie, émojis repères) ;
   - exactitude Go 1.26 (signatures d'API, comportements, nouveautés `🆕`) — recoupée avec les
     notes de version officielles et la compilation réelle ;
   - **dérive texte ↔ code** : les blocs annotés `// code/chNN/...` doivent refléter le fichier réel ;
   - bugs dans les extraits de code (inline et fichiers).
4. **Correctifs** appliqués dans `code/`, `chapitres/` et `annexes/` (détaillés plus bas).

## Bilan général

**Le livre est dans un état remarquable.** Sur 55 chapitres + 13 annexes + ~68 dossiers de code :
build/vet/test **entièrement au vert**, aucune data race, attributions de version **cohérentes de
bout en bout**, aucun lien cassé, aucun schéma box-drawing. **20 correctifs** appliqués, tous de
gravité faible à moyenne : commentaires trompeurs, dérives mineures texte↔code, imprécisions de
version, deux exercices manquants, et deux exemples d'outillage (`benchstat`) non reproductibles.
**Aucun bug bloquant, aucune section manquante.**

## Statut de la reconnaissance globale

| Contrôle | Résultat |
| --- | --- |
| `go build ./...` (module `code/`) | ✅ propre |
| `go vet ./...` | ✅ propre |
| `go test ./...` | ✅ tous au vert |
| `gofmt -l` | ⚠️ 1 fichier non formaté → **corrigé** |
| Caractères box-drawing Unicode interdits | ✅ aucun |
| Liens Markdown internes (`.md`) | ✅ aucun cassé |
| Chemins `// code/...` référencés | ✅ tous présents (hors placeholder du gabarit) |
| Marqueurs `TODO`/`FIXME`/texte inachevé | ✅ aucun réel |
| Couverture code : chapitres 1–54 | ✅ un dossier `code/chNN` par chapitre (ch. 0 philosophique, sans code) |

## Constats transversaux (inter-chapitres)

- **Cohérence des attributions de version** : ✅ aucune contradiction. Une même fonctionnalité
  reçoit partout la même version. `min`/`max`/`clear`/`slices`/`maps`/PGO/`slog` = **1.21** ;
  variable de boucle par itération, `range` sur entier, routage `net/http` = **1.22** ;
  range-over-func/`iter`, timers GC-friendly = **1.23** ; `for b.Loop()`, `os.Root`, alias
  génériques, `omitzero`, `crypto/rand.Text` = **1.24** ; `testing/synctest`, Flight Recorder,
  `WaitGroup.Go` = **1.25**.
- **Fonctionnalités Go 1.26 vérifiées** (compilation + notes de version officielles) :
  - `new(expr)` (ch. 3) — ✅ `new(7)`, `new(1 + 2)` compilent ; confirmé par les notes 1.26.
  - `errors.AsType[E error](err) (E, bool)` (ch. 10) — ✅ compile et fonctionne.
  - Contraintes génériques **auto-référentielles** (`type Adder[A Adder[A]]`, ch. 11 / annexe C) —
    ✅ confirmé nouveau en 1.26 par les notes (« the self-reference ... was not allowed » avant).
  - **PGO** : ❌ aucune amélioration PGO listée dans les notes 1.26 → mentions retirées (ch. 39, 40).
- **Sections standard** : ✅ tous les chapitres 1–54 ont « 📌 À retenir » et « 🔁 Pour aller plus loin ».

## Correctifs appliqués (20)

### Code (`code/`)

| # | Fichier | Problème | Correctif |
| --- | --- | --- | --- |
| 1 | `annexe-L-solutions/ch09_test.go` | Non formaté (échec `make check` / CI) | `gofmt -w` |
| 2 | `ch04-controlflow/controlflow.go` | Commentaire de doc de `firstPair` : « somme des valeurs » alors que le code teste l'**égalité** | Reformulé (« dont la valeur vaut target ») |
| 3 | `ch50-fichiers-fs/main.go` | Commentaire `Sync` surévaluant la « durabilité » (le `Rename` + le répertoire parent ne sont pas synchronisés) | Reformulé : atomicité garantie ; durabilité stricte = `fsync` du répertoire |
| 4 | `ch54-architecture/service/service.go` | `NoteStore` déclare `List` mais `Service` ne l'utilise **jamais** — contredit la leçon « ni plus ni moins » | Ajout de `Service.List` (délègue au store) → les 3 méthodes déclarées sont utilisées |
| 5 | `ch54-architecture/main.go` | La démo n'exerçait pas `List` | Ajout d'un `svc.List` affichant le total |
| 6 | `annexe-L-solutions/ch05.go` + `ch05_test.go` | Exercice 2 du ch. 5 (`incVal` → `*counter`) **manquant** | Ajout `ch05IncVal`/`ch05IncPtr` + test |
| 7 | `annexe-L-solutions/ch09.go` + `ch09_test.go` | Exercice 1 du ch. 9 (method set récepteur valeur vs pointeur) **manquant** | Ajout `ch09ValErr`/`ch09PtrErr` + test |

### Chapitres & annexes

| # | Fichier | Problème | Correctif |
| --- | --- | --- | --- |
| 8 | `chapitres/18-iterateurs.md` | `TestYieldAfterStopPanics` attribué à `iterators.go` (il est dans `iterators_test.go`) | Localisation corrigée |
| 9 | `annexes/I-formatage-fmt.md` (bloc `%c %q %U`) | Dérive : libellé non échappé (6 verbes / 3 args → `%!c(MISSING)`) ; le fichier réel utilise `%%` | `%%c %%q %%U` restauré (conforme au code) |
| 10 | `annexes/I-formatage-fmt.md` (tableau) | `%6.2f` sur `3.14` affiché `[····3.14]` (largeur 8) | `[··3.14]` (largeur 6) |
| 11 | `annexes/L-solutions-exercices.md` | Exercices ch5 ex2 et ch9 ex1 non couverts | Énoncés + solutions + explications ajoutés (miroir des #6/#7) |
| 12 | `chapitres/10-erreurs.md` (×3 : intro, 🆕, ⚡) | Nouveauté 1.26 `fmt.Errorf` généralisée à tort aux erreurs **formatées** (l'exemple `%d` fait toujours 2 allocs) | Restreint aux chaînes **sans verbe de formatage** ; formaté/`%w` = 2 allocs |
| 13 | `annexes/C-nouveautes-1.21-1.26.md` | `cmp.Or` daté 1.21 | Déplacé en **1.22** (1.21 = `cmp.Compare`/`cmp.Less`) |
| 14 | `annexes/C-nouveautes-1.21-1.26.md` | Timers 1.23 « non **démarrés** » | « non **référencés** » (GC sans `Stop()`) |
| 15 | `annexes/C-nouveautes-1.21-1.26.md` | Exemple de contrainte auto-réf. imprécis | Précisé : `type Adder[A Adder[A]] interface{ Add(A) A }` |
| 16 | `annexes/C-nouveautes-1.21-1.26.md` | Renvoi modernizers `go fix` → Ch. 13 (documentés en Ch. 1) | → **Ch. 1** (aligné sur le glossaire) |
| 17 | `chapitres/39-compilation-inlining-pgo.md` | « La PGO continue de s'affiner » sous « 🆕 1.26 » (absent des notes) | Remplacé par l'item réel `b.Loop` n'empêche plus l'inlining |
| 18 | `chapitres/40-methodologie-performance.md` | Idem : « PGO affine les chemins chauds » (1.26) | Remplacé par « davantage de slices sur la pile » |
| 19 | `chapitres/36-tests-benchmarks-fuzzing.md` | Commandes `benchstat` : benchmarks de noms différents (`Naive`/`Builder`) → pas de comparaison appariée malgré une sortie `Format-8` avec delta | `benchstat` apparie par nom → renommage commun (`sed`) documenté ; vérifié |
| 20 | `chapitres/40-methodologie-performance.md` | Commandes `benchstat` incomplètes (une ligne « avant ET après », pas de redirection ; `-bench=Dedup` capture les 2 benchmarks) | Commandes explicites `old.txt`/`new.txt` + unification du nom `Dedup` ; vérifié |

## Clarifications complémentaires (ch. 9, à la demande)

- **interface `nil` vs pointeur `nil` typé** : réécriture pédagogique (modèle « paire (type, valeur) »,
  analogie du colis étiqueté, **schéma ASCII**, contraste `bad()`/`good()`, renvoi à la démo `typedNilError`).
- **type switch multi-types** : règle explicitée (`v` garde le type statique `any` sur un `case` à
  plusieurs types) + exemple, vérifiée par compilation.
- **section ⚡ Performance** : les 4 points développés (dispatch indirect/itab, boxing, assertion ≠
  réflexion, code chaud → concret/génériques) avec une **nuance mesurée** sur les génériques (🔁 Annexe E).

## Observations (non corrigées — délibérées ou hors-défaut)

- **ch. 20** : la section ⚡ cite `BenchmarkChanUnbuffered/Buffered` sans `bench_test.go` livré (asymétrie).
- **ch. 22** : un bloc annoté `main.go` présente `ctx.Err()`/`context.Cause(ctx)` comme snippet conceptuel.
- **`ns/op` machine-dépendants** : temps de référence (Apple M3) variables ; `B/op` et `allocs/op` exacts partout.
- **Annexe M** : `encoding/json/v2`, `crypto/mlkem`, `weak`, `unique`, `structs` absents de l'index (choix éditorial).
- **Titre `## 🆕 Go 1.2x`** (générique, ~25 chapitres) : convention homogène — conservée.

## Vérification finale

- `gofmt -l ./...` : ✅ vide · `go build ./...` : ✅ · `go vet ./...` : ✅ · `go test ./...` : ✅
- Box-drawing / ponctuation pleine-largeur : ✅ aucun · liens `.md` internes : ✅ aucun cassé
- Sémantique du type switch multi-types (ajout ch. 9) : ✅ confirmée par compilation
- Mécanique `benchstat` corrigée (ch. 36/40) : ✅ produit bien un benchmark de nom commun
