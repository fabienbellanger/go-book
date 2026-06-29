# Projet 7 — Profiling & debug d'un service réel (capstone)

> **Objectif** — Le **chapitre-projet de synthèse** de la Partie VI. On reprend un
> service HTTP (l'ossature du [Projet 2](../2-api-rest/)) avec un **point chaud**
> réaliste — compter les mots les plus fréquents d'un texte — et on le **profile
> de bout en bout** : `pprof` (CPU, mémoire, goroutines…), **traces** +
> **`FlightRecorder`**, **benchmarks** + `benchstat`, `-race`, et **PGO**. Le
> livrable est un **rapport avant/après chiffré** : [`RAPPORT.md`](./RAPPORT.md).
>
> **Réinvestit** — [Ch. 36 Benchmarks & fuzzing](../../chapitres/36-tests-benchmarks-fuzzing.md),
> [Ch. 37 Profiling pprof](../../chapitres/37-profiling-pprof.md),
> [Ch. 38 Traces & FlightRecorder](../../chapitres/38-traces-flight-recorder.md),
> [Ch. 39 Compilation & PGO](../../chapitres/39-compilation-inlining-pgo.md),
> [Ch. 40 Méthodologie](../../chapitres/40-methodologie-performance.md).

---

## 1. Le service

`wordstats` est un petit serveur HTTP (mux 1.22, `slog`, arrêt propre — comme le
Projet 2) dont le handler fait un **vrai travail CPU** :

```
POST /stats?n=10[&impl=v1|v2]   analyse le corps, renvoie le top-N en JSON
GET  /healthz                   sonde de vivacité
GET  /debug/pprof/...           profils CPU/heap/goroutine/... (net/http/pprof)
```

Le paramètre `impl` choisit l'implémentation du point chaud — c'est ce qui permet
de **profiler la version naïve** (`v1`) puis de **vérifier la version optimisée**
(`v2`) sur le même endpoint.

```bash
make run ARGS="-addr :8080 -slow 50ms -tracedir /tmp"
curl -s --data-binary @testdata/corpus.txt 'localhost:8080/stats?n=5' | head
```

---

## 2. Le point chaud — deux implémentations comparables

Tout le sujet tient dans `internal/analyze`, avec deux fonctions au **même
contrat** (un test le garantit) :

- **`TopWordsRegexp`** (v1, naïve) — `regexp.Split` + `strings.ToLower` global.
  Lisible, mais c'est un gouffre à CPU et à allocations.
- **`TopWordsScan`** (v2, optimisée) — balayage d'octets, minuscule dans un
  tampon réutilisé, `map[string]*int` avec **lecture de clé sans allocation**.

> 🔁 La démarche complète (profil, diagnostic, correctif, mesures) est détaillée
> dans [`RAPPORT.md`](./RAPPORT.md). Ici, on résume **comment** produire chaque
> mesure.

---

## 3. Mesurer : benchmarks (`b.Loop` + `benchstat`)

```bash
make bench        # v1 vs v2, -benchmem, 10 itérations
```

Les benchmarks utilisent `b.Loop()` (Go 1.24) — préparation hors boucle, pas
d'élimination par le compilateur — et `b.SetBytes` pour un débit en Mo/s. Pour
une comparaison statistiquement propre :

```bash
go install golang.org/x/perf/cmd/benchstat@latest
make benchstat
benchstat -col /impl bench.txt     # tableau old/new avec p-value
```

Résultat mesuré (M3) : **×8,7 en CPU**, **×1286 en allocations**, **×5400 en
octets** — voir le rapport.

---

## 4. Profiler : `pprof` (CPU & mémoire)

```bash
make cpu          # profil CPU de la v1 -> top pprof (la regexp domine)
make mem          # profil mémoire de la v1 -> alloc_space
```

À la main, sur le service en marche :

```bash
# CPU pendant 5 s de charge :
go tool pprof 'http://localhost:8080/debug/pprof/profile?seconds=5'
# tas, goroutines, blocages, mutex :
go tool pprof  http://localhost:8080/debug/pprof/heap
go tool pprof  http://localhost:8080/debug/pprof/goroutine
```

Dans `pprof` : `top`, `list TopWordsRegexp`, `web` (graphe de flamme). Le
diagnostic saute aux yeux : `regexp.(*Regexp).split` et `strings.ToLower` en tête.

---

## 5. Tracer : `runtime/trace` & `FlightRecorder` (🆕 1.25)

Deux façons d'obtenir une trace d'exécution (ordonnancement, GC, blocages) :

```bash
# 1) Trace classique à la demande, via net/http/pprof :
curl -s 'http://localhost:8080/debug/pprof/trace?seconds=3' -o trace.out
go tool trace trace.out

# 2) FlightRecorder : capture AUTOMATIQUE sur requête lente.
make run ARGS="-addr :8080 -slow 20ms -tracedir /tmp"
# toute requête > 20 ms écrit /tmp/slow-N.trace
go tool trace /tmp/slow-1.trace
```

Le **FlightRecorder** garde en mémoire une **fenêtre glissante** de la trace
récente ; on la fige dans un fichier **juste après** un évènement rare (ici une
requête lente). On obtient le « passé » de l'incident **sans** tracer en continu —
idéal en production.

---

## 6. PGO (Profile-Guided Optimization)

```bash
make pgo          # collecte un profil du chemin chaud -> default.pgo, rebuild
go version -m bin/wordstats | grep pgo   # confirme l'embarquement
```

`go build` détecte automatiquement **`default.pgo`** à la racine du module
(`-pgo=auto`). Sur ce service, le gain mesuré est **dans le bruit** — et c'est une
leçon : le PGO prolonge l'optimisation algorithmique (inlining/dévirtualisation
des chemins riches en appels), il ne la remplace pas. Détails et chiffres dans le
rapport.

---

## 7. Détection de courses & de fuites

```bash
make test                 # go test -race ./...
```

- **`-race`** sur toute la suite (handlers concurrents, FlightRecorder).
- **Fuites de goroutines** : à l'arrêt, `srv.Shutdown` + `fr.Stop()` ; le profil
  `goroutine` (`/debug/pprof/goroutine?debug=2`) doit revenir à la ligne de base
  après la charge — la façon manuelle de chasser une fuite.

---

## 8. Tests

```bash
cd projets/7-profiling
go test -race ./...
```

- **`analyze`** — équivalence v1/v2 (`TestImplementationsAgree`), top-N, limites,
  Unicode ; benchmarks v1/v2.
- **`server`** — `/stats` (JSON), parité des deux moteurs via l'API, `/healthz`,
  endpoints **pprof** montés, capture **FlightRecorder** sur seuil franchi.

---

## 9. Points de vigilance

- **Mesurer avant d'optimiser** : on profile pour **trouver** le poste dominant,
  on ne devine pas. Le plus gros gain ici venait de l'algorithme (la regexp), pas
  d'un micro-réglage.
- **Garder un oracle** : un test d'équivalence entre l'ancienne et la nouvelle
  implémentation est ce qui rend l'optimisation **sûre**.
- **`string(buf)` en clé de map** : sans allocation **en lecture seulement** ;
  l'incrément `m[string(buf)]++` réalloue. D'où `map[string]*int` + `*p++`.
- **pprof en production** : les endpoints `/debug/pprof` exposent l'intérieur du
  process — à protéger (réseau interne, auth) hors d'un bac à sable.
- **PGO n'est pas magique** : profil **représentatif** obligatoire, et gain réel
  surtout sur le code riche en appels.

---

## 10. Pour aller plus loin

- **Charge réaliste** : `hey`/`vegeta` sur `/stats`, puis profil CPU sous charge.
- **Pipeline concurrent** (Projet 3) : paralléliser l'analyse de plusieurs corpus
  et profiler `block`/`mutex` pour traquer la contention.
- **Mémoire** : comparer `inuse_space` et `alloc_space`, et observer l'effet de
  `GOGC`/`GOMEMLIMIT` au profil.

---

## 📌 À retenir

- La boucle d'optimisation est **toujours** la même : **mesurer → profiler →
  corriger le poste dominant → re-mesurer → re-profiler**.
- **pprof** dit _où_ part le temps/la mémoire ; **les traces** disent _quand_ et
  _pourquoi_ ; **`FlightRecorder`** capture le passé d'un incident rare.
- Les gains majeurs viennent de l'**algorithme** et des **allocations** ; ici
  **×8,7** CPU et **×1286** allocations, à résultat identique.
- **`b.Loop` + `benchstat`** donnent des mesures fiables et comparables ; **PGO**
  est la cerise — utile, mais pas un substitut à l'optimisation.
