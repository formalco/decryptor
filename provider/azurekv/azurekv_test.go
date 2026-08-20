package azurekv

import (
	"context"
	"testing"

	"decryptor/provider"
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
		if err != nil {
			t.Fatalf("parseKeyURI(%q): %v", tc.keyID, err)
		}
		if vault != tc.vault || name != tc.name || version != tc.version {
			t.Fatalf("parseKeyURI(%q) = %q, %q, %q; want %q, %q, %q",
				tc.keyID, vault, name, version, tc.vault, tc.name, tc.version)
		}
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
		if _, _, _, err := parseKeyURI(keyID); err == nil {
			t.Fatalf("expected error for %q", keyID)
		}
	}
}

func TestUnwrapperRejectsMalformedKeyID(t *testing.T) {
	if _, err := (azureKV{}).Unwrapper(context.Background(), "not-a-uri"); err == nil {
		t.Fatal("expected error for malformed key ID")
	}
}

func TestResolveRegistersScheme(t *testing.T) {
	unwrap, err := provider.Resolve(context.Background(), "azure-key-vault://https://myvault.vault.azure.net/keys/mykey")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if unwrap == nil {
		t.Fatal("expected unwrap func")
	}
}
