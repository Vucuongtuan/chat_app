package crypto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"
	"time"
)

const (
	DefaultEnvelopeVersion = 1

	// MaxPlaintextSize is the largest plaintext a single message may carry.
	MaxPlaintextSize = 1 << 20
	// MaxEnvelopeBytes bounds the serialized envelope accepted from the network.
	MaxEnvelopeBytes = 2 << 20
	// MaxSkipPerMessage bounds how far ahead one message may jump the chain.
	MaxSkipPerMessage = 1000
	// MaxSkippedKeys bounds the number of stored skipped message keys.
	MaxSkippedKeys = 2000
	// PaddingBlockSize: plaintexts are padded up to a multiple of this size.
	PaddingBlockSize = 256

	maxIDLen    = 128
	aeadTagSize = 16
)

// Envelope is the wire format used for encrypted chat messages.
// The server stores ciphertext only; private keys remain on device.
//
// All header fields (including N and TimestampClient) are bound into the AEAD
// associated data, so tampering with any of them makes decryption fail.
type Envelope struct {
	Version           uint8  `json:"version"`
	MessageID         string `json:"message_id"`
	ChatID            string `json:"chat_id"`
	SenderDeviceID    string `json:"sender_device_id"`
	RecipientDeviceID string `json:"recipient_device_id"`
	TimestampClient   int64  `json:"timestamp_client"`
	N                 uint32 `json:"n"` // per-direction message counter
	Nonce             []byte `json:"nonce"`
	Ciphertext        []byte `json:"ciphertext"`
}

func NewEnvelope(chatID, messageID, senderID, recipientID string, n uint32, nonce, ciphertext []byte) *Envelope {
	return &Envelope{
		Version:           DefaultEnvelopeVersion,
		MessageID:         messageID,
		ChatID:            chatID,
		SenderDeviceID:    senderID,
		RecipientDeviceID: recipientID,
		TimestampClient:   time.Now().UnixMilli(),
		N:                 n,
		Nonce:             append([]byte(nil), nonce...),
		Ciphertext:        append([]byte(nil), ciphertext...),
	}
}

// Validate performs structural checks only (no crypto).
func (e *Envelope) Validate() error {
	if e == nil {
		return ErrInvalidEnvelope
	}
	if e.Version != DefaultEnvelopeVersion {
		return ErrUnknownVersion
	}
	for _, id := range []string{e.MessageID, e.ChatID, e.SenderDeviceID, e.RecipientDeviceID} {
		if id == "" || len(id) > maxIDLen {
			return fmt.Errorf("%w: bad id", ErrInvalidEnvelope)
		}
	}
	if len(e.Nonce) != XChaCha20Poly1305NonceSize {
		return fmt.Errorf("%w: bad nonce size", ErrInvalidEnvelope)
	}
	minCT := PaddingBlockSize + aeadTagSize
	maxCT := MaxPlaintextSize + PaddingBlockSize + aeadTagSize
	if len(e.Ciphertext) < minCT || len(e.Ciphertext) > maxCT {
		return fmt.Errorf("%w: bad ciphertext size", ErrInvalidEnvelope)
	}
	return nil
}

func (e *Envelope) Marshal() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(e)
}

// UnmarshalEnvelope parses untrusted input: size-limited, strict about
// unknown fields and trailing data, and structurally validated.
func UnmarshalEnvelope(data []byte) (*Envelope, error) {
	if len(data) == 0 || len(data) > MaxEnvelopeBytes {
		return nil, fmt.Errorf("%w: bad size", ErrInvalidEnvelope)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var env Envelope
	if err := dec.Decode(&env); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("%w: trailing data", ErrInvalidEnvelope)
	}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	return &env, nil
}

// NewMessageID returns a random 128-bit ID. It is deliberately independent of
// the plaintext: an ID derived from the content would let the server confirm
// guesses about what a message says.
func NewMessageID() (string, error) {
	b, err := NewRandomBytes(16)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// buildAAD builds unambiguous associated data: every variable-length field is
// length-prefixed. extra is caller-supplied application data.
func buildAAD(e *Envelope, extra []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("chatcrypto/aad/v1")
	buf.WriteByte(e.Version)

	var u32 [4]byte
	var u64 [8]byte
	binary.BigEndian.PutUint32(u32[:], e.N)
	buf.Write(u32[:])
	binary.BigEndian.PutUint64(u64[:], uint64(e.TimestampClient))
	buf.Write(u64[:])

	for _, f := range [][]byte{
		[]byte(e.ChatID),
		[]byte(e.MessageID),
		[]byte(e.SenderDeviceID),
		[]byte(e.RecipientDeviceID),
		extra,
	} {
		binary.BigEndian.PutUint32(u32[:], uint32(len(f)))
		buf.Write(u32[:])
		buf.Write(f)
	}
	return buf.Bytes()
}

// pad appends 0x80 then zeros up to a multiple of PaddingBlockSize
// (ISO/IEC 7816-4 style), so the marker is always present.
func pad(pt []byte) []byte {
	total := (len(pt)/PaddingBlockSize + 1) * PaddingBlockSize
	out := make([]byte, total)
	copy(out, pt)
	out[len(pt)] = 0x80
	return out
}

func unpad(b []byte) ([]byte, error) {
	i := len(b) - 1
	for i >= 0 && b[i] == 0 {
		i--
	}
	if i < 0 || b[i] != 0x80 {
		return nil, ErrBadPadding
	}
	return b[:i], nil
}

func finishPlain(padded []byte) ([]byte, error) {
	pt, err := unpad(padded)
	if err != nil {
		Zeroize(padded)
		return nil, err
	}
	out := append([]byte(nil), pt...)
	Zeroize(padded)
	return out, nil
}

func allZero(b []byte) bool {
	var acc byte
	for _, v := range b {
		acc |= v
	}
	return acc == 0
}

// ChatSession is a symmetric-ratchet session between two devices.
//
// It uses one chain per direction plus a per-direction counter, tolerates
// reordered and lost messages (bounded skipped-key cache), and only commits
// state after a message authenticates successfully.
//
// NOTE: this is NOT a Double Ratchet. There is no DH ratchet, so there is no
// post-compromise security, and the root key must come from a proper key
// agreement (X3DH/PQXDH). Do not ship it as the final protocol.
//
// A ChatSession is safe for concurrent use. Persist a Snapshot after every
// successful Encrypt/Decrypt (and before sending the envelope) so a crash
// cannot cause a message key to be reused.
type ChatSession struct {
	mu sync.Mutex

	chatID         string
	localDeviceID  string
	remoteDeviceID string

	sendCK, recvCK []byte
	sendN, recvN   uint32
	skipped        map[uint32][]byte
}

func validIDs(chatID, local, remote string) bool {
	for _, id := range []string{chatID, local, remote} {
		if id == "" || len(id) > maxIDLen {
			return false
		}
	}
	return local != remote
}

// NewChatSession creates a session from a 32-byte shared root key. The two
// directions are assigned by comparing device IDs (lower ID sends on the
// "low->high" chain), so both sides agree without extra coordination.
func NewChatSession(chatID, localDeviceID, remoteDeviceID string, rootKey []byte) (*ChatSession, error) {
	if len(rootKey) != 32 || allZero(rootKey) {
		return nil, ErrBadRootKey
	}
	if !validIDs(chatID, localDeviceID, remoteDeviceID) {
		return nil, ErrBadDeviceIDs
	}

	lowToHigh, err := HKDFSHA256(rootKey, nil, []byte("chatcrypto/chain/low->high"), ChainKeySize)
	if err != nil {
		return nil, err
	}
	highToLow, err := HKDFSHA256(rootKey, nil, []byte("chatcrypto/chain/high->low"), ChainKeySize)
	if err != nil {
		Zeroize(lowToHigh)
		return nil, err
	}

	s := &ChatSession{
		chatID:         chatID,
		localDeviceID:  localDeviceID,
		remoteDeviceID: remoteDeviceID,
		skipped:        make(map[uint32][]byte),
	}
	if localDeviceID < remoteDeviceID {
		s.sendCK, s.recvCK = lowToHigh, highToLow
	} else {
		s.sendCK, s.recvCK = highToLow, lowToHigh
	}
	return s, nil
}

// Encrypt encrypts plaintext for the remote device. aad is optional extra
// application data that the receiver must supply identically to Decrypt.
func (s *ChatSession) Encrypt(plaintext, aad []byte) (*Envelope, error) {
	if s == nil {
		return nil, ErrNilSession
	}
	if len(plaintext) > MaxPlaintextSize {
		return nil, ErrPlaintextTooLarge
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sendN >= math.MaxUint32-1 {
		return nil, ErrCounterExhausted
	}
	msgKey, nextCK, err := ChainStep(s.sendCK)
	if err != nil {
		return nil, err
	}
	defer Zeroize(msgKey)

	messageID, err := NewMessageID()
	if err != nil {
		Zeroize(nextCK)
		return nil, err
	}
	nonce, err := NewRandomBytes(XChaCha20Poly1305NonceSize)
	if err != nil {
		Zeroize(nextCK)
		return nil, err
	}

	env := &Envelope{
		Version:           DefaultEnvelopeVersion,
		MessageID:         messageID,
		ChatID:            s.chatID,
		SenderDeviceID:    s.localDeviceID,
		RecipientDeviceID: s.remoteDeviceID,
		TimestampClient:   time.Now().UnixMilli(),
		N:                 s.sendN,
		Nonce:             nonce,
	}

	padded := pad(plaintext)
	ct, err := SealXChaCha20Poly1305(msgKey, nonce, padded, buildAAD(env, aad))
	Zeroize(padded)
	if err != nil {
		Zeroize(nextCK)
		return nil, err
	}
	env.Ciphertext = ct

	// Commit only after success.
	Zeroize(s.sendCK)
	s.sendCK = nextCK
	s.sendN++
	return env, nil
}

// Decrypt authenticates and decrypts env. On any failure the session state is
// left unchanged, so a forged message cannot desynchronize the chain.
func (s *ChatSession) Decrypt(env *Envelope, aad []byte) ([]byte, error) {
	if s == nil {
		return nil, ErrNilSession
	}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	if env.ChatID != s.chatID ||
		env.SenderDeviceID != s.remoteDeviceID ||
		env.RecipientDeviceID != s.localDeviceID {
		return nil, ErrSessionMismatch
	}
	if env.N >= math.MaxUint32-1 {
		return nil, ErrCounterExhausted
	}
	full := buildAAD(env, aad)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Out-of-order message whose key was cached earlier.
	if key, ok := s.skipped[env.N]; ok {
		padded, err := OpenXChaCha20Poly1305(key, env.Nonce, env.Ciphertext, full)
		if err != nil {
			return nil, ErrDecrypt
		}
		Zeroize(key)
		delete(s.skipped, env.N)
		return finishPlain(padded)
	}

	if env.N < s.recvN {
		return nil, ErrReplay
	}
	if env.N-s.recvN > MaxSkipPerMessage {
		return nil, ErrTooManySkipped
	}

	// Work on a copy; nothing touches s until the message authenticates.
	ck := append([]byte(nil), s.recvCK...)
	n := s.recvN
	pending := make(map[uint32][]byte)
	fail := func(err error) ([]byte, error) {
		for _, k := range pending {
			Zeroize(k)
		}
		Zeroize(ck)
		return nil, err
	}

	for n < env.N {
		mk, next, err := ChainStep(ck)
		if err != nil {
			return fail(err)
		}
		Zeroize(ck)
		ck = next
		pending[n] = mk
		n++
	}

	mk, next, err := ChainStep(ck)
	if err != nil {
		return fail(err)
	}
	Zeroize(ck)
	padded, err := OpenXChaCha20Poly1305(mk, env.Nonce, env.Ciphertext, full)
	Zeroize(mk)
	if err != nil {
		Zeroize(next)
		return fail(ErrDecrypt)
	}

	// Commit.
	Zeroize(s.recvCK)
	s.recvCK = next
	s.recvN = n + 1
	for i, k := range pending {
		s.skipped[i] = k
	}
	s.evictSkippedLocked()
	return finishPlain(padded)
}

// evictSkippedLocked drops the oldest skipped keys beyond MaxSkippedKeys.
func (s *ChatSession) evictSkippedLocked() {
	for len(s.skipped) > MaxSkippedKeys {
		var oldest uint32
		first := true
		for n := range s.skipped {
			if first || n < oldest {
				oldest = n
				first = false
			}
		}
		Zeroize(s.skipped[oldest])
		delete(s.skipped, oldest)
	}
}

// Close wipes all key material held by the session.
func (s *ChatSession) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	Zeroize(s.sendCK)
	Zeroize(s.recvCK)
	for n, k := range s.skipped {
		Zeroize(k)
		delete(s.skipped, n)
	}
}

// SessionState is a serializable copy of a session. It contains secret keys:
// encrypt it (e.g. with the local storage key) before writing it anywhere.
type SessionState struct {
	Version        uint8             `json:"version"`
	ChatID         string            `json:"chat_id"`
	LocalDeviceID  string            `json:"local_device_id"`
	RemoteDeviceID string            `json:"remote_device_id"`
	SendCK         []byte            `json:"send_ck"`
	RecvCK         []byte            `json:"recv_ck"`
	SendN          uint32            `json:"send_n"`
	RecvN          uint32            `json:"recv_n"`
	Skipped        map[uint32][]byte `json:"skipped"`
}

// Snapshot returns a deep copy of the session state.
func (s *ChatSession) Snapshot() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()

	skipped := make(map[uint32][]byte, len(s.skipped))
	for n, k := range s.skipped {
		skipped[n] = append([]byte(nil), k...)
	}
	return SessionState{
		Version:        DefaultEnvelopeVersion,
		ChatID:         s.chatID,
		LocalDeviceID:  s.localDeviceID,
		RemoteDeviceID: s.remoteDeviceID,
		SendCK:         append([]byte(nil), s.sendCK...),
		RecvCK:         append([]byte(nil), s.recvCK...),
		SendN:          s.sendN,
		RecvN:          s.recvN,
		Skipped:        skipped,
	}
}

// RestoreChatSession rebuilds a session from a Snapshot, validating it first
// (the state may come from storage that was corrupted or tampered with).
func RestoreChatSession(st SessionState) (*ChatSession, error) {
	if st.Version != DefaultEnvelopeVersion {
		return nil, ErrUnknownVersion
	}
	if !validIDs(st.ChatID, st.LocalDeviceID, st.RemoteDeviceID) {
		return nil, ErrBadDeviceIDs
	}
	if len(st.SendCK) != ChainKeySize || len(st.RecvCK) != ChainKeySize {
		return nil, ErrInvalidState
	}
	if len(st.Skipped) > MaxSkippedKeys {
		return nil, ErrInvalidState
	}
	skipped := make(map[uint32][]byte, len(st.Skipped))
	for n, k := range st.Skipped {
		if len(k) != XChaCha20Poly1305KeySize || n >= st.RecvN {
			return nil, ErrInvalidState
		}
		skipped[n] = append([]byte(nil), k...)
	}
	return &ChatSession{
		chatID:         st.ChatID,
		localDeviceID:  st.LocalDeviceID,
		remoteDeviceID: st.RemoteDeviceID,
		sendCK:         append([]byte(nil), st.SendCK...),
		recvCK:         append([]byte(nil), st.RecvCK...),
		sendN:          st.SendN,
		recvN:          st.RecvN,
		skipped:        skipped,
	}, nil
}
