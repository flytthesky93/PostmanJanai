package service

import (
	"PostmanJanai/internal/constant"
	"strings"
	"testing"
)

func TestSecretCipher_RoundTrip(t *testing.T) {
	c, err := NewSecretCipher()
	if err != nil {
		t.Fatal(err)
	}
	plain := "super-secret-token-123"
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, constant.SecretCipherPrefix) {
		t.Fatalf("expected prefix, got %q", enc)
	}
	got, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if got != plain {
		t.Fatalf("Decrypt = %q, want %q", got, plain)
	}
}

func TestSecretCipher_PlaintextPassthrough(t *testing.T) {
	c, err := NewSecretCipher()
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Decrypt("not-encrypted-yet")
	if err != nil {
		t.Fatal(err)
	}
	if got != "not-encrypted-yet" {
		t.Fatalf("got %q", got)
	}
}
