package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
)

type IdentityKeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

func GenerateIdentityKeyPair() (*IdentityKeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &IdentityKeyPair{Private: priv, Public: pub}, nil
}

// Zeroize wipes the private key material.
func (k *IdentityKeyPair) Zeroize() {
	if k != nil {
		Zeroize(k.Private)
	}
}

// SignMessage signs message with priv. ed25519.Sign panics on a wrongly sized
// key, so the size is checked here.
func SignMessage(priv ed25519.PrivateKey, message []byte) ([]byte, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, ErrInvalidKey
	}
	return ed25519.Sign(priv, message), nil
}

// VerifySignature never panics: pub and sig usually come from the network,
// and ed25519.Verify panics on a wrongly sized public key.
func VerifySignature(pub ed25519.PublicKey, message, sig []byte) bool {
	if len(pub) != ed25519.PublicKeySize || len(sig) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(pub, message, sig)
}

// ConstantTimeCompare compares two byte slices in constant time (the length
// difference itself is not hidden).
func ConstantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
