# 52 — Réseau bas niveau (`net`)

> **Objectif** — Comprendre le **modèle réseau** de Go sous `net/http` : écouter et
> se connecter en TCP (`net.Listen`/`net.Dial`), traiter les connexions avec une
> **goroutine par connexion**, cadrer un flux d'octets, poser des **deadlines**,
> annuler une connexion via `context`, échanger des datagrammes **UDP** et résoudre
> des noms **DNS**.
>
> **Prérequis** — [Ch. 41 — Entrées/sorties & flux](41-io-flux.md) (`io.Reader`/`Writer`,
> `bufio`), [Ch. 19 — Goroutines](19-goroutines.md), [Ch. 22 — `context`](22-context.md)

---

## Introduction

`net/http` ([Ch. 45](45-net-http.md)) est confortable, mais il est **bâti au-dessus**
du package `net`. Dès qu'on sort du HTTP — un protocole maison, un serveur TCP, du
UDP, un client qui parle à Redis ou à un annuaire — c'est `net` qu'on manipule
directement. Ce chapitre pose les fondations que le [Projet 5 — Service réseau](../projets/5-service-reseau/)
réinvestit.

L'idée centrale à retenir : **une connexion réseau est un `io.ReadWriteCloser`**
([Ch. 41](41-io-flux.md)). Tout ce qu'on a appris sur les flux — `io.Copy`, `bufio`,
la décoration — s'applique tel quel à une socket, sans rien connaître du réseau.

L'exemple complet est dans [`code/ch52-reseau-net/`](../code/ch52-reseau-net/).

---

## Le modèle client/serveur TCP

Un serveur suit toujours le même squelette : **écouter**, puis **accepter** en boucle,
et traiter chaque connexion **dans sa propre goroutine**.

```
  net.Listen("tcp", ":9000")          Accept()            go handleConn(c)
  +----------------+   boucle   +----------------+      +------------------+
  |  net.Listener  | ---------> |    net.Conn    | ---> | goroutine dédiée |
  +----------------+            +----------------+      +------------------+
        ^  un port qui attend         ^ une connexion         ^ une par client
        |                             | établie                 (Ch. 19 / 23)
```

```go
// code/ch52-reseau-net/main.go
func serveEcho(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // Listener fermé : fin normale de la boucle d'acceptation
		}
		go handleEcho(conn)
	}
}
```

- **`net.Listen("tcp", addr)`** renvoie un `net.Listener`. L'adresse `"127.0.0.1:0"`
  demande à l'OS de choisir un **port éphémère** libre — pratique en test, on lit le
  port réel avec `ln.Addr()`.
- **`Accept()`** bloque jusqu'à l'arrivée d'un client, puis renvoie une `net.Conn`.
- **Une goroutine par connexion** : c'est le modèle Go canonique. Léger grâce à
  l'ordonnanceur ([Ch. 28](28-ordonnanceur-gmp.md)), il évite l'`epoll` manuel des
  autres langages — le runtime le fait pour vous.

Côté client, `net.Dial` (ou `net.DialTimeout`) établit la connexion :

```go
// code/ch52-reseau-net/main.go
conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
```

## Une connexion est un flux `io`

`net.Conn` embarque `io.Reader`, `io.Writer` et `io.Closer` : c'est un
`io.ReadWriteCloser`. On y branche donc directement l'outillage du [Ch. 41](41-io-flux.md).
Le handler d'écho lit ligne par ligne avec un `bufio.Scanner` et renvoie chaque ligne :

```go
// code/ch52-reseau-net/main.go
func handleEcho(conn net.Conn) {
	defer conn.Close() // TOUJOURS fermer : sinon fuite de descripteur + goroutine
	sc := bufio.NewScanner(conn)
	for sc.Scan() { // ScanLines découpe le FLUX d'octets en lignes
		if _, err := fmt.Fprintln(conn, sc.Text()); err != nil {
			return
		}
	}
}
```

> 💡 Rien dans ce handler ne parle « réseau ». Remplacez `conn` par un fichier ou un
> `bytes.Buffer` et le code fonctionne à l'identique : c'est toute la puissance des
> interfaces `io` ([Ch. 9](09-interfaces.md), [Ch. 41](41-io-flux.md)).

### TCP est un flux d'octets, pas un flux de messages

⚠️ **Le piège le plus courant du débutant réseau.** TCP ne préserve **pas** les
frontières entre vos écritures. Trois `Write` de 10 octets côté client peuvent arriver
en **un seul** `Read` de 30 octets — ou en cinq `Read`. TCP garantit l'**ordre** et
l'**intégrité** des octets, jamais leur **découpage**.

```
  émetteur : Write("AB") Write("CD") Write("EF")
  réseau   :        A B C D E F           (les frontières sont perdues)
  récepteur:  Read -> "ABCDE"   Read -> "F"   (découpage IMPRÉVISIBLE)
```

Il faut donc **cadrer** soi-même les messages. Trois stratégies :

| Cadrage                         | Comment                                            | Outil Go                          |
| ------------------------------- | -------------------------------------------------- | --------------------------------- |
| **Délimiteur** (ex. `'\n'`)     | un caractère marque la fin du message              | `bufio.Scanner`, `ReadString('\n')` |
| **Longueur préfixée**           | 2 ou 4 octets de taille, puis le corps             | `binary.Read` + `io.ReadFull`     |
| **Taille fixe**                 | chaque message fait exactement N octets            | `io.ReadFull(conn, buf)`          |

Notre serveur d'écho utilise le cadrage par ligne : `bufio.Scanner` côté serveur,
`ReadString('\n')` côté client, avec un `\n` ajouté par `Fprintln`.

## Deadlines : ne jamais bloquer pour toujours

Un `Read` sur une socket **bloque** tant que rien n'arrive. Si le pair se tait (ou
disparaît sans fermer proprement), la goroutine reste coincée **indéfiniment**. La
parade est la **deadline**, propre à `net` :

```go
// code/ch52-reseau-net/main.go
conn.SetDeadline(time.Now().Add(2 * time.Second)) // instant absolu, pas une durée
```

⚠️ Une deadline est un **instant absolu** (`time.Time`), pas un délai relatif. Il faut
la **repositionner** avant chaque opération longue (souvent : au début de chaque
itération de lecture). `SetReadDeadline` et `SetWriteDeadline` ciblent un seul sens.

Un dépassement se reconnaît via `net.Error.Timeout()` — le motif idiomatique
([Ch. 10](10-erreurs.md)) :

```go
// code/ch52-reseau-net/main.go
func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
```

> 💡 Une deadline dépassée renvoie une erreur qui satisfait aussi
> `errors.Is(err, os.ErrDeadlineExceeded)`. La connexion **reste utilisable** : on peut
> repositionner la deadline et réessayer.

## Annuler une connexion avec `context`

`net.DialTimeout` borne la durée d'établissement, mais ne réagit pas à une annulation
extérieure. Dès qu'un `context` circule ([Ch. 22](22-context.md)), utilisez
`net.Dialer.DialContext` : l'annulation du contexte interrompt la tentative en cours.

```go
// code/ch52-reseau-net/main.go
func dialContext(ctx context.Context, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "tcp", addr)
}
```

C'est exactement ce que fait `http.Client` en interne : son `Transport` possède un
`DialContext` et propage le contexte de la requête jusqu'à la socket.

## UDP : des datagrammes sans connexion

Là où TCP est un flux fiable et ordonné, **UDP** transporte des **datagrammes**
indépendants : pas de connexion, pas de garantie de livraison ni d'ordre, mais une
latence minimale (DNS, jeux, métriques, VoIP). L'API change : `net.ListenPacket`
donne un `net.PacketConn`, avec `ReadFrom`/`WriteTo` qui exposent l'adresse du pair.

```go
// code/ch52-reseau-net/main.go
srv, err := net.ListenPacket("udp", "127.0.0.1:0")
// ...
n, addr, err := srv.ReadFrom(buf) // qui a envoyé ? -> addr
srv.WriteTo(buf[:n], addr)        // on répond à cet expéditeur
```

⚠️ **Un `ReadFrom` = un datagramme entier.** Contrairement à TCP, les frontières
**sont** préservées, mais un datagramme trop grand pour le buffer est **tronqué**
silencieusement. Dimensionnez le buffer à la MTU (~1500 octets sur Ethernet, ou 65 507
octets au maximum théorique) selon le protocole.

## Adresses et résolution DNS

`net` fournit une boîte à outils pour manipuler adresses et noms — **sans** I/O pour
les fonctions purement syntaxiques :

| Fonction                         | Rôle                                                   |
| -------------------------------- | ------------------------------------------------------ |
| `net.SplitHostPort("h:p")`       | sépare hôte et port (gère les IPv6 entre crochets)     |
| `net.JoinHostPort(h, p)`         | l'inverse, avec les crochets IPv6 si besoin            |
| `net.ParseIP("::1")`             | analyse une IP → `net.IP` (`IsLoopback`, `To4`…)       |
| `net.LookupHost(name)`           | résout un nom → adresses (I/O, DNS)                    |
| `net.LookupIP` / `LookupCNAME`   | variantes typées de la résolution                     |
| `net.DefaultResolver`            | résolveur configurable, méthodes `...(ctx, ...)`       |

Pour toute **résolution** (donc une I/O réseau), préférez la variante à `context`, qui
respecte annulation et timeout :

```go
// code/ch52-reseau-net/main.go
func resolveHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}
```

## ⚠️ Pièges

- **Oublier `conn.Close()`** (et `ln.Close()`) : chaque connexion non fermée fuit un
  descripteur de fichier **et** la goroutine qui la traite ([Ch. 23](23-patterns-concurrence.md)).
  Réflexe : `defer conn.Close()` dès l'`Accept`/`Dial` réussi.
- **Croire qu'un `Read` renvoie un message complet** : c'est un **flux** d'octets
  (voir plus haut). Cadrez toujours explicitement.
- **Aucune deadline** : une lecture sur un pair muet bloque pour toujours. Toute
  connexion exposée au réseau doit avoir des deadlines.
- **Ignorer les erreurs de `Accept`** : distinguez une erreur fatale (Listener fermé →
  on sort) d'une erreur transitoire. Ne bouclez pas à vide en ignorant l'erreur.
- **Fuite d'écoute en test** : un `net.Listen` sans `Close` laisse le port occupé.
  Utilisez `t.Cleanup(func() { ln.Close() })`.
- **Datagramme UDP tronqué** : un buffer trop petit perd la fin du message sans
  erreur. Dimensionnez selon la MTU.

## ⚡ Performance

- **Goroutine par connexion.** Le modèle passe à l'échelle grâce à l'ordonnanceur
  ([Ch. 28](28-ordonnanceur-gmp.md)) : des dizaines de milliers de goroutines
  bloquées en `Read` ne coûtent presque rien (le runtime les parque sur du `netpoll`).
  Pas besoin de pool de threads manuel.
- **Tamponnez.** Enveloppez `conn` dans un `bufio.Reader`/`bufio.Writer`
  ([Ch. 41](41-io-flux.md)) pour amortir les syscalls, et **réutilisez** les buffers
  (`sync.Pool`, [Ch. 21](21-synchronisation.md)) sur les chemins chauds.
- **`TCP_NODELAY`.** Go **désactive l'algorithme de Nagle par défaut** (`SetNoDelay(true)`),
  ce qui privilégie la latence. Pour un protocole qui envoie beaucoup de petits paquets
  qu'on pourrait regrouper, `(*net.TCPConn).SetNoDelay(false)` peut réduire le trafic.
- **`io.Copy` zéro-copie** entre deux sockets (ou socket↔fichier) : le noyau peut
  transférer sans repasser par l'espace utilisateur (`sendfile`/`splice`).

## 🧪 À tester soi-même

Dans [`code/ch52-reseau-net/`](../code/ch52-reseau-net/) :

```bash
cd code && go test ./ch52-reseau-net/
```

Les tests écoutent sur `127.0.0.1:0` (port éphémère, aucun accès réseau externe) :
aller-retour TCP, annulation par `context`, expiration de deadline, aller-retour UDP.
Exercice : ajoutez un cadrage **par longueur préfixée** (`binary.Write` d'un `uint32`,
puis le corps) et un test qui prouve qu'un message contenant un `\n` transite intact —
là où le cadrage par ligne le couperait.

---

## 📌 À retenir

- **`net.Listen` + boucle `Accept` + goroutine par connexion** : le squelette de tout
  serveur Go. `net.Dial`/`DialContext` côté client.
- **`net.Conn` est un `io.ReadWriteCloser`** : `bufio`, `io.Copy` et la décoration du
  [Ch. 41](41-io-flux.md) s'y appliquent sans rien savoir du réseau.
- **TCP est un flux d'octets** : cadrez vos messages (délimiteur, longueur préfixée,
  taille fixe). **UDP** préserve les datagrammes mais sans fiabilité ni ordre.
- **Toujours poser des deadlines** (instants absolus) et **toujours** `Close`.
  `net.Error.Timeout()` distingue un dépassement d'une autre erreur.
- Pour toute I/O réseau annulable (dial, DNS), passez par les API à **`context`**.

## 🔁 Pour aller plus loin

- [Ch. 45 — `net/http`](45-net-http.md) : le serveur/client HTTP, bâti sur `net`.
- [Ch. 41 — Entrées/sorties & flux](41-io-flux.md) : `io`/`bufio` qu'on branche sur `net.Conn`.
- [Ch. 22 — `context`](22-context.md) : annulation propagée jusqu'à la socket.
- [Ch. 23 — Patterns de concurrence](23-patterns-concurrence.md) : ne pas fuiter les goroutines de connexion.
- [Projet 5 — Service réseau TCP/RPC](../projets/5-service-reseau/) : mise en pratique complète.
- Référence : [`pkg.go.dev/net`](https://pkg.go.dev/net).
