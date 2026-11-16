package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterAndResolve(t *testing.T) {
	Register("test-scheme", stubProvider{key: []byte("cek")})

	unwrap, err := Resolve(t.Context(), "test-scheme://some-key-id")
	require.NoError(t, err)
	got, err := unwrap(nil)
	require.NoError(t, err)
	require.Equal(t, []byte("cek"), got)
}

func TestResolveRejectsUnsupportedScheme(t *testing.T) {
	if _, err := Resolve(context.Background(), "gcp-kms://projects/p/cryptoKeys/k"); err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestResolveRejectsMalformedURI(t *testing.T) {
	if _, err := Resolve(context.Background(), "not-a-uri"); err == nil {
		t.Fatal("expected error for malformed key URI")
	}
}

type stubProvider struct {
	key []byte
}

func (s stubProvider) Unwrapper(_ context.Context, _ string) (UnwrapFunc, error) {
	return func(_ []byte) ([]byte, error) { return s.key, nil }, nil
}
