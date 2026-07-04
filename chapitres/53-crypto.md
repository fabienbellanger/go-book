# 53 — Cryptographie pratique

> **Objectif** — Savoir **utiliser correctement** les primitives cryptographiques de la
> bibliothèque standard : hacher un flux, générer de l'aléa et des jetons sûrs, comparer des
> secrets sans fuite de temps, signer avec HMAC, chiffrer en mode authentifié (AES-GCM) et
> configurer TLS sans se tirer une balle dans le pied. Le fil rouge : **choisir la bonne
> primitive** et **ne pas réinventer** ce que la stdlib fait déjà bien.
>
> **Prérequis** — [Ch. 41 — Entrées/sorties & flux](41-io-flux.md) (`io.Reader`/`io.Writer`),
> [Ch. 47 — Sécurité & chaîne d'approvisionnement](47-securite-supply-chain.md) (le volet politique
> de la sécurité).

---

## Introduction

La cryptographie ne se rédige pas « à la main » : les erreurs y sont **silencieuses** (le code
compile, chiffre, et reste vulnérable) et **coûteuses**. La bonne nouvelle, c'est que la
bibliothèque standard de Go couvre l'essentiel — hachage, HMAC, chiffrement authentifié, TLS — avec
des API sobres. Ce chapitre montre **le geste correct** pour chaque besoin courant, en évitant les
pièges classiques.

Le ch. 47 pose la **politique** (« jamais `math/rand` pour un secret », mots de passe salés, TLS
durci) ; ce chapitre en donne la **mécanique** : les fonctions à appeler, dans quel ordre, avec
quelles vérifications. Chaque règle du ch. 47 a ici son code.

Une distinction structure tout le chapitre :

```
  Ai-je une clé secrète partagée ?
       |                     |
      non                   oui
       |                     |
   HACHAGE              +----+--------------------+
   (empreinte)          |                          |
   sha256               je veux PROUVER          je veux CACHER
                        l'intégrité              le contenu
                        -> HMAC                  -> chiffrement AEAD
                           (signature)              (AES-GCM)
```

L'exemple complet est dans [`code/ch53-crypto/`](../code/ch53-crypto/).

---

## Hacher un flux

Une **empreinte** (hash) réduit une donnée de taille quelconque à une valeur fixe. `crypto/sha256`
et `crypto/sha512` renvoient un type qui implémente `hash.Hash` — lequel **est un `io.Writer`**
([Ch. 41](41-io-flux.md)). On peut donc hacher un flux **bloc par bloc**, sans jamais le charger en
entier :

```go
// code/ch53-crypto/main.go
func hashStream(r io.Reader) ([]byte, error) {
	h := sha256.New() // h implémente io.Writer
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil // Sum(nil) renvoie l'empreinte sans rien y préfixer
}
```

`io.Copy(h, r)` pousse le flux dans le hacheur ; `Sum(nil)` restitue les 32 octets de l'empreinte.
Le même patron hache un fichier de plusieurs gigaoctets en mémoire constante — c'est exactement
ce qui rend `sha256sum` efficace.

> 💡 **Une empreinte n'est pas un secret.** SHA-256 est **déterministe** et **non réversible**, mais
> il n'a pas de clé : n'importe qui peut recalculer l'empreinte d'un message. Pour prouver qu'un
> message vient bien de vous, il faut une clé — c'est le rôle du **HMAC** (plus bas).

> ⚠️ **MD5 et SHA-1 : checksums, pas sécurité.** Les collisions sont **pratiques** sur ces deux
> algorithmes. `crypto/md5`/`crypto/sha1` restent utiles pour un checksum non sécurisé (détecter une
> corruption accidentelle, une clé de cache), **jamais** pour signer, authentifier ou dériver quoi
> que ce soit de sensible. Par défaut : **SHA-256**.

## Aléa cryptographique & jetons

Tout ce qui doit **résister à un attaquant** — clé, nonce, sel, jeton de session, mot de passe
temporaire — se tire de `crypto/rand`, le générateur du système (CSPRNG). **Jamais** de
`math/rand`/`math/rand/v2`, qui est déterministe et prédictible (🔁 [Ch. 47 §3.1](47-securite-supply-chain.md)).

```go
// code/ch53-crypto/main.go
func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil { // rand.Read ne renvoie jamais de lecture partielle
		return nil, err
	}
	return b, nil
}
```

Pour un **jeton** prêt à l'emploi, Go 1.24 offre mieux qu'un `Read` + encodage manuel :

```go
// code/ch53-crypto/main.go
func newToken() string {
	return rand.Text() // 🆕 Go 1.24 : chaîne base32, >= 128 bits d'entropie
}
```

🆕 `rand.Text()` tire du CSPRNG et renvoie une chaîne lisible (base32, sans caractères ambigus),
avec au moins 128 bits d'entropie : l'outil idéal pour un identifiant de session ou un lien de
réinitialisation, sans se soucier de l'encodage.

| Besoin                              | Bon outil                          | À proscrire                     |
| ----------------------------------- | ---------------------------------- | ------------------------------- |
| Clé / nonce / sel                   | `crypto/rand.Read`                 | `math/rand`                     |
| Jeton lisible                       | `rand.Text()` (1.24)               | `math/rand`, compteur, horodate |
| Dé de simulation, jeu, échantillon  | `math/rand/v2`                     | —                               |

## Comparer des secrets en temps constant

Comparer un secret (jeton reçu, MAC, empreinte) avec `==` ou `bytes.Equal` **s'arrête au premier
octet différent**. La durée de la comparaison **fuit** alors la position de la différence : un
attaquant qui mesure ce temps peut reconstituer le secret octet par octet (_timing attack_).
`crypto/subtle` compare en temps **indépendant du contenu** :

```go
// code/ch53-crypto/main.go
func equalSecret(a, b []byte) bool {
	// ConstantTimeCompare renvoie 1 si égaux ET de même longueur, 0 sinon.
	return subtle.ConstantTimeCompare(a, b) == 1
}
```

Pour comparer un **MAC**, préférez `hmac.Equal` (ci-dessous), qui fait la même chose avec la bonne
sémantique.

## HMAC : intégrité + authenticité

Un **HMAC** répond à la question « ce message vient-il bien de quelqu'un qui connaît la clé, et
n'a-t-il pas été modifié ? ». On l'utilise pour signer un cookie, un webhook, un paramètre d'URL.
`crypto/hmac` se combine à une fonction de hachage :

```go
// code/ch53-crypto/main.go
func signHMAC(key, msg []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg) // hmac.Hash est aussi un io.Writer
	return mac.Sum(nil)
}

func verifyHMAC(key, msg, sig []byte) bool {
	expected := signHMAC(key, msg)
	return hmac.Equal(sig, expected) // temps constant : PAS bytes.Equal
}
```

```
  émetteur                              destinataire (même clé)
  msg ---> HMAC(clé, msg) = sig         msg', sig'
           envoie (msg, sig)  ------->  recalcule HMAC(clé, msg')
                                        hmac.Equal(sig', recalculé) ?
                                          oui -> intègre & authentique
                                          non -> rejeté
```

> ⚠️ **Vérifiez avec `hmac.Equal`, jamais `==`.** Comparer un MAC avec `==`/`bytes.Equal` rouvre la
> faille de timing que le HMAC était censé éviter.

> 💡 **HMAC ≠ chiffrement.** Le HMAC **ne cache pas** le message : il l'accompagne d'une preuve. Si
> le contenu doit rester secret, il faut chiffrer (section suivante) — et l'AEAD fait déjà les deux.

## Chiffrement authentifié : AES-GCM

Pour **cacher** un contenu, on chiffre. La bonne primitive par défaut est un mode **AEAD**
(_Authenticated Encryption with Associated Data_) : il chiffre **et** authentifie en une passe, si
bien qu'une altération du chiffré est **détectée** au déchiffrement. En Go, c'est `crypto/aes` +
`crypto/cipher.NewGCM`.

```go
// code/ch53-crypto/main.go
func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := randomBytes(gcm.NonceSize()) // ⚠️ un nonce UNIQUE par message
	if err != nil {
		return nil, err
	}
	// Seal(dst, nonce, plaintext, additionalData) : en passant nonce comme dst,
	// le nonce se retrouve en tête du blob renvoyé.
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

Au déchiffrement, `Open` **vérifie le tag** : un octet modifié dans le chiffré (ou une mauvaise
clé) renvoie une **erreur**, pas un clair corrompu.

```go
// code/ch53-crypto/main.go
func decrypt(key, blob []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("chiffré trop court pour contenir le nonce")
	}
	nonce, ciphertext := blob[:ns], blob[ns:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
```

Le format retenu ici est `nonce || ciphertext || tag` : on préfixe le nonce (qui n'est **pas**
secret) pour tout transporter ensemble.

> ⚠️ **Ne réutilisez jamais un nonce avec la même clé.** Avec GCM, réutiliser un nonce **casse**
> la confidentialité et l'authentification. Tirez-le de `crypto/rand` à **chaque** message (12
> octets → risque de collision négligeable). Si vous chiffrez d'énormes volumes sous une seule clé,
> renouvelez la clé.

> ⚠️ **Chiffrer sans authentifier est une faute.** Le mode CBC brut (`cipher.NewCBCEncrypter`) ne
> détecte pas une altération et se prête aux attaques par _padding oracle_. **Par défaut : AEAD**
> (GCM, ou ChaCha20-Poly1305 via `golang.org/x/crypto`). N'utilisez un mode bloc nu que si vous
> savez précisément pourquoi.

## TLS : le chiffrement du réseau

La plupart du temps, on ne chiffre pas soi-même : on parle **TLS**, et `net/http` ([Ch. 45](45-net-http.md))
comme `crypto/tls` s'en chargent. L'essentiel est de **configurer** correctement.

Côté **client**, une config durcie fixe une version minimale et **garde la vérification de
certificat active** :

```go
// code/ch53-crypto/main.go
func clientTLSConfig(serverName string) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
		// InsecureSkipVerify: false — NE JAMAIS mettre à true en production.
	}
}
```

> ⚠️ **`InsecureSkipVerify: true` désactive la vérification du certificat** : la connexion devient
> interceptable (_man-in-the-middle_). C'est un réglage de débogage, jamais de production. En revue
> de code, c'est un signal d'alarme (🔁 [Ch. 47 §7.2](47-securite-supply-chain.md)).

Pour un serveur, on fournit un ou plusieurs `tls.Certificate` (voir `selfSignedCert` dans le code
d'appui, qui en génère un en mémoire pour la démo). Le **mTLS** (authentification mutuelle) s'active
en posant `ClientAuth: tls.RequireAndVerifyClientCert` et un `ClientCAs` côté serveur : le client
présente alors lui aussi un certificat. L'aller-retour complet — serveur loopback, client qui
**vérifie** le certificat via un pool de confiance dédié plutôt que de désactiver la vérification —
est dans `tlsEcho` ([`code/ch53-crypto/main.go`](../code/ch53-crypto/main.go)). La mécanique réseau
sous-jacente (`net.Listen`, `net.Dial`) est le sujet du [Ch. 52 — Réseau bas niveau](52-reseau-net.md).

## Mots de passe : ni hachage nu, ni stdlib

Cas particulier qui **ne relève pas des primitives ci-dessus** : un mot de passe ne se hache
**pas** avec SHA-256. Un hachage rapide se force à des milliards d'essais par seconde sur GPU. Il
faut une fonction de dérivation **lente et salée** — `bcrypt`, `scrypt` ou `argon2` — que la
bibliothèque standard **ne fournit pas** : elles vivent dans `golang.org/x/crypto` (maintenu par
l'équipe Go, hors stdlib).

```go
// pseudo-code : hash := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
//               err  := bcrypt.CompareHashAndPassword(hash, []byte(saisi))
```

⚡ La **lenteur est ici une fonctionnalité** : on règle le facteur de coût pour qu'un hachage prenne
~100 ms — imperceptible à un login légitime, ruineux pour une attaque par force brute
(🔁 [Ch. 47 §3.3](47-securite-supply-chain.md)).

## ⚠️ Pièges

- **`math/rand` pour un secret.** Prédictible : un attaquant reconstitue clés et jetons. `crypto/rand`
  (ou `rand.Text()`) pour **tout** ce qui doit résister.
- **Réutiliser un nonce GCM** avec la même clé : casse confidentialité **et** authentification.
  Nonce aléatoire par message.
- **Comparer un MAC/jeton avec `==` ou `bytes.Equal`** : fuite par timing. `hmac.Equal` /
  `subtle.ConstantTimeCompare`.
- **MD5/SHA-1 « pour sécuriser »** : collisions pratiques. Réservés aux checksums non sensibles.
- **`InsecureSkipVerify: true`** en production : man-in-the-middle grand ouvert.
- **Chiffrer sans authentifier** (CBC nu) : altérations indétectables, _padding oracle_. Utiliser
  un mode AEAD.
- **Hacher un mot de passe avec SHA-256** : trop rapide. `bcrypt`/`argon2`, salés et lents.

## ⚡ Performance

- **AES-GCM est matériellement accéléré** (jeu d'instructions AES-NI sur x86, équivalents ARM) :
  chiffrer/déchiffrer coûte quelques cycles par octet. Inutile d'inventer « plus rapide ».
- **Hacher en streaming** (`io.Copy(h, r)`) garde une empreinte mémoire constante quelle que soit
  la taille de l'entrée — pas de `io.ReadAll` préalable.
- **Le coût du hachage de mot de passe est voulu** : on le calibre (facteur de travail) pour rester
  cher à attaquer. Ne le baissez pas « pour aller plus vite ».
- **Réutiliser un `cipher.AEAD`** (issu de `NewGCM`) sur plusieurs messages est correct et évite de
  reconstruire le bloc AES à chaque appel.

## 🧪 À tester soi-même

Dans [`code/ch53-crypto/`](../code/ch53-crypto/) :

```bash
cd code && go test ./ch53-crypto/
```

Les tests vérifient l'empreinte SHA-256 de référence, l'aller-retour HMAC (avec rejet d'un message
et d'une signature altérés), le chiffrement/déchiffrement AES-GCM (avec **détection** d'un chiffré
modifié) et l'aller-retour TLS vérifié. Ajoutez un test qui **réutilise** volontairement un nonce
fixe et observez ce que l'on perd — puis revenez au nonce aléatoire.

---

## 📌 À retenir

- **Choisir la primitive** : pas de clé → **hachage** (`sha256`) ; clé + prouver l'intégrité →
  **HMAC** ; clé + cacher le contenu → **AEAD** (`AES-GCM`).
- **`crypto/rand` pour tout secret**, `rand.Text()` (🆕 1.24) pour un jeton ; `math/rand/v2` reste
  cantonné aux simulations.
- **Comparer les secrets en temps constant** : `hmac.Equal`, `subtle.ConstantTimeCompare`.
- **AES-GCM** chiffre **et** authentifie ; un nonce **unique** par message, tiré de `crypto/rand`.
  Fuir CBC nu.
- **TLS** : `MinVersion: tls.VersionTLS12`, vérification de certificat **active** ; `mTLS` via
  `ClientAuth`.
- **Mots de passe** : `bcrypt`/`argon2` (via `x/crypto`), salés et lents — jamais un hachage nu.

## 🔁 Pour aller plus loin

- [Ch. 47 — Sécurité & chaîne d'approvisionnement](47-securite-supply-chain.md) : la politique de
  sécurité (secrets, TLS durci, `govulncheck`, builds reproductibles).
- [Ch. 52 — Réseau bas niveau](52-reseau-net.md) : `net.Dial`/`net.Listen` sous TLS.
- [Ch. 45 — `net/http`](45-net-http.md) : HTTPS côté client et serveur.
- [Ch. 41 — Entrées/sorties & flux](41-io-flux.md) : `hash.Hash` comme `io.Writer`.
- Références : [`pkg.go.dev/crypto`](https://pkg.go.dev/crypto),
  [`pkg.go.dev/crypto/cipher`](https://pkg.go.dev/crypto/cipher),
  [`pkg.go.dev/crypto/tls`](https://pkg.go.dev/crypto/tls),
  [`pkg.go.dev/golang.org/x/crypto`](https://pkg.go.dev/golang.org/x/crypto).
