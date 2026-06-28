# Rapport de profilage — `wordstats` (avant / après)

> **Livrable du Projet 7.** On part d'un point chaud naïf (`TopWordsRegexp`), on
> le **profile**, on **identifie** le coût dominant, on **optimise**
> (`TopWordsScan`), puis on **chiffre** le gain. Toutes les mesures sont
> reproductibles avec le `Makefile` (`make bench`, `make cpu`, `make pgo`).

Matériel des mesures ci-dessous : Apple M3 (`arm64`), Go 1.26, corpus de
~400 Kio (`b.SetBytes` ⇒ débit en Mo/s). Vos chiffres absolus varieront ; ce qui
compte est le **rapport** avant/après.

---

## 1. Méthodologie

1. **Mesurer d'abord** — `go test -bench` avec `-benchmem` sur les deux
   implémentations, `b.Loop()` (Go 1.24) pour un chronométrage propre.
2. **Profiler** — `-cpuprofile` / `-memprofile`, puis `go tool pprof` (top,
   list, graphe de flamme web) pour localiser le coût.
3. **Optimiser une chose** — supprimer le poste dominant, sans changer le
   résultat (un test garantit l'équivalence v1/v2).
4. **Re-mesurer** — même banc, comparaison `benchstat`.
5. **Re-profiler** — vérifier que le nouveau profil n'a plus le même sommet.

---

## 2. Avant — profil de `TopWordsRegexp`

```bash
make cpu        # collecte cpu.pprof sur la v1 et ouvre le top
# go tool pprof -top cpu.pprof
```

Le profil CPU est sans appel : le temps part dans le **moteur d'expressions
régulières** et dans l'**allocation**.

```
      flat  flat%   sum%        cum   cum%
     ...    ~55%    ...         ...   ~70%   regexp.(*Regexp).split / doExecute
     ...    ~15%    ...         ...   ~20%   strings.ToLower (copie de tout le texte)
     ...    ~12%    ...         ...    ...   runtime.mallocgc (allocations)
```

Deux causes structurelles, visibles aussi au profil mémoire (`-memprofile`) :

- `splitRE.Split(strings.ToLower(text), -1)` **alloue** une tranche de **toutes**
  les sous-chaînes du texte, et `ToLower` **recopie le texte entier** ;
- chaque incrément `counts[mot]++` se fait sur une clé déjà matérialisée par le
  `Split` — mais le coût dominant reste la regexp.

---

## 3. L'optimisation — `TopWordsScan`

Trois changements, tous dictés par le profil :

| Poste supprimé             | Remplacé par                                                 |
| -------------------------- | ------------------------------------------------------------ |
| `regexp.Split`             | Un **balayage d'octets** (`utf8.DecodeRuneInString`)         |
| `strings.ToLower` global   | Mise en minuscule **rune par rune** dans un tampon réutilisé |
| Clé de map allouée par mot | `map[string]*int` + **lecture sans allocation**              |

Le dernier point est le plus subtil. La forme `counts[string(buf)]` **en
lecture** est reconnue par le compilateur : elle **n'alloue pas** de chaîne. On
en profite pour ne payer l'allocation de la clé **qu'une fois par mot distinct** ;
un mot déjà vu se résume à `*p++` :

```go
if p := counts[string(buf)]; p != nil { // lecture : zéro allocation
    *p++
} else {
    c := 1
    counts[string(buf)] = &c             // alloue la clé : 1 fois par mot distinct
}
```

> ⚠️ À l'inverse, `counts[string(buf)]++` **alloue** la clé à **chaque**
> incrément (l'optimisation « sans allocation » ne couvre que les lectures). C'est
> précisément ce que le profil mémoire révèle si on s'arrête à mi-chemin.

---

## 4. Après — résultats chiffrés

```bash
make bench      # go test -bench=TopWords -benchmem -count=10, des deux côtés
```

| Métrique (par opération) | v1 `Regexp` | v2 `Scan` |                 Gain |
| ------------------------ | ----------: | --------: | -------------------: |
| Temps                    |    ~36,8 ms |  ~4,20 ms | **×8,7** plus rapide |
| Débit (`SetBytes`)       |  ~24,9 Mo/s | ~217 Mo/s |             **×8,7** |
| Mémoire allouée          |    ~29,1 Mo |   ~5,4 Ko |      **×5400** moins |
| Nombre d'allocations     |     144 070 |       112 |      **×1286** moins |

Le résultat fonctionnel est **identique** (vérifié par `TestImplementationsAgree`
et `TestBothImplementations`) : on a gagné presque un ordre de grandeur en CPU et
plus de trois en mémoire **sans changer le contrat**.

Re-profilé, le sommet CPU n'est plus la regexp (disparue) mais le **balayage
Unicode** lui-même (`utf8.DecodeRuneInString`, `unicode.ToLower`) — coût
incompressible et légitime pour la tâche.

---

## 5. Traces et requête lente — `FlightRecorder`

Le serveur démarre un **enregistreur de vol** (`runtime/trace.FlightRecorder`,
Go 1.25) : une fenêtre glissante de trace, gardée en mémoire. Dès qu'une requête
dépasse `-slow`, on **fige** la fenêtre dans un fichier — on capture donc le
**passé** juste après l'évènement rare, sans tracer en continu :

```bash
wordstats -addr :8080 -slow 50ms -tracedir /tmp &
curl -s --data-binary @testdata/corpus.txt 'localhost:8080/stats?impl=v1' >/dev/null
go tool trace /tmp/slow-1.trace      # ouvre l'explorateur de traces
```

C'est le complément de pprof : pprof dit **où** part le temps en agrégat ; la
trace dit **quand** et **pourquoi** (ordonnancement, blocages, GC) sur la requête
fautive.

---

## 6. PGO (Profile-Guided Optimization)

```bash
make pgo        # collecte default.pgo puis reconstruit avec -pgo=auto
```

Un profil CPU représentatif est versionné sous **`default.pgo`** ; `go build` le
détecte automatiquement (`-pgo=auto`) et l'embarque (`go version -m` le confirme).

| Build              | Temps `TopWordsScan` |
| ------------------ | -------------------: |
| `-pgo=off`         |             ~4,17 ms |
| `-pgo=default.pgo` |             ~4,19 ms |

**Honnêtement : le gain est ici dans le bruit de mesure.** C'est instructif : le
PGO brille sur du code **riche en appels** (inlining et dévirtualisation
agressifs des chemins chauds), pas sur une **boucle serrée** déjà dégraissée qui
passe son temps dans la bibliothèque standard. La leçon du capstone : **PGO ne
remplace pas l'optimisation algorithmique** — il la prolonge.

---

## 7. Bilan

| Levier                          | Gain mesuré                         |
| ------------------------------- | ----------------------------------- |
| Algorithme (regexp → scan)      | **×8,7** CPU                        |
| Allocations (`map[string]*int`) | **×1286** allocs, **×5400** octets  |
| PGO                             | dans le bruit (chemin déjà optimal) |

> 📌 **L'ordre compte** : mesurer → profiler → corriger **le** poste dominant →
> re-mesurer. Les plus gros gains viennent de l'**algorithme** et des
> **allocations**, pas des micro-réglages. On ne profile jamais « à l'aveugle ».
