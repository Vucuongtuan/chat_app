package crypto

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

// ---------- helpers ----------

func newPair(t *testing.T) (*ChatSession, *ChatSession) {
	t.Helper()
	root, err := NewRandomBytes(32)
	if err != nil {
		t.Fatalf("root key: %v", err)
	}
	a, err := NewChatSession("chat-1", "device-a", "device-b", root)
	if err != nil {
		t.Fatalf("session a: %v", err)
	}
	b, err := NewChatSession("chat-1", "device-b", "device-a", root)
	if err != nil {
		t.Fatalf("session b: %v", err)
	}
	return a, b
}

func enc(t *testing.T, s *ChatSession, msg string) *Envelope {
	t.Helper()
	env, err := s.Encrypt([]byte(msg), nil)
	if err != nil {
		t.Fatalf("encrypt %q: %v", msg, err)
	}
	return env
}

func expect(t *testing.T, s *ChatSession, env *Envelope, want string) {
	t.Helper()
	got, err := s.Decrypt(env, nil)
	if err != nil {
		t.Fatalf("decrypt (want %q): %v", want, err)
	}
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// ---------- AEAD ----------

func TestAEADRoundTrip(t *testing.T) {
	key, err := NewXChaCha20Poly1305Key()
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := NewRandomBytes(XChaCha20Poly1305NonceSize)
	if err != nil {
		t.Fatal(err)
	}
	pt := []byte("hello encrypted world")
	aad := []byte("chat:abc")

	ct, err := SealXChaCha20Poly1305(key, nonce, pt, aad)
	if err != nil {
		t.Fatal(err)
	}
	got, err := OpenXChaCha20Poly1305(key, nonce, ct, aad)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, pt) {
		t.Fatalf("got %q want %q", got, pt)
	}

	if _, err := OpenXChaCha20Poly1305(key, nonce, ct, []byte("chat:other")); err == nil {
		t.Fatal("wrong AAD must fail")
	}
	otherKey, _ := NewXChaCha20Poly1305Key()
	if _, err := OpenXChaCha20Poly1305(otherKey, nonce, ct, aad); err == nil {
		t.Fatal("wrong key must fail")
	}
}

func TestAEADRejectsBadSizes(t *testing.T) {
	if _, err := SealXChaCha20Poly1305(make([]byte, 5), make([]byte, 24), nil, nil); err == nil {
		t.Fatal("short key must fail")
	}
	if _, err := SealXChaCha20Poly1305(make([]byte, 32), make([]byte, 5), nil, nil); err == nil {
		t.Fatal("short nonce must fail")
	}
	if _, err := NewRandomBytes(-1); err == nil {
		t.Fatal("negative length must fail")
	}
}

// ---------- identity ----------

func TestSignVerify(t *testing.T) {
	kp, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("signed prekey")
	sig, err := SignMessage(kp.Private, msg)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifySignature(kp.Public, msg, sig) {
		t.Fatal("valid signature rejected")
	}
	if VerifySignature(kp.Public, []byte("other"), sig) {
		t.Fatal("signature over different message accepted")
	}
}

func TestSignVerifyBadSizesDoNotPanic(t *testing.T) {
	kp, _ := GenerateIdentityKeyPair()
	msg := []byte("m")
	sig, _ := SignMessage(kp.Private, msg)

	if VerifySignature(ed25519.PublicKey{1, 2, 3}, msg, sig) {
		t.Fatal("short public key accepted")
	}
	if VerifySignature(kp.Public, msg, sig[:10]) {
		t.Fatal("short signature accepted")
	}
	if _, err := SignMessage(ed25519.PrivateKey{1, 2, 3}, msg); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("want ErrInvalidKey, got %v", err)
	}
}

// ---------- KDF ----------

// RFC 5869, Appendix A.1.
func TestHKDFRFC5869Vector(t *testing.T) {
	ikm := bytes.Repeat([]byte{0x0b}, 22)
	salt, _ := hex.DecodeString("000102030405060708090a0b0c")
	info, _ := hex.DecodeString("f0f1f2f3f4f5f6f7f8f9")
	want := "3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865"

	got, err := HKDFSHA256(ikm, salt, info, 42)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(got) != want {
		t.Fatalf("got %x want %s", got, want)
	}
}

func TestHKDFRejectsBadInput(t *testing.T) {
	if _, err := HKDFSHA256(nil, nil, nil, 32); err == nil {
		t.Fatal("empty secret must fail")
	}
	if _, err := HKDFSHA256([]byte("k"), nil, nil, 0); err == nil {
		t.Fatal("zero length must fail")
	}
}

func TestChainStep(t *testing.T) {
	ck := bytes.Repeat([]byte{7}, ChainKeySize)
	mk1, next1, err := ChainStep(ck)
	if err != nil {
		t.Fatal(err)
	}
	mk2, next2, _ := ChainStep(ck)
	if !bytes.Equal(mk1, mk2) || !bytes.Equal(next1, next2) {
		t.Fatal("ChainStep must be deterministic")
	}
	if bytes.Equal(mk1, next1) {
		t.Fatal("message key and next chain key must differ")
	}
	if _, _, err := ChainStep(make([]byte, 5)); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("want ErrInvalidKey, got %v", err)
	}
}

func TestArgon2Validation(t *testing.T) {
	salt := bytes.Repeat([]byte{1}, 16)
	small := Argon2Params{Time: 1, MemoryKiB: 64, Threads: 1, KeyLen: 32}

	k1, err := Argon2IDKey([]byte("pass"), salt, small)
	if err != nil {
		t.Fatal(err)
	}
	k2, _ := Argon2IDKey([]byte("pass"), salt, small)
	if !bytes.Equal(k1, k2) {
		t.Fatal("argon2 must be deterministic")
	}
	k3, _ := Argon2IDKey([]byte("pass"), bytes.Repeat([]byte{2}, 16), small)
	if bytes.Equal(k1, k3) {
		t.Fatal("different salt must give different key")
	}

	if _, err := Argon2IDKey([]byte("pass"), salt[:8], small); !errors.Is(err, ErrSaltTooShort) {
		t.Fatalf("want ErrSaltTooShort, got %v", err)
	}
	if _, err := Argon2IDKey([]byte("pass"), salt, Argon2Params{}); !errors.Is(err, ErrInvalidKDFParams) {
		t.Fatalf("zero params must be rejected, got %v", err)
	}
	huge := small
	huge.MemoryKiB = 1 << 30
	if _, err := Argon2IDKey([]byte("pass"), salt, huge); !errors.Is(err, ErrInvalidKDFParams) {
		t.Fatalf("huge memory must be rejected, got %v", err)
	}
	if _, err := Argon2IDKey(nil, salt, small); err == nil {
		t.Fatal("empty passphrase must fail")
	}
}

// ---------- session: construction ----------

func TestNewChatSessionValidation(t *testing.T) {
	good, _ := NewRandomBytes(32)

	if _, err := NewChatSession("c", "a", "b", good[:31]); !errors.Is(err, ErrBadRootKey) {
		t.Fatalf("short root key: got %v", err)
	}
	if _, err := NewChatSession("c", "a", "b", make([]byte, 32)); !errors.Is(err, ErrBadRootKey) {
		t.Fatalf("all-zero root key: got %v", err)
	}
	if _, err := NewChatSession("c", "a", "a", good); !errors.Is(err, ErrBadDeviceIDs) {
		t.Fatalf("same device ids: got %v", err)
	}
	if _, err := NewChatSession("", "a", "b", good); !errors.Is(err, ErrBadDeviceIDs) {
		t.Fatalf("empty chat id: got %v", err)
	}
}

// ---------- session: happy paths ----------

func TestSessionRoundTripBothDirections(t *testing.T) {
	a, b := newPair(t)
	expect(t, b, enc(t, a, "a1"), "a1")
	expect(t, a, enc(t, b, "b1"), "b1")
	expect(t, b, enc(t, a, "a2"), "a2")
	expect(t, a, enc(t, b, "b2"), "b2")
}

func TestSessionSimultaneousSend(t *testing.T) {
	a, b := newPair(t)
	fromA := enc(t, a, "from a")
	fromB := enc(t, b, "from b") // sent before b has seen fromA
	expect(t, b, fromA, "from a")
	expect(t, a, fromB, "from b")
}

func TestSessionEmptyPlaintext(t *testing.T) {
	a, b := newPair(t)
	got, err := b.Decrypt(enc(t, a, ""), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty plaintext, got %q", got)
	}
}

func TestSessionOutOfOrder(t *testing.T) {
	a, b := newPair(t)
	m0 := enc(t, a, "m0")
	m1 := enc(t, a, "m1")
	m2 := enc(t, a, "m2")
	m3 := enc(t, a, "m3")

	expect(t, b, m2, "m2")
	expect(t, b, m0, "m0")
	expect(t, b, m3, "m3")
	expect(t, b, m1, "m1")

	if _, err := b.Decrypt(m2, nil); !errors.Is(err, ErrReplay) {
		t.Fatalf("replay after out-of-order: got %v", err)
	}
}

func TestSessionLostMessage(t *testing.T) {
	a, b := newPair(t)
	_ = enc(t, a, "lost")
	m1 := enc(t, a, "m1")
	expect(t, b, m1, "m1")
}

// ---------- session: attacks and failures ----------

func TestSessionReplayRejected(t *testing.T) {
	a, b := newPair(t)
	env := enc(t, a, "once")
	expect(t, b, env, "once")
	if _, err := b.Decrypt(env, nil); !errors.Is(err, ErrReplay) {
		t.Fatalf("want ErrReplay, got %v", err)
	}
}

func TestSessionTamperCiphertextFailsAndKeepsState(t *testing.T) {
	a, b := newPair(t)
	env := enc(t, a, "secret message")

	bad := *env
	bad.Ciphertext = append([]byte(nil), env.Ciphertext...)
	bad.Ciphertext[0] ^= 0xFF
	if _, err := b.Decrypt(&bad, nil); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("want ErrDecrypt, got %v", err)
	}

	// A failed forgery must not desynchronize the session.
	expect(t, b, env, "secret message")
}

func TestSessionHeaderTamperFails(t *testing.T) {
	a, b := newPair(t)
	env := enc(t, a, "hdr")

	cases := map[string]func(e *Envelope){
		"message id": func(e *Envelope) { e.MessageID = "forged" },
		"timestamp":  func(e *Envelope) { e.TimestampClient++ },
		"counter":    func(e *Envelope) { e.N++ },
	}
	for name, mutate := range cases {
		bad := *env
		mutate(&bad)
		if _, err := b.Decrypt(&bad, nil); err == nil {
			t.Fatalf("tampering with %s must fail", name)
		}
	}
	expect(t, b, env, "hdr")
}

func TestSessionMismatchRejected(t *testing.T) {
	a, b := newPair(t)
	env := enc(t, a, "x")

	wrongChat := *env
	wrongChat.ChatID = "chat-2"
	if _, err := b.Decrypt(&wrongChat, nil); !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("wrong chat: got %v", err)
	}
	wrongRecipient := *env
	wrongRecipient.RecipientDeviceID = "device-z"
	if _, err := b.Decrypt(&wrongRecipient, nil); !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("wrong recipient: got %v", err)
	}
	wrongSender := *env
	wrongSender.SenderDeviceID = "device-z"
	if _, err := b.Decrypt(&wrongSender, nil); !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("wrong sender: got %v", err)
	}
	// A session must not decrypt its own outgoing messages.
	if _, err := a.Decrypt(env, nil); !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("own message: got %v", err)
	}
}

func TestSessionExtraAADBound(t *testing.T) {
	a, b := newPair(t)
	env, err := a.Encrypt([]byte("bound"), []byte("ctx-1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Decrypt(env, []byte("ctx-2")); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("wrong extra AAD: got %v", err)
	}
	got, err := b.Decrypt(env, []byte("ctx-1"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "bound" {
		t.Fatalf("got %q", got)
	}
}

func TestSessionTooManySkipped(t *testing.T) {
	a, b := newPair(t)
	var envs []*Envelope
	for i := 0; i < MaxSkipPerMessage+2; i++ {
		envs = append(envs, enc(t, a, "m"))
	}
	if _, err := b.Decrypt(envs[len(envs)-1], nil); !errors.Is(err, ErrTooManySkipped) {
		t.Fatalf("want ErrTooManySkipped, got %v", err)
	}
	// State must be intact.
	expect(t, b, envs[0], "m")
}

func TestSessionRejectsOversizePlaintext(t *testing.T) {
	a, _ := newPair(t)
	if _, err := a.Encrypt(make([]byte, MaxPlaintextSize+1), nil); !errors.Is(err, ErrPlaintextTooLarge) {
		t.Fatalf("want ErrPlaintextTooLarge, got %v", err)
	}
}

func TestSessionNilSafety(t *testing.T) {
	var s *ChatSession
	if _, err := s.Encrypt([]byte("x"), nil); !errors.Is(err, ErrNilSession) {
		t.Fatalf("encrypt on nil: %v", err)
	}
	if _, err := s.Decrypt(&Envelope{}, nil); !errors.Is(err, ErrNilSession) {
		t.Fatalf("decrypt on nil: %v", err)
	}
	_, b := newPair(t)
	if _, err := b.Decrypt(nil, nil); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("nil envelope: %v", err)
	}
}

// ---------- session: privacy properties ----------

func TestMessageIDIndependentOfPlaintext(t *testing.T) {
	a, _ := newPair(t)
	e1 := enc(t, a, "same text")
	e2 := enc(t, a, "same text")
	if e1.MessageID == e2.MessageID {
		t.Fatal("message ids must be unique even for identical plaintext")
	}
}

func TestPaddingHidesShortLengths(t *testing.T) {
	a, _ := newPair(t)
	short := enc(t, a, "a")
	longer := enc(t, a, "a considerably longer message, but still under one block")
	if len(short.Ciphertext) != len(longer.Ciphertext) {
		t.Fatalf("ciphertext lengths leak plaintext length: %d vs %d",
			len(short.Ciphertext), len(longer.Ciphertext))
	}
}

// ---------- session: concurrency and persistence ----------

func TestConcurrentEncryptUsesUniqueCounters(t *testing.T) {
	a, _ := newPair(t)
	const workers, per = 8, 50

	var mu sync.Mutex
	seen := make(map[uint32]bool)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < per; i++ {
				env, err := a.Encrypt([]byte("x"), nil)
				if err != nil {
					t.Errorf("encrypt: %v", err)
					return
				}
				mu.Lock()
				if seen[env.N] {
					t.Errorf("duplicate counter %d", env.N)
				}
				seen[env.N] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(seen) != workers*per {
		t.Fatalf("got %d unique counters want %d", len(seen), workers*per)
	}
}

func TestSnapshotRestoreRoundTrip(t *testing.T) {
	a, b := newPair(t)
	m0 := enc(t, a, "m0")
	m1 := enc(t, a, "m1")
	m2 := enc(t, a, "m2")

	expect(t, b, m1, "m1") // skips m0, so the snapshot holds a skipped key

	raw, err := json.Marshal(b.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	var st SessionState
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreChatSession(st)
	if err != nil {
		t.Fatal(err)
	}

	expect(t, restored, m0, "m0")
	expect(t, restored, m2, "m2")
	expect(t, a, enc(t, restored, "reply"), "reply")
}

func TestRestoreRejectsBadState(t *testing.T) {
	a, _ := newPair(t)
	good := a.Snapshot()

	bad := good
	bad.SendCK = good.SendCK[:5]
	if _, err := RestoreChatSession(bad); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("short chain key: got %v", err)
	}

	bad = good
	bad.Version = 99
	if _, err := RestoreChatSession(bad); !errors.Is(err, ErrUnknownVersion) {
		t.Fatalf("bad version: got %v", err)
	}

	bad = good
	bad.Skipped = map[uint32][]byte{5: make([]byte, 32)} // 5 >= RecvN (0)
	if _, err := RestoreChatSession(bad); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("skipped key ahead of counter: got %v", err)
	}
}

// ---------- envelope parsing ----------

func TestEnvelopeMarshalRoundTrip(t *testing.T) {
	a, b := newPair(t)
	env := enc(t, a, "over the wire")

	raw, err := env.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := UnmarshalEnvelope(raw)
	if err != nil {
		t.Fatal(err)
	}
	expect(t, b, parsed, "over the wire")
}

func TestUnmarshalEnvelopeRejectsGarbage(t *testing.T) {
	a, _ := newPair(t)
	valid, err := enc(t, a, "x").Marshal()
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string][]byte{
		"empty":         nil,
		"not json":      []byte("\x00\x01garbage"),
		"unknown field": []byte(`{"version":1,"bogus":1}`),
		"trailing data": append(append([]byte(nil), valid...), []byte(`{}`)...),
		"oversize":      []byte(strings.Repeat("a", MaxEnvelopeBytes+1)),
		"bad version":   []byte(`{"version":9}`),
		"missing ids":   []byte(`{"version":1}`),
	}
	for name, data := range cases {
		if _, err := UnmarshalEnvelope(data); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func FuzzUnmarshalEnvelope(f *testing.F) {
	f.Add([]byte(`{"version":1}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		// Must never panic; errors are fine.
		_, _ = UnmarshalEnvelope(data)
	})
}
