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
	"errors"
	"io"
	"strings"
)

// ciphertextPrefix tags the current ciphertext scheme.
const ciphertextPrefix = "v1:"

// keyBytes is the required key length: AES-256.
const keyBytes = 32

// Errors returned by the SecretBox. They are intentionally generic so callers never
// leak key material or plaintext into logs when reporting a failure.
var (
	// ErrNotConfigured is returned by Encrypt/Decrypt when no key was provided.
	ErrNotConfigured = errors.New("secretbox: encryption key not configured")
	// ErrInvalidKey is returned by NewSecretBox when the key is not base64 of exactly
	// 32 bytes (AES-256).
	ErrInvalidKey = errors.New("secretbox: invalid encryption key (must be base64 of 32 bytes)")
	// ErrInvalidCiphertext is returned by Decrypt when the input is malformed, uses an
	// unknown version or fails authentication.
	ErrInvalidCiphertext = errors.New("secretbox: invalid ciphertext")
)

// SecretBox encrypts and decrypts short secrets with AES-256-GCM. Build one with
// NewSecretBox. When constructed without a key it is "not configured": Configured()
// reports false and Encrypt/Decrypt return ErrNotConfigured, so the application can
// start and only the payment endpoints degrade to 503.
type SecretBox struct {
	aead cipher.AEAD
}

// NewSecretBox builds a SecretBox from the key material. An empty key yields a
// not-configured box (no error). The key MUST be base64 (standard or URL, padded or
// not) that decodes to EXACTLY 32 bytes; anything else is ErrInvalidKey.
func NewSecretBox(key string) (*SecretBox, error) {
	if key == "" {
		return &SecretBox{}, nil
	}

	raw := decodeKey(key)
	if len(raw) != keyBytes {
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

// decodeKey tries the base64 encodings and returns the first successful decode, or
// nil when none decode. Length is validated by the caller.
func decodeKey(key string) []byte {
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(key); err == nil && len(b) == keyBytes {
			return b
		}
	}
	return nil
}

// Configured reports whether the box holds a usable key.
func (s *SecretBox) Configured() bool {
	return s != nil && s.aead != nil
}

// Encrypt seals plaintext and returns the versioned, base64url-encoded ciphertext.
func (s *SecretBox) Encrypt(plaintext []byte) (string, error) {
	if !s.Configured() {
		return "", ErrNotConfigured
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
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
