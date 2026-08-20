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
// URI identifying which KMS unwraps the CEK, e.g. "aws-kms://arn:aws:kms:..." or
// "azure-key-vault://https://{vault}.vault.azure.net/keys/{name}".
var (
	allowedKeyAlgorithms     = []jose.KeyAlgorithm{jose.RSA_OAEP_256}
	allowedContentEncryption = []jose.ContentEncryption{jose.A256GCM}
)

// kmsKeyDecrypter delegates JWE key unwrapping to a provider's KMS; go-jose runs
// the A256GCM content decryption itself.
type kmsKeyDecrypter struct {
	unwrap provider.UnwrapFunc
	// go-jose reports any unwrap failure as a bare "error in cryptographic
	// primitive", so keep the cause to report instead.
	err error
}

func (d *kmsKeyDecrypter) DecryptKey(encryptedKey []byte, _ jose.Header) ([]byte, error) {
	key, err := d.unwrap(encryptedKey)
	d.err = err
	return key, err
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
	decrypter := &kmsKeyDecrypter{unwrap: unwrap}
	plaintext, err := obj.Decrypt(decrypter)
	if err != nil {
		if decrypter.err != nil {
			err = decrypter.err
		}
		return "", fmt.Errorf("decrypt JWE: %w", err)
	}
	return string(plaintext), nil
}
