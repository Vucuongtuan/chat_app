package crypto

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	XChaCha20Poly1305KeySize   = 32
	XChaCha20Poly1305NonceSize = 24
)

func NewRandomBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("crypto: invalid random length %d", n)
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func NewXChaCha20Poly1305Key() ([]byte, error) {
	return NewRandomBytes(XChaCha20Poly1305KeySize)
}

func SealXChaCha20Poly1305(key, nonce, plaintext, aad []byte) ([]byte, error) {
	if len(key) != XChaCha20Poly1305KeySize {
		return nil, fmt.Errorf("invalid key length: got %d want %d", len(key), XChaCha20Poly1305KeySize)
	}
	if len(nonce) != XChaCha20Poly1305NonceSize {
		return nil, fmt.Errorf("invalid nonce length: got %d want %d", len(nonce), XChaCha20Poly1305NonceSize)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	return aead.Seal(nil, nonce, plaintext, aad), nil
}

func OpenXChaCha20Poly1305(key, nonce, ciphertext, aad []byte) ([]byte, error) {
	if len(key) != XChaCha20Poly1305KeySize {
		return nil, fmt.Errorf("invalid key length: got %d want %d", len(key), XChaCha20Poly1305KeySize)
	}
	if len(nonce) != XChaCha20Poly1305NonceSize {
		return nil, fmt.Errorf("invalid nonce length: got %d want %d", len(nonce), XChaCha20Poly1305NonceSize)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, ciphertext, aad)
}
