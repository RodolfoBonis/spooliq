package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"testing"
)

func newKey(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func TestSecretBoxRoundTrip(t *testing.T) {
	box, err := NewSecretBox(newKey(t))
	if err != nil {
		t.Fatalf("NewSecretBox: %v", err)
	}
	if !box.Configured() {
		t.Fatal("expected configured box")
	}

	plaintext := "sk_live_super_secret_api_key_1234"
	ct, err := box.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !strings.HasPrefix(ct, "v1:") {
		t.Fatalf("ciphertext missing v1 prefix: %q", ct)
	}
	if strings.Contains(ct, plaintext) {
		t.Fatal("ciphertext leaks plaintext")
	}

	got, err := box.DecryptString(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestSecretBoxNonceIsRandom(t *testing.T) {
	box, _ := NewSecretBox(newKey(t))
	a, _ := box.EncryptString("same")
	b, _ := box.EncryptString("same")
	if a == b {
		t.Fatal("expected distinct ciphertexts for the same plaintext (random nonce)")
	}
}

func TestSecretBoxNotConfigured(t *testing.T) {
	box, err := NewSecretBox("")
	if err != nil {
		t.Fatalf("NewSecretBox(\"\"): %v", err)
	}
	if box.Configured() {
		t.Fatal("empty key must yield a not-configured box")
	}
	if _, err := box.EncryptString("x"); err != ErrNotConfigured {
		t.Fatalf("Encrypt err = %v want ErrNotConfigured", err)
	}
	if _, err := box.DecryptString("v1:abc"); err != ErrNotConfigured {
		t.Fatalf("Decrypt err = %v want ErrNotConfigured", err)
	}
}

func TestSecretBoxInvalidKey(t *testing.T) {
	// 10 bytes is not a valid AES key length in any supported encoding.
	if _, err := NewSecretBox("short"); err != ErrInvalidKey {
		t.Fatalf("err = %v want ErrInvalidKey", err)
	}
}

func TestSecretBoxTamperAndFormat(t *testing.T) {
	box, _ := NewSecretBox(newKey(t))
	ct, _ := box.EncryptString("hello")

	cases := map[string]string{
		"missing prefix":  strings.TrimPrefix(ct, "v1:"),
		"unknown version": "v2:" + strings.TrimPrefix(ct, "v1:"),
		"bad base64":      "v1:!!!not-base64!!!",
		"too short":       "v1:" + base64.RawURLEncoding.EncodeToString([]byte("tiny")),
		"tampered":        ct[:len(ct)-2] + "00",
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := box.DecryptString(bad); err != ErrInvalidCiphertext {
				t.Fatalf("err = %v want ErrInvalidCiphertext", err)
			}
		})
	}
}

func TestSecretBoxCrossKeyFails(t *testing.T) {
	box1, _ := NewSecretBox(newKey(t))
	box2, _ := NewSecretBox(newKey(t))
	ct, _ := box1.EncryptString("secret")
	if _, err := box2.DecryptString(ct); err != ErrInvalidCiphertext {
		t.Fatalf("decrypt with wrong key err = %v want ErrInvalidCiphertext", err)
	}
}
