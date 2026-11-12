package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestParsePasswordHash(t *testing.T) {
	hash, err := parsePasswordHash("")
	if err != nil || hash != nil {
		t.Fatalf("parsePasswordHash(\"\") = %x, %v, want nil, nil", hash, err)
	}

	want := sha256.Sum256([]byte("s3cret"))
	for _, raw := range []string{sha256Hex("s3cret"), strings.ToUpper(sha256Hex("s3cret")), " " + sha256Hex("s3cret") + "\n"} {
		hash, err := parsePasswordHash(raw)
		if err != nil {
			t.Fatalf("parsePasswordHash(%q) error: %v", raw, err)
		}
		if !bytes.Equal(hash, want[:]) {
			t.Fatalf("parsePasswordHash(%q) = %x, want %x", raw, hash, want)
		}
	}

	for _, raw := range []string{"s3cret", sha256Hex("s3cret")[:62], sha256Hex("s3cret") + "00", strings.Repeat("z", 64)} {
		if _, err := parsePasswordHash(raw); err == nil {
			t.Fatalf("parsePasswordHash(%q) succeeded, want an error", raw)
		}
	}
}

func TestAuthorized(t *testing.T) {
	cases := []struct {
		name     string
		expected string
		supplied string
		want     bool
	}{
		{"no password configured", "", "", true},
		{"no password configured ignores header", "", "anything", true},
		{"missing password", "s3cret", "", false},
		{"wrong password", "s3cret", "nope", false},
		{"prefix of password", "s3cret", "s3cre", false},
		{"hash instead of password", "s3cret", sha256Hex("s3cret"), false},
		{"correct password", "s3cret", "s3cret", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var expectedHash []byte
			if tc.expected != "" {
				sum := sha256.Sum256([]byte(tc.expected))
				expectedHash = sum[:]
			}
			if got := authorized(expectedHash, tc.supplied); got != tc.want {
				t.Fatalf("authorized(sha256(%q), %q) = %v, want %v", tc.expected, tc.supplied, got, tc.want)
			}
		})
	}
}

func withPassword(t *testing.T, password string) {
	t.Helper()
	previousHash, previousCORS := decryptorPasswordHash, corsHeaders
	sum := sha256.Sum256([]byte(password))
	decryptorPasswordHash = sum[:]
	corsHeaders = newCORSHeaders(defaultCORSAllowedOrigin)
	t.Cleanup(func() { decryptorPasswordHash, corsHeaders = previousHash, previousCORS })
}

func TestHTTPHandlerRejectsWrongPassword(t *testing.T) {
	withPassword(t, "s3cret")

	req := httptest.NewRequest(http.MethodPost, "/decrypt", strings.NewReader("not-a-jwe"))
	req.Header.Set(passwordHeader, "nope")
	rec := httptest.NewRecorder()
	httpHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Fatal("401 must carry CORS headers so the browser can read the status")
	}
}

func TestHTTPHandlerIgnoresAuthorizationHeader(t *testing.T) {
	withPassword(t, "s3cret")

	req := httptest.NewRequest(http.MethodPost, "/decrypt", strings.NewReader("not-a-jwe"))
	req.Header.Set("Authorization", "Bearer s3cret")
	rec := httptest.NewRecorder()
	httpHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHTTPHandlerPassesCorrectPasswordThrough(t *testing.T) {
	withPassword(t, "s3cret")

	req := httptest.NewRequest(http.MethodPost, "/decrypt", strings.NewReader("not-a-jwe"))
	req.Header.Set(passwordHeader, "s3cret")
	rec := httptest.NewRecorder()
	httpHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHTTPHandlerPreflightAllowsPasswordHeader(t *testing.T) {
	withPassword(t, "s3cret")

	req := httptest.NewRequest(http.MethodOptions, "/decrypt", nil)
	rec := httptest.NewRecorder()
	httpHandler(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, passwordHeader) {
		t.Fatalf("Access-Control-Allow-Headers = %q, want it to include %s", got, passwordHeader)
	}
}

func TestLambdaHandlerReadsPasswordHeaderCaseInsensitively(t *testing.T) {
	withPassword(t, "s3cret")

	resp, err := lambdaHandler(t.Context(), events.APIGatewayProxyRequest{
		Headers: map[string]string{"x-decryptor-password": "s3cret"},
		Body:    "not-a-jwe",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	resp, err = lambdaHandler(t.Context(), events.APIGatewayProxyRequest{Body: "not-a-jwe"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}
