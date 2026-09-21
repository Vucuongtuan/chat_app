package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"hash"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

const (
	DefaultHKDFSize   = 32
	ChainKeySize      = 32
	MinArgon2SaltSize = 16
)

// HKDFSHA256 derives length bytes from secret using HKDF-SHA256.
func HKDFSHA256(secret, salt, info []byte, length int) ([]byte, error) {
	if len(secret) == 0 {
		return nil, ErrInvalidKey
	}
	if length <= 0 || length > 255*sha256.Size {
		return nil, fmt.Errorf("crypto: invalid hkdf length %d", length)
	}
	kdf := hkdf.New(sha256.New, secret, salt, info)
	out := make([]byte, length)
	if _, err := kdf.Read(out); err != nil {
		return nil, err
	}
	return out, nil
}

// ChainStep advances a symmetric ratchet chain (Signal-style):
//
//	messageKey   = HMAC-SHA256(chainKey, 0x01)
//	nextChainKey = HMAC-SHA256(chainKey, 0x02)
//
// The caller must discard chainKey after the step.
func ChainStep(chainKey []byte) (messageKey, nextChainKey []byte, err error) {
	if len(chainKey) != ChainKeySize {
		return nil, nil, ErrInvalidKey
	}
	mac := hmac.New(sha256.New, chainKey)
	mac.Write([]byte{0x01})
	messageKey = mac.Sum(nil)
	mac.Reset()
	mac.Write([]byte{0x02})
	nextChainKey = mac.Sum(nil)
	return messageKey, nextChainKey, nil
}

// Argon2Params are the cost parameters for Argon2id. Store them next to any
// blob derived with them so they can be raised later.
type Argon2Params struct {
	Time      uint32 // iterations
	MemoryKiB uint32 // memory in KiB
	Threads   uint8
	KeyLen    uint32
}

// DefaultArgon2Params is a reasonable starting point; benchmark on the
// weakest target device and adjust.
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{Time: 3, MemoryKiB: 64 * 1024, Threads: 4, KeyLen: 32}
}

// validate rejects zero values (no silent defaults) and absurdly large values
// (parameters may come from an untrusted backup blob, so this is a DoS guard).
func (p Argon2Params) validate() error {
	if p.Time < 1 || p.Time > 10 {
		return ErrInvalidKDFParams
	}
	if p.Threads < 1 {
		return ErrInvalidKDFParams
	}
	if p.MemoryKiB < 8*uint32(p.Threads) || p.MemoryKiB > 1<<20 {
		return ErrInvalidKDFParams
	}
	if p.KeyLen < 16 || p.KeyLen > 64 {
		return ErrInvalidKDFParams
	}
	return nil
}

// Argon2IDKey derives a key from a passphrase. Use it (not HKDF) for
// anything derived from human-chosen secrets.
func Argon2IDKey(passphrase, salt []byte, p Argon2Params) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, ErrInvalidKey
	}
	if len(salt) < MinArgon2SaltSize {
		return nil, ErrSaltTooShort
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return argon2.IDKey(passphrase, salt, p.Time, p.MemoryKiB, p.Threads, p.KeyLen), nil
}

func HashBytes(h hash.Hash, data ...[]byte) []byte {
	for _, part := range data {
		_, _ = h.Write(part)
	}
	return h.Sum(nil)
}
