package main

import (
	"context"
	"fmt"
	"strings"

	"decryptor/provider"

	jose "github.com/go-jose/go-jose/v4"
)

// Logs are sealed as compact JWEs: a fresh AES-256-GCM content key (CEK) wrapped
// per record with RSA-OAEP-256 to the org's KMS public key. The "kid" is the key
// URI identifying which KMS unwraps the CEK, e.g. "aws-kms://arn:aws:kms:...".
var (
	allowedKeyAlgorithms     = []jose.KeyAlgorithm{jose.RSA_OAEP_256}
	allowedContentEncryption = []jose.ContentEncryption{jose.A256GCM}
)

// kmsKeyDecrypter delegates JWE key unwrapping to a provider's KMS. go-jose hands
// it the wrapped CEK and, once unwrapped, performs the A256GCM content decryption
// itself (with the protected header as AAD).
type kmsKeyDecrypter struct {
	unwrap provider.UnwrapFunc
}

func (d kmsKeyDecrypter) DecryptKey(encryptedKey []byte, _ jose.Header) ([]byte, error) {
	return d.unwrap(encryptedKey)
}

func parseJWE(value string) (*jose.JSONWebEncryption, error) {
	obj, err := jose.ParseEncrypted(strings.TrimSpace(value), allowedKeyAlgorithms, allowedContentEncryption)
	if err != nil {
		return nil, fmt.Errorf("parse JWE: %w", err)
	}
	return obj, nil
}

func decryptJWE(ctx context.Context, obj *jose.JSONWebEncryption) (string, error) {
	unwrap, err := provider.Resolve(ctx, obj.Header.KeyID)
	if err != nil {
		return "", err
	}
	return decryptWith(obj, unwrap)
}

func decryptWith(obj *jose.JSONWebEncryption, unwrap provider.UnwrapFunc) (string, error) {
	plaintext, err := obj.Decrypt(kmsKeyDecrypter{unwrap: unwrap})
	if err != nil {
		return "", fmt.Errorf("decrypt JWE: %w", err)
	}
	return string(plaintext), nil
}
