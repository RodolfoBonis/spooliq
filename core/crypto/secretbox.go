// Package crypto provides a small authenticated-encryption helper (AES-256-GCM)
// used to protect integration secrets at rest (e.g. the InfinitePay API key).
//
// Ciphertext format is versioned so the scheme can evolve: "v1:" followed by the
// URL-safe base64 (no padding) of nonce‖ciphertext. The 12-byte random nonce is
// prepended to the GCM output, which also carries the authentication tag.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

// ciphertextPrefix tags the current ciphertext scheme. A future scheme would use
// a new prefix and Decrypt would branch on it.
const ciphertextPrefix = "v1:"

// Errors returned by the SecretBox. They are intentionally generic so callers
// never leak key material or plaintext into logs when reporting a failure.
var (
	// ErrNotConfigured is returned by Encrypt/Decrypt when no key was provided.
	// Callers translate it into a 503 payments_not_configured so the app keeps
	// booting without the key.
	ErrNotConfigured = errors.New("secretbox: encryption key not configured")
	// ErrInvalidKey is returned by NewSecretBox when the key cannot be decoded to
	// a valid AES key length (16, 24 or 32 bytes).
	ErrInvalidKey = errors.New("secretbox: invalid encryption key")
	// ErrInvalidCiphertext is returned by Decrypt when the input is malformed,
	// uses an unknown version or fails authentication.
	ErrInvalidCiphertext = errors.New("secretbox: invalid ciphertext")
)

// SecretBox encrypts and decrypts short secrets with AES-GCM. The zero value is
// not usable; build one with NewSecretBox. When constructed without a key it is
// "not configured": Configured() reports false and Encrypt/Decrypt return
// ErrNotConfigured, so the application can start and only the payment endpoints
// degrade to 503.
type SecretBox struct {
	aead cipher.AEAD
}

// NewSecretBox builds a SecretBox from the raw key material. An empty key yields a
// not-configured box (no error) so the app boots without PAYMENTS_ENCRYPTION_KEY.
//
// The key is decoded, in order, as: standard base64, URL base64, hex, then raw
// bytes. Whatever decoding first yields an AES key length (16/24/32 bytes) wins;
// 32 bytes (AES-256) is the intended size. Anything else is ErrInvalidKey.
func NewSecretBox(key string) (*SecretBox, error) {
	if key == "" {
		return &SecretBox{}, nil
	}

	raw := decodeKey(key)
	if raw == nil {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, ErrInvalidKey
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrInvalidKey
	}
	return &SecretBox{aead: aead}, nil
}

// decodeKey tries several encodings and returns the first result with a valid AES
// key length, or nil when none match.
func decodeKey(key string) []byte {
	candidates := [][]byte{}
	if b, err := base64.StdEncoding.DecodeString(key); err == nil {
		candidates = append(candidates, b)
	}
	if b, err := base64.RawStdEncoding.DecodeString(key); err == nil {
		candidates = append(candidates, b)
	}
	if b, err := base64.URLEncoding.DecodeString(key); err == nil {
		candidates = append(candidates, b)
	}
	if b, err := base64.RawURLEncoding.DecodeString(key); err == nil {
		candidates = append(candidates, b)
	}
	if b, err := hex.DecodeString(key); err == nil {
		candidates = append(candidates, b)
	}
	candidates = append(candidates, []byte(key))

	for _, c := range candidates {
		switch len(c) {
		case 16, 24, 32:
			return c
		}
	}
	return nil
}

// Configured reports whether the box holds a usable key.
func (s *SecretBox) Configured() bool {
	return s != nil && s.aead != nil
}

// Encrypt seals plaintext and returns the versioned, base64url-encoded ciphertext.
// It returns ErrNotConfigured when the box has no key.
func (s *SecretBox) Encrypt(plaintext []byte) (string, error) {
	if !s.Configured() {
		return "", ErrNotConfigured
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	// Seal appends the ciphertext (and tag) to nonce, so the output is nonce‖ct.
	sealed := s.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertextPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// EncryptString is a convenience wrapper over Encrypt for string plaintext.
func (s *SecretBox) EncryptString(plaintext string) (string, error) {
	return s.Encrypt([]byte(plaintext))
}

// Decrypt opens a ciphertext produced by Encrypt. It returns ErrNotConfigured when
// the box has no key and ErrInvalidCiphertext for any malformed or tampered input.
func (s *SecretBox) Decrypt(ciphertext string) ([]byte, error) {
	if !s.Configured() {
		return nil, ErrNotConfigured
	}
	if !strings.HasPrefix(ciphertext, ciphertextPrefix) {
		return nil, ErrInvalidCiphertext
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, ciphertextPrefix))
	if err != nil {
		return nil, ErrInvalidCiphertext
	}
	nonceSize := s.aead.NonceSize()
	if len(raw) < nonceSize {
		return nil, ErrInvalidCiphertext
	}
	nonce, body := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := s.aead.Open(nil, nonce, body, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}
	return plaintext, nil
}

// DecryptString is a convenience wrapper over Decrypt returning a string.
func (s *SecretBox) DecryptString(ciphertext string) (string, error) {
	b, err := s.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
