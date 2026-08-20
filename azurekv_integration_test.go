//go:build integration

// Exercises the full decrypt path (provider registry -> azurekv -> Key Vault
// Decrypt) against a real vault. Point it at an RSA key the default Azure
// credential chain may decrypt with:
//
//	TEST_AZURE_VAULT_URL=https://myvault.vault.azure.net \
//	  TEST_AZURE_KEY_NAME=formal-logs \
//	  go test -tags integration -run TestDecryptViaKeyVault ./...
package main

import (
	"context"
	"crypto/rsa"
	"math/big"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
)

func TestDecryptViaKeyVault(t *testing.T) {
	vaultURL := os.Getenv("TEST_AZURE_VAULT_URL")
	keyName := os.Getenv("TEST_AZURE_KEY_NAME")
	if vaultURL == "" || keyName == "" {
		t.Skip("set TEST_AZURE_VAULT_URL and TEST_AZURE_KEY_NAME")
	}

	ctx := context.Background()
	pub := fetchVaultPublicKey(ctx, t, vaultURL, keyName)

	// Versionless, as the Formal encryption key registers it.
	keyID := vaultURL + "/keys/" + keyName

	const plaintext = "super secret value"
	jwe := sealJWE(t, plaintext, pub, "azure-key-vault://"+keyID)

	got, err := decryptValue(ctx, jwe)
	if err != nil {
		t.Fatalf("decryptValue: %v", err)
	}
	if got != plaintext {
		t.Fatalf("got %q, want %q", got, plaintext)
	}
}

func fetchVaultPublicKey(ctx context.Context, t *testing.T, vaultURL, keyName string) *rsa.PublicKey {
	t.Helper()

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		t.Fatalf("azure credential: %v", err)
	}
	client, err := azkeys.NewClient(vaultURL, cred, nil)
	if err != nil {
		t.Fatalf("key vault client: %v", err)
	}
	resp, err := client.GetKey(ctx, keyName, "", nil)
	if err != nil {
		t.Fatalf("GetKey (may the credential read %s in %s?): %v", keyName, vaultURL, err)
	}
	if resp.Key == nil || len(resp.Key.N) == 0 || len(resp.Key.E) == 0 {
		t.Fatalf("key %s is not RSA", keyName)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(resp.Key.N),
		E: int(new(big.Int).SetBytes(resp.Key.E).Int64()),
	}
}
