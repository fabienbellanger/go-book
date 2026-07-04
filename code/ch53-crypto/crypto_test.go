package main

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

// TestHashStream vérifie que hashStream produit l'empreinte SHA-256 attendue
// (valeur de référence bien connue) et qu'elle fait 32 octets.
func TestHashStream(t *testing.T) {
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	sum, err := hashStream(strings.NewReader("abc"))
	if err != nil {
		t.Fatalf("hashStream: %v", err)
	}
	if len(sum) != 32 {
		t.Errorf("longueur = %d, attendu 32", len(sum))
	}
	if got := hex.EncodeToString(sum); got != want {
		t.Errorf("empreinte = %s, attendu %s", got, want)
	}
}

// TestEqualSecret couvre les trois cas de la comparaison en temps constant.
func TestEqualSecret(t *testing.T) {
	if !equalSecret([]byte("token"), []byte("token")) {
		t.Error("secrets identiques déclarés différents")
	}
	if equalSecret([]byte("token"), []byte("toker")) {
		t.Error("secrets différents déclarés égaux")
	}
	if equalSecret([]byte("token"), []byte("tokens")) {
		t.Error("longueurs différentes déclarées égales")
	}
}

// TestHMACRoundTrip : une signature valide passe, un message ou une signature
// altérés échouent.
func TestHMACRoundTrip(t *testing.T) {
	key, err := randomBytes(32)
	if err != nil {
		t.Fatalf("randomBytes: %v", err)
	}
	msg := []byte("virement:42")
	sig := signHMAC(key, msg)

	if !verifyHMAC(key, msg, sig) {
		t.Error("signature légitime rejetée")
	}
	if verifyHMAC(key, []byte("virement:99"), sig) {
		t.Error("message altéré accepté")
	}
	badSig := bytes.Clone(sig)
	badSig[0] ^= 0xFF
	if verifyHMAC(key, msg, badSig) {
		t.Error("signature altérée acceptée")
	}
}

// TestEncryptDecrypt : le déchiffrement rend le clair d'origine, et toute
// altération du chiffré est détectée (Open renvoie une erreur).
func TestEncryptDecrypt(t *testing.T) {
	key, err := randomBytes(32)
	if err != nil {
		t.Fatalf("randomBytes: %v", err)
	}
	plaintext := []byte("message confidentiel")

	blob, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := decrypt(key, blob)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("déchiffré = %q, attendu %q", got, plaintext)
	}

	// Un octet modifié dans le chiffré doit faire échouer l'authentification.
	tampered := bytes.Clone(blob)
	tampered[len(tampered)-1] ^= 0x01
	if _, err := decrypt(key, tampered); err == nil {
		t.Error("altération du chiffré non détectée")
	}
}

// TestNewToken vérifie que deux jetons diffèrent et ne sont pas vides.
func TestNewToken(t *testing.T) {
	a, b := newToken(), newToken()
	if a == "" || b == "" {
		t.Fatal("jeton vide")
	}
	if a == b {
		t.Error("deux jetons identiques : aléa défaillant")
	}
}

// TestTLSEcho vérifie l'aller-retour TLS vérifié sur loopback.
func TestTLSEcho(t *testing.T) {
	const msg = "hello over TLS"
	got, err := tlsEcho(msg)
	if err != nil {
		t.Fatalf("tlsEcho: %v", err)
	}
	if got != msg {
		t.Errorf("écho = %q, attendu %q", got, msg)
	}
}
