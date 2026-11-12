package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
)

const passwordHeader = "X-Decryptor-Password"

var decryptorPasswordHash []byte

func parsePasswordHash(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	hash, err := hex.DecodeString(raw)
	if err != nil || len(hash) != sha256.Size {
		return nil, errors.New("DECRYPTOR_PASSWORD_SHA256 must be the hex-encoded SHA-256 of the password (64 hex characters), not the password itself")
	}
	return hash, nil
}

func authorized(expectedHash []byte, supplied string) bool {
	if len(expectedHash) == 0 {
		return true
	}
	got := sha256.Sum256([]byte(supplied))
	return subtle.ConstantTimeCompare(got[:], expectedHash) == 1
}
