package service

import (
	"PostmanJanai/internal/constant"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

// SecretCipher encrypts short strings (environment variable values, proxy passwords)
// using AES-256-GCM with a key derived from a compile-time constant.
//
// This is **not** a substitute for OS-level secret storage — it only prevents casual
// plaintext reads from the SQLite file. A future phase may migrate to Credential Manager / Keychain.
type SecretCipher struct {
	gcm cipher.AEAD
}

func NewSecretCipher() (*SecretCipher, error) {
	key := deriveSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretCipher{gcm: gcm}, nil
}

func deriveSecretKey() []byte {
	// Obfuscated seed — not security-through-obscurity, just avoids a naked 32-byte literal in the binary.
	const seed = "pmj:v1:local-secret-key-material:" + constant.AppName
	sum := sha256.Sum256([]byte(seed))
	return sum[:]
}

// Encrypt returns ciphertext prefixed with constant.SecretCipherPrefix.
func (c *SecretCipher) Encrypt(plain string) (string, error) {
	if c == nil {
		return "", errors.New("nil cipher")
	}
	if plain == "" {
		return "", nil
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.gcm.Seal(nonce, nonce, []byte(plain), nil)
	b64 := base64.RawStdEncoding.EncodeToString(sealed)
	return constant.SecretCipherPrefix + b64, nil
}

// Decrypt accepts either ciphertext with SecretCipherPrefix or legacy plaintext (returns as-is).
func (c *SecretCipher) Decrypt(stored string) (string, error) {
	if c == nil {
		return "", errors.New("nil cipher")
	}
	s := strings.TrimSpace(stored)
	if s == "" {
		return "", nil
	}
	if !strings.HasPrefix(s, constant.SecretCipherPrefix) {
		return s, nil
	}
	payload := strings.TrimPrefix(s, constant.SecretCipherPrefix)
	raw, err := base64.RawStdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}
	if len(raw) < c.gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce := raw[:c.gcm.NonceSize()]
	ct := raw[c.gcm.NonceSize():]
	plain, err := c.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
