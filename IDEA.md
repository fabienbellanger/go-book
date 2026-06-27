# Idea

Ecrite un livre pour **apprendre** et **comprendre comment fonctionne** le language **Golang** dans sa dernière version (`1.26`).

- Ce livre devra être écrit en **français** avec le language Markdown dans un style facile à l'écrire. Il est destiné à des développeurs ayant déjà des notions de base en information et algorithmie. Le contenu doit être détaillé et précis sans être verbeux : utilisation, de schemas, exemples de code, listes.
- Les émojis sont autorisés. 
- Chaque chapitre doit être un fichier séparé.
- Chaque notion devra être illustré avec des exemples simples. Les exemples auront des variables, types, fonctions, etc en anglais mais les commentaires dans le code devront être en français.
- Pour illustrer certaines notions, n'hésite pas à faire des schemas au format **ASCII pur** (pour que tous les caractères soint correctement alignés).
- Parle entre autre de l'aspect performance, profiling (pprof), tests, benchmarks, race condition.

Pour rédiger ton livre et le structurer inspire toi :
- Doc Go officiel : https://go.dev/doc/
- La bibliothèque standard : https://pkg.go.dev/std
- Tour of Go : https://go.dev/tour/welcome/1
- Go by example : https://gobyexample.com/

Je veux que le livre aborde tous les aspects de Go (type, fonction, runtime, concurrence).

Exemple mais pas forcèment le sommaire final attendu :

```
Introduction

FONDEMENTS DU RUNTIME
1. Architecture et modèle de concurrence 
2. Bootstrap du runtime et initialisation
3. Modèle mémoire de Go
4. Allocation et gestion de la mémoire
5. Garbage collector et récupération mémoire
6. Observabilité et optimisation mémoire
7. Architecture et ordonnancement
8. Composants du runtime et monitoring
9. Fondements des structures de données

TYPES ET STRUCTURES DE DONNÉES
10. Slices et arrays
11. Strings
12. Maps – tables de hachage internes
13. Interfaces et système de types
14. Types paramétrés : polymorphisme à la compilation

MÉCANISMES AVANCÉS DU LANGAGE
15. Switch et sélection de cas
16. Fonctions anonymes et closures
17. Defer : garanties d’exécution
18. Panic et Recover : gestion des conditions exceptionnelles
19. Itérateurs par fonction
20. Package unsafe : contourner la sécurité du type
21. Réflexion : introspection et manipulation dynamique
22. Compilation et optimisations

CONCURRENCE MAÎTRISÉE
23. Goroutines
24. Primitives de synchronisation
25. Fonctions avancées de synchronisation
26. Channels : communication et signaling
27. Outils pour la programmation concurrente

ANNEXES
Démonstrations techniques et benchmarks
Algorithmes
Glossaire
```

A la fin du cours, fait des mini projets pour illistrer les aspects suivant :
- CLI (arguments, concurrence, etc.)
- API REST (routing, middleware, logger, base de données, etc.)
- Profiling, deboggage, benchmark (pprof, tests, benchmarks, traces, etc.)
- Autres à définir

Questionnne et challenge moi pour établir un plan pour ce livre.
