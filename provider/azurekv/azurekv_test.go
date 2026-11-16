package azurekv

import (
	"testing"

	"decryptor/provider"

	"github.com/stretchr/testify/require"
)

func TestParseKeyURI(t *testing.T) {
	for _, tc := range []struct {
		keyID   string
		vault   string
		name    string
		version string
	}{
		{
			keyID:   "https://myvault.vault.azure.net/keys/mykey/abc123",
			vault:   "https://myvault.vault.azure.net",
			name:    "mykey",
			version: "abc123",
		},
		{
			keyID: "https://myvault.vault.azure.net/keys/mykey",
			vault: "https://myvault.vault.azure.net",
			name:  "mykey",
		},
		{
			keyID: "https://myhsm.managedhsm.azure.net/keys/mykey",
			vault: "https://myhsm.managedhsm.azure.net",
			name:  "mykey",
		},
	} {
		vault, name, version, err := parseKeyURI(tc.keyID)
		require.NoError(t, err, tc.keyID)
		require.Equal(t, []string{tc.vault, tc.name, tc.version}, []string{vault, name, version}, tc.keyID)
	}
}

func TestParseKeyURIRejectsMalformed(t *testing.T) {
	for _, keyID := range []string{
		"not-a-uri",
		"https://myvault.vault.azure.net/secrets/mykey",
		"https://myvault.vault.azure.net/keys/",
		"https://myvault.vault.azure.net/",
		"https://myvault.vault.azure.net/keys/mykey/abc123/extra",
		// A kid is attacker-controlled: it must not send our token elsewhere.
		"http://myvault.vault.azure.net/keys/mykey",
		"https://myvault.vault.azure.net.attacker.example/keys/mykey",
		"https://attacker.example/keys/mykey",
		"https://vault.azure.net/keys/mykey",
	} {
		_, _, _, err := parseKeyURI(keyID)
		require.Error(t, err, keyID)
	}
}

func TestUnwrapperRejectsMalformedKeyID(t *testing.T) {
	_, err := (azureKV{}).Unwrapper(t.Context(), "not-a-uri")
	require.Error(t, err)
}

func TestResolveRegistersScheme(t *testing.T) {
	unwrap, err := provider.Resolve(t.Context(), "azure-key-vault://https://myvault.vault.azure.net/keys/mykey")
	require.NoError(t, err)
	require.NotNil(t, unwrap)
}
