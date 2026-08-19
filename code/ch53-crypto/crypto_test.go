package main

import (
	"bytes"
	"crypto/mldsa"
	"encoding/hex"
	"errors"
	"slices"
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

// --- crypto/mldsa : signatures post-quantiques (🆕 Go 1.27) ---

// TestMLDSARoundTrip : une signature valide se vérifie, une altération est
// détectée.
func TestMLDSARoundTrip(t *testing.T) {
	priv, err := GenerateMLDSAKey()
	if err != nil {
		t.Fatalf("GenerateMLDSAKey: %v", err)
	}
	msg := []byte("virement de 100 EUR vers le compte 42")

	sig, err := SignPQ(priv, msg)
	if err != nil {
		t.Fatalf("SignPQ: %v", err)
	}
	pub := priv.PublicKey()

	if err := VerifyPQ(pub, msg, sig); err != nil {
		t.Errorf("VerifyPQ sur un message intact : %v", err)
	}
	if err := VerifyPQ(pub, []byte("virement de 900 EUR vers le compte 42"), sig); err == nil {
		t.Error("VerifyPQ aurait dû rejeter un message altéré")
	}
	// Un seul bit retourné dans la signature suffit.
	altérée := slices.Clone(sig)
	altérée[0] ^= 1
	if err := VerifyPQ(pub, msg, altérée); err == nil {
		t.Error("VerifyPQ aurait dû rejeter une signature altérée")
	}
}

// TestMLDSASéparationDeDomaine : le champ Options.Context isole les usages.
// Une signature émise pour une application ne vaut pas pour une autre.
func TestMLDSASéparationDeDomaine(t *testing.T) {
	priv, err := GenerateMLDSAKey()
	if err != nil {
		t.Fatalf("GenerateMLDSAKey: %v", err)
	}
	msg := []byte("jeton de session")
	sig, err := SignPQ(priv, msg)
	if err != nil {
		t.Fatalf("SignPQ: %v", err)
	}
	pub := priv.PublicKey()

	if err := VerifyPQAvecContexte(pub, msg, sig, contexteApp); err != nil {
		t.Errorf("même contexte : %v", err)
	}
	err = VerifyPQAvecContexte(pub, msg, sig, "une-autre-application")
	if !errors.Is(err, ErrContexteInvalide) {
		t.Errorf("contexte différent : err = %v ; attendu ErrContexteInvalide", err)
	}
}

// TestMLDSATailles fige le VRAI coût de la migration : ~50x la taille d'une
// signature Ed25519. C'est le critère de décision, pas la vitesse.
func TestMLDSATailles(t *testing.T) {
	clePublique, signature := TaillesMLDSA()

	if clePublique != 1952 {
		t.Errorf("clé publique ML-DSA-65 = %d octets ; attendu 1952", clePublique)
	}
	if signature != 3309 {
		t.Errorf("signature ML-DSA-65 = %d octets ; attendu 3309", signature)
	}

	// La clé PRIVÉE, elle, n'est qu'une graine de 32 octets : tout le reste
	// s'en redérive.
	priv, err := GenerateMLDSAKey()
	if err != nil {
		t.Fatalf("GenerateMLDSAKey: %v", err)
	}
	if got := len(priv.Bytes()); got != mldsa.PrivateKeySize {
		t.Errorf("graine privée = %d octets ; attendu %d", got, mldsa.PrivateKeySize)
	}

	// Mise en perspective : une signature Ed25519 fait 64 octets.
	if ratio := signature / 64; ratio < 40 {
		t.Errorf("ratio ML-DSA/Ed25519 = %d ; attendu ~50", ratio)
	}
}
