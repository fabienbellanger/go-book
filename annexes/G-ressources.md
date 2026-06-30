# Annexe G — Ressources

> **Objectif** — Rassembler les ressources externes fiables pour approfondir,
> rester à jour et résoudre un problème : documentation officielle, apprentissage,
> internals, propositions, outils et communauté.

---

Privilégiez toujours la **documentation officielle** (`go.dev`) et la **référence
de chaque paquet** (`pkg.go.dev`) : ce sont les sources qui font autorité et qui
suivent la version courante du langage.

## Documentation officielle

- **<https://go.dev>** — Portail principal du langage : le point d'entrée par
  défaut pour les téléchargements, la documentation, le blog et les outils
  officiels.
- **<https://go.dev/doc/effective_go>** — _Effective Go_ : les idiomes
  fondateurs du style Go (nommage, interfaces, erreurs, concurrence). À lire une
  fois en entier. 🔁 voir Annexe F.
- **<https://go.dev/ref/spec>** — _The Go Programming Language Specification_ : la
  référence formelle et complète du langage. La source de vérité en cas de doute
  sémantique.
- **<https://go.dev/ref/mem>** — _The Go Memory Model_ : garanties de visibilité
  des écritures entre goroutines (relation _happens-before_). 🔁 voir Ch. 25.
- **<https://pkg.go.dev/std>** — Documentation de toute la **bibliothèque
  standard**, package par package, avec exemples exécutables.
- **<https://go.dev/doc/>** — Index de la documentation : pratique pour
  naviguer quand on ne sait pas par quel guide commencer (modules, generics,
  etc.).

## Apprentissage

- **<https://go.dev/tour/>** — _A Tour of Go_ : introduction interactive,
  exécutable dans le navigateur. Idéal pour les premiers pas et pour réviser une
  notion isolée.
- **<https://gobyexample.com>** — _Go by Example_ : recueil d'exemples courts et
  commentés, classés par thème. Parfait pour retrouver vite « comment on fait X ».
- **<https://go.dev/doc/tutorial/>** — Tutoriels officiels guidés pas à pas
  (premier module, API web, generics, fuzzing) : pour pratiquer sur un petit
  projet complet plutôt que sur des extraits isolés.

## Blog & notes de version

- **<https://go.dev/blog/>** — _The Go Blog_ : annonces, plongées techniques
  (GC, ordonnanceur, generics, itérateurs…) signées par l'équipe Go. À suivre
  pour rester à jour et comprendre le contexte d'une nouveauté.
- **<https://go.dev/doc/devel/release>** — Index de toutes les **notes de
  version** : pratique pour retrouver le changelog d'une version précise,
  passée ou future. 🔁 voir Annexe C.
- **<https://go.dev/doc/go1.26>** — Notes de la version 1.26 (et, en remplaçant le
  numéro dans l'URL, de chaque version). À consulter à chaque montée de version.

## Runtime & internals

- **<https://go.dev/blog/>** — Plusieurs articles de fond y détaillent le
  **ramasse-miettes**, l'**ordonnanceur** et la gestion mémoire ; cherchez les
  billets sur le GC et le scheduler. 🔁 voir Ch. 27, Ch. 28.
- **<https://go.dev/blog/ismmkeynote>** — Exposé de référence sur la conception et
  l'évolution du **GC** de Go (compromis latence/débit) ; niveau avancé, pour
  comprendre les choix d'implémentation au-delà du guide pratique.
- **<https://github.com/golang/go/tree/master/src/runtime>** — Le **code source
  du runtime**, abondamment commenté (fichiers `malloc.go`, `mgc.go`, `proc.go`,
  `map.go`…). La source ultime pour comprendre un comportement interne.
- **<https://go.dev/doc/gc-guide>** — _A Guide to the Go Garbage Collector_ :
  guide officiel du GC, de `GOGC` et `GOMEMLIMIT`. 🔁 voir Ch. 27.
- **<https://go.dev/doc/diagnostics>** — Panorama officiel des outils de
  **diagnostic** (profilage, traçage, détecteur de course, debug). 🔁 voir Ch. 29.

## Propositions & design

- **<https://github.com/golang/go/issues>** — Le **suivi des problèmes** et des
  propositions ; toute évolution du langage y est discutée publiquement. Utile
  pour suivre l'avancement d'une fonctionnalité ou signaler un bug.
- **<https://github.com/golang/proposal>** — Dépôt des **documents de
  conception** (design docs) des fonctionnalités majeures : pour comprendre le
  raisonnement derrière une décision de langage.
- **<https://go.dev/s/proposal-process>** — Description du **processus de
  proposition** : comment une idée devient une fonctionnalité du langage.

## Outils

- **<https://golangci-lint.run>** — `golangci-lint` : méta-linter agrégeant de
  nombreux analyseurs ; standard de fait en CI.
- **<https://pkg.go.dev/golang.org/x/perf/cmd/benchstat>** — `benchstat` :
  compare statistiquement des séries de benchmarks (gain, p-value). 🔁 voir Ch. 36.
- **<https://github.com/google/pprof>** — `pprof` : analyse et visualisation des
  profils (top, listes, graphes de flamme). 🔁 voir Ch. 37.
- **<https://github.com/go-delve/delve>** — _Delve_ : le débogueur de référence
  pour Go (`dlv`), conscient des goroutines. Pour déboguer pas à pas un
  programme, y compris concurrent.
- **<https://pkg.go.dev/golang.org/x/tools>** — Modules outils officiels
  (`goimports`, `stringer`, analyseurs `go/analysis`…). 🔁 voir Projet 6.

## Communauté

- **<https://go.dev/help>** — Page d'aide officielle pointant vers les canaux
  d'entraide et les ressources d'apprentissage. Point de départ si vous ne
  savez pas par où commencer.
- **<https://github.com/golang/go/wiki>** — Le **wiki** communautaire : FAQ,
  guides, listes de bibliothèques et de retours d'expérience. Pratique pour
  trouver une bibliothèque tierce ou un retour d'usage.
- **<https://reddit.com/r/golang>** — Communauté active (actualités, questions,
  retours d'expérience).
- **Gophers Slack** (invitation via **<https://invite.slack.golangbridge.org>**) —
  Discussions en temps réel, nombreux canaux thématiques. Pour une question
  urgente ou un échange direct avec la communauté.

---

## 📌 À retenir

- En cas de doute **sémantique**, la **spécification** (`go.dev/ref/spec`) tranche ;
  pour un **paquet**, c'est `pkg.go.dev`.
- À chaque montée de version, lisez les **notes de version** : elles signalent les
  nouveautés et les changements de comportement (🔁 voir Annexe C).
- Pour les **internals**, rien ne remplace le **code source commenté du runtime** ;
  le blog Go en donne les clés de lecture.
- Gardez le réflexe **outils** : `golangci-lint` en CI, `benchstat` pour mesurer,
  `pprof` et `dlv` pour diagnostiquer.
