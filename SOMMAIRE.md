# Sommaire — Comprendre et maîtriser Go 1.26

> Table des matières cliquable. Les chapitres non encore rédigés pointent vers leur
> fichier cible (à créer dans `chapitres/`). Voir [PLAN.md](PLAN.md) pour le détail.

## Partie 0 — Introduction & mise en route

- Ch. 0 — [Pourquoi Go ? Philosophie & panorama](chapitres/00-pourquoi-go.md)
- Ch. 1 — [Installation, toolchain & premier programme](chapitres/01-installation-toolchain.md)

## Partie I — Fondamentaux du langage

- Ch. 2 — [Structure d'un programme : packages, `import`, `main`](chapitres/02-structure-programme.md)
- Ch. 3 — [Variables, constantes & types de base](chapitres/03-variables-constantes-types.md)
- Ch. 4 — [Flux de contrôle](chapitres/04-flux-controle.md)
- Ch. 5 — [Fonctions](chapitres/05-fonctions.md)
- Ch. 6 — [Arrays & slices (usage)](chapitres/06-arrays-slices.md)
- Ch. 7 — [Maps & strings (usage)](chapitres/07-maps-strings.md)
- Ch. 8 — [Structs, méthodes & composition](chapitres/08-structs-methodes.md)
- Ch. 9 — [Interfaces (fondamentaux)](chapitres/09-interfaces.md)
- Ch. 10 — [Gestion des erreurs](chapitres/10-erreurs.md)
- Ch. 11 — [Généricité : types paramétrés](chapitres/11-genericite.md)
- Ch. 12 — [Packages, modules & organisation du code](chapitres/12-packages-modules.md)
- Ch. 13 — [Tests & outillage de base](chapitres/13-tests-outillage.md)

## Partie II — Mécanismes avancés du langage

- Ch. 14 — [`switch` & sélection de cas](chapitres/14-switch.md)
- Ch. 15 — [Fonctions anonymes & closures](chapitres/15-closures.md)
- Ch. 16 — [`defer` : garanties d'exécution](chapitres/16-defer.md)
- Ch. 17 — [`panic` & `recover`](chapitres/17-panic-recover.md)
- Ch. 18 — [Itérateurs par fonction (range-over-func)](chapitres/18-iterateurs.md)

## Partie III — Concurrence

- Ch. 19 — [Goroutines : le modèle](chapitres/19-goroutines.md)
- Ch. 20 — [Channels & `select`](chapitres/20-channels-select.md)
- Ch. 21 — [Primitives de synchronisation](chapitres/21-synchronisation.md)
- Ch. 22 — [`context` : annulation, délais, valeurs](chapitres/22-context.md)
- Ch. 23 — [Patterns de concurrence, data races & tests concurrents](chapitres/23-patterns-concurrence.md)

## Partie IV — Runtime & modèle mémoire

- Ch. 24 — [Architecture du runtime & bootstrap](chapitres/24-runtime-bootstrap.md)
- Ch. 25 — [Le modèle mémoire de Go](chapitres/25-modele-memoire.md)
- Ch. 26 — [Allocation mémoire & escape analysis](chapitres/26-allocation-escape.md)
- Ch. 27 — [Le garbage collector](chapitres/27-garbage-collector.md)
- Ch. 28 — [L'ordonnanceur (G-M-P)](chapitres/28-ordonnanceur-gmp.md)
- Ch. 29 — [Observabilité du runtime & monitoring](chapitres/29-observabilite-runtime.md)

## Partie V — Internals des structures de données & du système de types

- Ch. 30 — [Slices & arrays en profondeur](chapitres/30-slices-profondeur.md)
- Ch. 31 — [Strings en profondeur](chapitres/31-strings-profondeur.md)
- Ch. 32 — [Maps : tables de hachage](chapitres/32-maps-hachage.md)
- Ch. 33 — [Interfaces & système de types en profondeur](chapitres/33-interfaces-profondeur.md)
- Ch. 34 — [Réflexion (`reflect`)](chapitres/34-reflexion.md)
- Ch. 35 — [`unsafe` & interopérabilité bas niveau](chapitres/35-unsafe-cgo.md)

## Partie VI — Performance, profiling & outils

- Ch. 36 — [Tests avancés, benchmarks & fuzzing](chapitres/36-tests-benchmarks-fuzzing.md)
- Ch. 37 — [Profiling avec pprof](chapitres/37-profiling-pprof.md)
- Ch. 38 — [Traces d'exécution & Flight Recorder](chapitres/38-traces-flight-recorder.md)
- Ch. 39 — [Compilation, inlining, PGO & optimisations](chapitres/39-compilation-inlining-pgo.md)
- Ch. 40 — [Méthodologie de performance](chapitres/40-methodologie-performance.md)

## Partie VII — La bibliothèque standard en pratique & mise en production

> Lisible dès la fin de la Partie I : ces chapitres montrent « la façon Go » des
> packages du quotidien, puis comment embarquer et déployer un binaire.

- Ch. 41 — [Entrées/sorties & flux : `io`, `bufio`, `bytes`](chapitres/41-io-flux.md)
- Ch. 42 — [Encodages & sérialisation : `encoding/json`, `gob`/`csv`/`xml`, `regexp`](chapitres/42-encodages-serialisation.md)
- Ch. 43 — [Journalisation structurée : `log/slog`](chapitres/43-journalisation-slog.md)
- Ch. 44 — [Le temps en pratique : `time`](chapitres/44-temps.md)
- Ch. 45 — [`net/http` : serveur & client](chapitres/45-net-http.md)
- Ch. 46 — [Embarquer & déployer : `embed`, build tags, binaires statiques](chapitres/46-embed-build-deploiement.md)
- Ch. 47 — [Sécurité & chaîne d'approvisionnement](chapitres/47-securite-supply-chain.md)

## Partie VIII — Projets pratiques

- Projet 1 — [Outil CLI](projets/1-cli/)
- Projet 2 — [API REST](projets/2-api-rest/)
- Projet 3 — [Pipeline concurrent / worker pool](projets/3-pipeline/)
- Projet 4 — [Bibliothèque générique réutilisable](projets/4-lib-generique/)
- Projet 5 — [Service réseau TCP/RPC](projets/5-service-reseau/)
- Projet 6 — [Générateur de code (`go:generate`)](projets/6-codegen/)
- Projet 7 — [Profiling & debug d'un service réel (capstone)](projets/7-profiling/)

## Annexes

- A — [Glossaire](annexes/A-glossaire.md)
- B — [Antisèche des commandes `go`](annexes/B-antiseche-go.md)
- C — [Carte des nouveautés Go 1.21 → 1.26](annexes/C-nouveautes-1.21-1.26.md)
- D — [Algorithmes & structures de données en Go](annexes/D-algorithmes.md)
- E — [Démonstrations techniques & benchmarks](annexes/E-demonstrations-benchmarks.md)
- F — [Idiomes & style](annexes/F-idiomes-style.md)
- G — [Ressources](annexes/G-ressources.md)
- H — [Concurrence sûre : éviter data races & deadlocks](annexes/H-concurrence-sure.md)
