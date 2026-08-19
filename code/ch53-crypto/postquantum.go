package main

import (
	"crypto/mldsa"
	"crypto/rand"
	"errors"
)

// Signatures POST-QUANTIQUES avec crypto/mldsa (🆕 Go 1.27, FIPS 204).
//
// Pourquoi maintenant, alors qu'aucun ordinateur quantique ne casse RSA ?
// Deux raisons, toutes deux liées à la DURÉE DE VIE des données :
//   - « récolter maintenant, déchiffrer plus tard » : un trafic capté
//     aujourd'hui restera déchiffrable le jour venu ;
//   - un certificat racine ou une signature de firmware doit rester valide
//     pendant des années.
//
// Go traite les deux moitiés du problème séparément :
//   - ÉCHANGE DE CLÉS : crypto/mlkem (FIPS 203, Go 1.24). Rien à faire — TLS
//     négocie déjà par défaut un mode HYBRIDE (X25519 + ML-KEM), sûr tant que
//     l'attaquant ne casse pas les deux.
//   - SIGNATURE : crypto/mldsa (FIPS 204, Go 1.27), ci-dessous.

// contexteApp est une chaîne de DOMAINE : une signature produite avec ce
// contexte ne se vérifie qu'avec le même. C'est ce qui empêche de rejouer une
// signature d'un protocole dans un autre. Donnez-lui une valeur propre à votre
// application ET à sa version.
const contexteApp = "gobook/ch53 v1"

// GenerateMLDSAKey produit une paire de clés ML-DSA-65.
//
// Trois tailles normalisées : MLDSA44 (le plus léger), MLDSA65 (recommandé par
// défaut), MLDSA87 (marge maximale). Le compromis porte sur la taille des
// signatures et des clés publiques, pas sur la vitesse.
func GenerateMLDSAKey() (*mldsa.PrivateKey, error) {
	return mldsa.GenerateKey(mldsa.MLDSA65())
}

// SignPQ signe msg. La clé privée n'est qu'une GRAINE de 32 octets : tout le
// reste (clé publique de 1952 octets incluse) s'en redérive.
func SignPQ(priv *mldsa.PrivateKey, msg []byte) ([]byte, error) {
	return priv.Sign(rand.Reader, msg, &mldsa.Options{Context: contexteApp})
}

// VerifyPQ vérifie une signature. Renvoie nil si et seulement si la signature
// est valide POUR CE MESSAGE et POUR CE CONTEXTE.
func VerifyPQ(pub *mldsa.PublicKey, msg, sig []byte) error {
	return mldsa.Verify(pub, msg, sig, &mldsa.Options{Context: contexteApp})
}

// TaillesMLDSA renvoie les tailles en octets du paramétrage retenu, pour
// mesurer le vrai coût de la migration.
//
// ⚠️ C'est LÀ que se joue la décision. Une signature Ed25519 fait 64 octets ;
// une ML-DSA-65 en fait 3309, soit ~50x. Sur une chaîne de certificats, un
// jeton porté dans un en-tête HTTP ou un protocole à petits messages, l'impact
// réseau est immédiat. Mesurez avant de migrer.
func TaillesMLDSA() (clePublique, signature int) {
	p := mldsa.MLDSA65()
	return p.PublicKeySize(), p.SignatureSize()
}

// ErrContexteInvalide est renvoyée par VerifyPQAvecContexte quand le contexte
// de vérification ne correspond pas à celui de la signature.
var ErrContexteInvalide = errors.New("contexte de signature invalide")

// VerifyPQAvecContexte permet d'observer la séparation de domaine : la même
// signature, vérifiée sous un autre contexte, est rejetée.
func VerifyPQAvecContexte(pub *mldsa.PublicKey, msg, sig []byte, ctx string) error {
	if err := mldsa.Verify(pub, msg, sig, &mldsa.Options{Context: ctx}); err != nil {
		return errors.Join(ErrContexteInvalide, err)
	}
	return nil
}
