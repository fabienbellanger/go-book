<!--
GABARIT DE CHAPITRE — à copier pour chaque nouveau chapitre.
Conventions (cf. PLAN.md §2) :
- Langue : français. Markdown simple.
- Code : identifiants en anglais, commentaires en français.
- Schémas : ASCII pur (+ - | / \ < > v ^), jamais de box-drawing Unicode.
- Densité : précis sans verbosité — schémas, exemples, listes, tableaux.
- Émojis repères : 🆕 nouveauté · ⚠️ piège · 💡 astuce · 🔁 renvoi · ⚡ perf · 🧪 test · 📌 synthèse.
- Les exemples compilables vivent dans code/chXX-slug/ (module unique example.com/gobook).
Supprimer ce commentaire et les encarts non pertinents dans le chapitre final.
-->

# Ch. XX — Titre du chapitre

> **Objectif** — En une à deux phrases : ce que le lecteur saura faire/comprendre à la fin.

> **Prérequis** — Chapitres ou notions à connaître (liens internes). *(facultatif)*

---

## Introduction

Mise en situation courte : pourquoi cette notion, à quel problème elle répond.

## Notion principale

Explication progressive. Une idée par section, du simple au détaillé.

```go
// code/chXX-slug/main.go
package main

import "fmt"

// greet renvoie un message de bienvenue.
func greet(name string) string {
	return fmt.Sprintf("Bonjour, %s !", name) // commentaire en français
}

func main() {
	fmt.Println(greet("Go"))
}
```

### Schéma (ASCII pur)

```
  +---------+       +---------+
  |  étape  | ----> |  étape  |
  +---------+       +---------+
```

---

## 🆕 Go 1.2x

Ce qui a changé récemment et touche ce chapitre. *(supprimer si rien)*

## ⚠️ Pièges

- Erreur classique 1 — pourquoi, et comment l'éviter.
- Erreur classique 2.

## ⚡ Performance

Coût, allocations, alternatives. Renvoi vers la partie internals si pertinent (🔁).

## 🧪 À tester soi-même

Petit exercice ou benchmark à reproduire dans `code/chXX-slug/`.

```bash
cd code && go test ./chXX-slug/...
```

---

## 📌 À retenir

- Point clé 1.
- Point clé 2.
- Point clé 3.

## 🔁 Pour aller plus loin

- Renvoi interne : voir Ch. YY pour les internals.
- Référence externe : doc officielle / pkg.go.dev.
