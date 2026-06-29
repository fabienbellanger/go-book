# Comprendre et maîtriser Go 1.26

Un livre en français pour **apprendre à écrire** du Go idiomatique **et comprendre comment
il fonctionne** sous le capot (runtime, GC, scheduler, layout mémoire), à jour pour
**Go 1.26**.

Destiné aux développeur·se·s ayant déjà des bases en programmation et en algorithmique,
mais **nouveaux en Go**. Contenu détaillé et précis, sans verbosité : schémas ASCII,
exemples compilables, listes et tableaux.

## 📖 Sommaire

La table des matières complète et cliquable est dans **[SOMMAIRE.md](SOMMAIRE.md)**.

Le livre est organisé en 9 parties (48 chapitres, 7 projets, 8 annexes) :

| Partie  | Thème                                                     | Chapitres |
| ------- | --------------------------------------------------------- | --------- |
| 0       | Introduction & mise en route                              | 0–1       |
| I       | Fondamentaux du langage                                   | 2–13      |
| II      | Mécanismes avancés du langage                             | 14–18     |
| III     | Concurrence                                               | 19–23     |
| IV      | Runtime & modèle mémoire                                  | 24–29     |
| V       | Internals des structures de données & types               | 30–35     |
| VI      | Performance, profiling & outils                           | 36–40     |
| VII     | La bibliothèque standard en pratique & mise en production | 41–47     |
| VIII    | Projets pratiques                                         | 7 projets |
| Annexes | Glossaire, antisèche, idiomes, concurrence sûre…          | A–H       |

## 🧭 Parcours de lecture

- 🟢 **Débutant Go** : Parties 0 → I → II → III, puis projets 1 et 2.
- 🟡 **Lecture intégrale** : dans l'ordre — c'est le parcours conçu.
- 🔵 **« Je connais Go, je veux les internals »** : Parties IV → V → VI (suivre les renvois 🔁).
- 🟣 **Focus concurrence** : Partie III → Ch. 28 (scheduler) → Ch. 25 (modèle mémoire) → Annexe H → Projet 3.
- 🟠 **Focus performance** : Partie VI → Ch. 26-27 (alloc/GC) → Projet 7.
- 🟤 **Focus production / stdlib** : Partie VII (lisible dès la fin de la Partie I) → projets 2 et 5.

## 🗂️ Organisation du dépôt

```
go-book/
├─ README.md          présentation + parcours de lecture
├─ SOMMAIRE.md        table des matières cliquable
├─ PLAN.md            plan de production détaillé
├─ IDEA.md            brief d'origine
├─ chapitres/         un fichier Markdown par chapitre (+ _gabarit.md)
├─ code/              exemples compilables — module unique example.com/gobook
├─ projets/           7 projets pratiques
└─ annexes/           glossaire, antisèche, ressources…
```

## ✅ Exécuter et valider les exemples

Tous les exemples des chapitres vivent dans **un seul module Go** (`code/`), pour que
l'ensemble du livre se valide en une commande :

```bash
cd code
go test ./...   # tous les exemples et tests passent
go vet ./...    # analyse statique propre
```

**Prérequis** : Go **1.26** ou supérieur (`go version`).

## ✍️ Conventions

- **Langue** : français ; identifiants de code en anglais, **commentaires en français**.
- **Schémas** : ASCII pur, pour un alignement garanti partout.
- **Émojis repères** : 🆕 nouveauté · ⚠️ piège · 💡 astuce · 🔁 renvoi · ⚡ perf · 🧪 test · 📌 synthèse.
- Un nouveau chapitre part de **[chapitres/\_gabarit.md](chapitres/_gabarit.md)**.

## 📚 Sources de référence

- [Documentation officielle Go](https://go.dev/doc/)
- [Bibliothèque standard](https://pkg.go.dev/std)
- [A Tour of Go](https://go.dev/tour/)
- [Go by Example](https://gobyexample.com/)

## 📄 Licence

Voir [LICENSE](LICENSE).
