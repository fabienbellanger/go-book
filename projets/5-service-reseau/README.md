# Projet 5 — Service réseau TCP : `kvd`

> **Objectif** — Écrire un **service réseau** complet avec le seul package `net` :
> un mini serveur **clé-valeur** parlant un **protocole binaire maison** sur TCP.
> On y traite ce que tout serveur réseau doit affronter : le **framing** (un flux
> d'octets n'a pas de messages), **une goroutine par connexion**, les
> **deadlines/timeouts**, l'**arrêt propre**, la **robustesse** face aux entrées
> malformées, et des **tests d'intégration** sur de vraies sockets.
>
> **Réinvestit** — [Ch. 8 Structures](../../chapitres/08-structs.md),
> [Ch. 19 Goroutines](../../chapitres/19-goroutines.md),
> [Ch. 21 Synchronisation](../../chapitres/21-synchronisation.md),
> [Ch. 22 Context](../../chapitres/22-context.md),
> [Ch. 31 Strings & octets](../../chapitres/31-strings-profondeur.md).

---

## 1. Cahier des charges

`kvd` stocke des paires clé→valeur en mémoire et répond à quatre opérations :

| Op     | Requête      | Réponse                       |
| ------ | ------------ | ----------------------------- |
| `PING` | —            | `OK` (« pong »)               |
| `SET`  | clé + valeur | `OK`                          |
| `GET`  | clé          | `OK` + valeur, ou `NOT_FOUND` |
| `DEL`  | clé          | `OK`, ou `NOT_FOUND`          |

Contraintes :

- **Protocole binaire maison** : frames préfixées de leur longueur,
  (dé)sérialisation via `encoding/binary` — pas de JSON.
- **Concurrence** : une goroutine par connexion, magasin protégé par `RWMutex`.
- **Deadlines** : fermeture des connexions **inactives** ; délai d'écriture.
- **Arrêt propre** : sur `SIGINT`/`SIGTERM`, on cesse d'accepter, on laisse un
  **délai de grâce**, puis on ferme.
- **Robustesse** : frame surdimensionnée ou payload tronqué ⇒ erreur, jamais de
  panique ni de fuite mémoire.

```bash
$ kvd -addr :7000 &
$ # le client de référence est le package internal/client (voir § 6)
```

---

## 2. Le problème du framing

TCP est un **flux d'octets**, pas de messages : un `Write` côté client peut
arriver en plusieurs morceaux, ou plusieurs `Write` fusionnés. Il faut donc
**délimiter** soi-même les messages. La solution classique : **préfixer par la
longueur**.

```
+----------------+---------------------------+
| longueur uint32|          payload          |
+----------------+---------------------------+
   (big-endian)     (requête ou réponse)
```

```go
func ReadFrame(r io.Reader) ([]byte, error) {
    var hdr [4]byte
    io.ReadFull(r, hdr[:])                 // lit EXACTEMENT 4 octets
    n := binary.BigEndian.Uint32(hdr[:])
    if n > MaxFrameSize { return nil, … }  // borne anti-abus
    payload := make([]byte, n)
    io.ReadFull(r, payload)                // attend les n octets, même fragmentés
    return payload, nil
}
```

> 💡 **`io.ReadFull` est la clé** : il boucle jusqu'à obtenir le nombre exact
> d'octets, ou renvoie `io.ErrUnexpectedEOF`. Lire « ce qui vient » avec un
> simple `Read` serait un bug classique de réseau.

Le payload encode op/statut puis des champs `[longueur][octets]`. La borne
`MaxFrameSize` (16 Mio) empêche un client malveillant d'annoncer 4 Gio pour
épuiser la mémoire du serveur.

---

## 3. Une goroutine par connexion

```go
for {
    conn, err := ln.Accept()
    if err != nil { /* arrêt ou erreur */ }
    s.wg.Go(func() {            // WaitGroup.Go (Go 1.25)
        defer s.untrackConn(conn)
        s.handleConn(conn)      // traite les requêtes en série sur CETTE connexion
    })
}
```

Chaque connexion est servie **séquentiellement** par sa goroutine (`handleConn`
boucle : lire une requête → répondre). Le **parallélisme** vient du nombre de
connexions. Le magasin partagé est protégé par un `RWMutex` (lectures
concurrentes, écritures exclusives), et `Get`/`Set` **copient** la valeur pour
qu'aucune tranche ne soit partagée entre goroutines.

---

## 4. Deadlines & robustesse

```go
conn.SetReadDeadline(time.Now().Add(s.idleTimeout))  // réarmé à chaque requête
req, err := protocol.ReadRequest(br)
// …
conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
protocol.WriteResponse(conn, resp)
```

- **Idle timeout** : une connexion qui n'envoie rien pendant `idleTimeout` est
  fermée — sinon un client oisif (ou malveillant) immobilise une goroutine
  indéfiniment.
- **Write timeout** : un client qui ne lit pas sa réponse ne doit pas bloquer le
  serveur.
- **Entrées malformées** : `ReadRequest` renvoie une erreur (frame trop grande,
  payload tronqué, opération inconnue) ; `handleConn` ferme alors la connexion
  proprement, sans paniquer.

---

## 5. Arrêt propre

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
// …
go func() {
    <-ctx.Done()
    ln.Close()                            // débloque Accept : plus de nouvelles connexions
    time.AfterFunc(grace, s.closeAllConns) // après le délai de grâce, ferme les connexions
}()
// la boucle Accept se termine, puis :
s.wg.Wait()                               // attend la fin des goroutines de connexion
```

Deux leviers complémentaires : **fermer le listener** débloque `Accept` (arrêt
des _nouvelles_ connexions), **fermer les connexions actives** débloque les
lectures en cours (`io.ReadFull` rend `net.ErrClosed`). Le **délai de grâce**
laisse les requêtes en vol se terminer avant la fermeture forcée.

---

## 6. Le client

Le package `internal/client` enveloppe le protocole : chaque appel est un
aller-retour borné par un timeout.

```go
c, _ := client.Dial("127.0.0.1:7000", 2*time.Second)
defer c.Close()

c.Set("ville", []byte("Nantes"))
v, ok, _ := c.Get("ville")   // v = "Nantes", ok = true
_, ok, _ = c.Get("absente")  // ok = false
c.Delete("ville")
c.Ping()
```

---

## 7. Tests

```bash
cd projets/5-service-reseau
go test -race ./...
```

- **Protocole** (`protocol_test.go`) — aller-retour requête/réponse (y compris
  valeurs binaires avec octets nuls), frame **tronquée** ⇒ `io.ErrUnexpectedEOF`,
  frame **surdimensionnée** ⇒ erreur, payload malformé ⇒ erreur.
- **Intégration** (`server_test.go`) — vrai serveur sur `127.0.0.1:0` (port
  éphémère), vrai client TCP : CRUD + `PING`, **25 clients concurrents** (sous
  `-race`), **idle timeout** (connexion fermée après inactivité), **arrêt**
  (nouvelles connexions refusées ensuite).

> 🧪 **À tester soi-même** : ajouter une opération `KEYS` (liste des clés) ou
> `CAS` (compare-and-swap), avec son test d'intégration. Le squelette
> requête/réponse + dispatch se réutilise tel quel.

---

## 8. Build & cross-compilation

```bash
make run ARGS="-addr :7100"   # lance le serveur
make build                    # bin/kvd
make dist                     # binaires statiques pour 5 plateformes
```

Options : `-addr`, `-idle-timeout`, `-grace`, `-version`.

---

## 9. Points de vigilance

- **Toujours `io.ReadFull`** pour un message de taille connue ; ne jamais
  supposer qu'un `Read` rend tout d'un coup (ni un seul message).
- **Borner les tailles** (`MaxFrameSize`) : une longueur vient du réseau, donc
  d'un attaquant potentiel. Allouer `make([]byte, n)` sans borne est une faille.
- **Copier les tranches partagées** : la valeur lue d'une frame pointe dans un
  buffer réutilisé ; le magasin en stocke une **copie** (`bytes.Clone`), et
  `Get` en rend une autre — sinon course de données.
- **Fermer pour débloquer** : sur le réseau, la façon d'interrompre un `Read`
  bloquant est de **fermer la connexion** (ou de poser une deadline) ; il n'y a
  pas d'annulation directe d'un appel `net` en cours.
- **`net.ErrClosed` et `io.EOF` sont normaux** à l'arrêt et quand le client part
  — les distinguer des vraies erreurs dans les logs.

---

## 10. Pour aller plus loin

- **Persistance** : journal d'écritures (WAL) rejoué au démarrage.
- **Expiration** (TTL) par clé, avec balayage en arrière-plan.
- **TLS** : envelopper le listener avec `crypto/tls` (`tls.NewListener`).
- **Pipelining** : autoriser plusieurs requêtes en vol par connexion (lecture et
  écriture découplées par deux goroutines + un canal).
- **Profilage** (Projet 7) : tracer le serveur sous charge avec `runtime/trace`.

---

## 📌 À retenir

- Sur TCP, **délimiter ses messages** soi-même : le **préfixe de longueur** +
  `io.ReadFull` est le patron de framing de référence.
- **Une connexion = une goroutine** ; le parallélisme vient du nombre de
  connexions, l'état partagé se protège (`RWMutex`, copies défensives).
- Les **deadlines** (idle, write) protègent le serveur des clients lents ou
  absents ; **fermer** la connexion est la façon de débloquer un `Read`.
- **Arrêt propre** = fermer le listener (stop des nouvelles connexions) + délai
  de grâce + fermeture des connexions actives + `WaitGroup.Wait`.
- **Tout ce qui vient du réseau est suspect** : borner les tailles, gérer les
  payloads tronqués, ne jamais paniquer.
