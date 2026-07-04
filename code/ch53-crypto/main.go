// Command ch53-crypto illustre les usages courants de la cryptographie de la
// bibliothèque standard, « bien faits » :
//   - hachage en streaming (hash.Hash est un io.Writer) ;
//   - aléa cryptographique (crypto/rand) et jetons ;
//   - comparaison en temps constant (crypto/subtle, hmac.Equal) ;
//   - HMAC (intégrité + authenticité avec clé partagée) ;
//   - chiffrement authentifié AES-GCM (crypto/cipher) ;
//   - configuration TLS durcie et aller-retour sur loopback.
//
// Aucune de ces fonctions ne stocke un mot de passe : pour cela, voir le chapitre
// (bcrypt/argon2 via golang.org/x/crypto), hors bibliothèque standard.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"time"
)

// hashStream calcule l'empreinte SHA-256 d'un flux SANS le charger entièrement en
// mémoire : hash.Hash est un io.Writer, donc io.Copy l'alimente bloc par bloc
// (🔁 ch41). Le même patron fonctionne pour un fichier de plusieurs gigaoctets.
func hashStream(r io.Reader) ([]byte, error) {
	h := sha256.New() // h implémente io.Writer
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil // Sum(nil) renvoie l'empreinte sans rien y préfixer
}

// randomBytes renvoie n octets tirés du CSPRNG du système via crypto/rand.
// C'est la SEULE source acceptable pour une clé, un nonce, un sel ou un jeton.
func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil { // rand.Read ne renvoie jamais de lecture partielle
		return nil, err
	}
	return b, nil
}

// newToken renvoie un jeton aléatoire imprévisible. rand.Text() (🆕 Go 1.24) tire
// du crypto/rand et renvoie une chaîne base32 d'au moins 128 bits d'entropie :
// idéale pour un identifiant de session ou un lien de réinitialisation.
func newToken() string {
	return rand.Text()
}

// equalSecret compare deux secrets en TEMPS CONSTANT. La comparaison ne s'arrête
// pas au premier octet différent : sa durée ne dépend pas de l'endroit de la
// différence, ce qui ferme la porte aux attaques par mesure de temps.
func equalSecret(a, b []byte) bool {
	// ConstantTimeCompare renvoie 1 si égaux ET de même longueur, 0 sinon.
	return subtle.ConstantTimeCompare(a, b) == 1
}

// signHMAC produit le HMAC-SHA256 d'un message avec une clé partagée. Un HMAC
// prouve à la fois l'intégrité (le message n'a pas changé) et l'authenticité
// (il vient de quelqu'un qui connaît la clé).
func signHMAC(key, msg []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg) // hmac.Hash est aussi un io.Writer
	return mac.Sum(nil)
}

// verifyHMAC recalcule le HMAC attendu et le compare en temps constant. On
// utilise hmac.Equal, PAS bytes.Equal : comparer un MAC avec == fuit de
// l'information par timing.
func verifyHMAC(key, msg, sig []byte) bool {
	expected := signHMAC(key, msg)
	return hmac.Equal(sig, expected)
}

// encrypt chiffre plaintext avec AES-GCM, un mode AEAD (chiffrement + tag
// d'authentification). Le résultat est nonce||ciphertext||tag : on préfixe le
// nonce pour tout stocker/transmettre ensemble. La clé doit faire 16, 24 ou 32
// octets (AES-128/192/256).
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

// decrypt inverse encrypt. Open ÉCHOUE si le tag ne correspond pas : un octet
// modifié dans le chiffré (ou un mauvais nonce/clé) renvoie une erreur au lieu
// d'un clair corrompu. C'est tout l'intérêt de l'AEAD sur un mode non authentifié.
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

// clientTLSConfig renvoie une configuration TLS durcie pour un CLIENT : version
// minimale TLS 1.2 et vérification de certificat active (InsecureSkipVerify reste
// à false, sa valeur zéro). serverName est le nom attendu dans le certificat.
func clientTLSConfig(serverName string) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
		// InsecureSkipVerify: false — NE JAMAIS mettre à true en production.
	}
}

// selfSignedCert génère en mémoire un certificat auto-signé pour 127.0.0.1,
// valable une heure. Utilisé uniquement pour la démonstration/les tests : en
// production, le certificat vient d'une autorité (Let's Encrypt, PKI interne).
func selfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv, Leaf: leaf}, nil
}

// tlsEcho monte un mini serveur TLS sur loopback, y connecte un client qui
// vérifie le certificat (via un pool de confiance construit à partir du certificat
// auto-signé), envoie msg et lit l'écho. Illustre un aller-retour TLS complet et
// vérifié, sans réseau externe.
func tlsEcho(msg string) (string, error) {
	cert, err := selfSignedCert()
	if err != nil {
		return "", err
	}

	srvCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", srvCfg) // :0 = port libre choisi par l'OS
	if err != nil {
		return "", err
	}
	defer ln.Close()

	go func() { // serveur : renvoie tel quel ce qu'il reçoit
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		io.Copy(conn, conn) // écho, jusqu'à ce que le client ferme son sens écriture
	}()

	// Client : on fait confiance au certificat auto-signé en l'ajoutant à un pool
	// racine dédié (au lieu de désactiver la vérification).
	pool := x509.NewCertPool()
	pool.AddCert(cert.Leaf)
	cliCfg := clientTLSConfig("127.0.0.1")
	cliCfg.RootCAs = pool

	conn, err := tls.Dial("tcp", ln.Addr().String(), cliCfg)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if _, err := io.WriteString(conn, msg); err != nil {
		return "", err
	}
	conn.CloseWrite() // signale l'EOF au serveur pour que son io.Copy se termine

	out, err := io.ReadAll(conn)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func main() {
	// 1. Hachage d'un flux.
	sum, _ := hashStream(strings.NewReader("abc"))
	fmt.Printf("SHA-256(\"abc\") = %s\n", hex.EncodeToString(sum))

	// 2. Jeton aléatoire sûr.
	fmt.Printf("jeton            = %s\n", newToken())

	// 3. HMAC : signer puis vérifier.
	key, _ := randomBytes(32)
	msg := []byte("montant=100;compte=42")
	sig := signHMAC(key, msg)
	fmt.Printf("HMAC valide      = %v\n", verifyHMAC(key, msg, sig))
	fmt.Printf("HMAC falsifié    = %v\n", verifyHMAC(key, []byte("montant=999;compte=42"), sig))

	// 4. Chiffrement authentifié AES-GCM.
	aesKey, _ := randomBytes(32) // AES-256
	blob, _ := encrypt(aesKey, []byte("message secret"))
	clear, _ := decrypt(aesKey, blob)
	fmt.Printf("déchiffré        = %q\n", clear)

	// 5. Aller-retour TLS vérifié sur loopback.
	echo, err := tlsEcho("ping over TLS")
	if err != nil {
		fmt.Println("TLS:", err)
		return
	}
	fmt.Printf("écho TLS         = %q\n", echo)
}
