package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"

	jose "github.com/go-jose/go-jose/v4"
)

// sealJWE produces a compact JWE the way Formal's clients do: an A256GCM content
// key wrapped via RSA-OAEP-256, with a "ver" protected header.
func sealJWE(t *testing.T, plaintext string, pub *rsa.PublicKey, keyURI string) string {
	t.Helper()
	opts := (&jose.EncrypterOptions{}).WithHeader("ver", 1)
	encrypter, err := jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{Algorithm: jose.RSA_OAEP_256, Key: pub, KeyID: keyURI},
		opts,
	)
	if err != nil {
		t.Fatalf("build encrypter: %v", err)
	}
	obj, err := encrypter.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	compact, err := obj.CompactSerialize()
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return compact
}

func TestDecryptJWE(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	const plaintext = "super secret value"
	jwe := sealJWE(t, plaintext, &key.PublicKey, "aws-kms://arn:aws:kms:us-east-1:123456789012:key/abcd")

	obj, err := parseJWE(jwe)
	if err != nil {
		t.Fatalf("parseJWE: %v", err)
	}

	// Stand in for KMS: unwrap the CEK with the matching RSA private key.
	unwrap := func(encryptedKey []byte) ([]byte, error) {
		return rsa.DecryptOAEP(sha256.New(), rand.Reader, key, encryptedKey, nil)
	}

	got, err := decryptWith(obj, unwrap)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("got %q, want %q", got, plaintext)
	}
}

// Empty log fields seal to a valid JWE whose ciphertext segment is empty
// (header.ek.iv..tag). go-jose <=v4.1.4 rejects the successful empty-plaintext
// decrypt (go-jose#43), which the handler maps to HTTP 500. This must round-trip.
func TestDecryptJWEEmptyPlaintext(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	jwe := sealJWE(t, "", &key.PublicKey, "aws-kms://arn:aws:kms:us-east-1:123456789012:key/abcd")

	if segments := strings.Split(jwe, "."); segments[3] != "" {
		t.Fatalf("expected empty ciphertext segment, got %q", segments[3])
	}

	obj, err := parseJWE(jwe)
	if err != nil {
		t.Fatalf("parseJWE: %v", err)
	}

	unwrap := func(encryptedKey []byte) ([]byte, error) {
		return rsa.DecryptOAEP(sha256.New(), rand.Reader, key, encryptedKey, nil)
	}

	got, err := decryptWith(obj, unwrap)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty string", got)
	}
}

// The KMS's own error must survive go-jose's generic one.
func TestDecryptJWEReportsUnwrapError(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	jwe := sealJWE(t, "secret", &key.PublicKey, "aws-kms://arn:aws:kms:us-east-1:123456789012:key/abcd")
	obj, err := parseJWE(jwe)
	if err != nil {
		t.Fatalf("parseJWE: %v", err)
	}

	unwrap := func([]byte) ([]byte, error) {
		return nil, errors.New("caller is not authorized to perform kms:Decrypt")
	}

	_, err = decryptWith(obj, unwrap)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("want the KMS error, got %v", err)
	}
}

func TestParseJWERejectsNonJWE(t *testing.T) {
	if _, err := parseJWE("formalencrypt:not-a-jwe"); err == nil {
		t.Fatal("expected error for non-JWE input")
	}
}
