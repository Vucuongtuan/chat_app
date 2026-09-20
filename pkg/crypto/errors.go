package crypto

import "errors"

// Sentinel errors. Callers should use errors.Is to test for them.
var (
	ErrNilSession        = errors.New("crypto: nil session")
	ErrBadRootKey        = errors.New("crypto: root key must be 32 non-zero bytes")
	ErrBadDeviceIDs      = errors.New("crypto: invalid chat or device ids")
	ErrInvalidEnvelope   = errors.New("crypto: invalid envelope")
	ErrUnknownVersion    = errors.New("crypto: unsupported envelope version")
	ErrSessionMismatch   = errors.New("crypto: envelope does not belong to this session")
	ErrReplay            = errors.New("crypto: replayed or already consumed message")
	ErrTooManySkipped    = errors.New("crypto: too many skipped messages")
	ErrDecrypt           = errors.New("crypto: decryption failed")
	ErrPlaintextTooLarge = errors.New("crypto: plaintext too large")
	ErrBadPadding        = errors.New("crypto: invalid padding")
	ErrSaltTooShort      = errors.New("crypto: salt too short")
	ErrInvalidKDFParams  = errors.New("crypto: invalid kdf parameters")
	ErrInvalidKey        = errors.New("crypto: invalid key")
	ErrCounterExhausted  = errors.New("crypto: message counter exhausted")
	ErrInvalidState      = errors.New("crypto: invalid session state")
)
