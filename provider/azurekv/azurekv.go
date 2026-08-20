// Package azurekv unwraps JWE content keys with Azure Key Vault for the
// "azure-key-vault" scheme.
package azurekv

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"decryptor/provider"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
)

const scheme = "azure-key-vault"

// The kid is caller-supplied and we hand our managed identity's token to
// whatever host it names. https://aka.ms/azsdk/blog/vault-uri
var vaultHostSuffixes = []string{
	".vault.azure.net",
	".vault.azure.cn",
	".vault.usgovcloudapi.net",
	".vault.microsoftazure.de",
	".managedhsm.azure.net",
	".managedhsm.azure.cn",
	".managedhsm.usgovcloudapi.net",
	".managedhsm.microsoftazure.de",
}

func init() {
	provider.Register(scheme, azureKV{})
}

// azureKV expects keyID to be a Key Vault key URI; an omitted version resolves
// to the key's current version, and the key must allow Decrypt with RSA-OAEP-256.
type azureKV struct{}

var (
	credOnce sync.Once
	cred     *azidentity.DefaultAzureCredential
	credErr  error

	clientsMu sync.Mutex
	clients   = map[string]*azkeys.Client{}
)

func defaultCredential() (*azidentity.DefaultAzureCredential, error) {
	credOnce.Do(func() {
		cred, credErr = azidentity.NewDefaultAzureCredential(nil)
	})
	return cred, credErr
}

func keyClient(vaultURL string) (*azkeys.Client, error) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if c, ok := clients[vaultURL]; ok {
		return c, nil
	}
	azCred, err := defaultCredential()
	if err != nil {
		return nil, fmt.Errorf("resolve Azure credential: %w", err)
	}
	c, err := azkeys.NewClient(vaultURL, azCred, nil)
	if err != nil {
		return nil, fmt.Errorf("create Azure Key Vault client for %s: %w", vaultURL, err)
	}
	clients[vaultURL] = c
	return c, nil
}

func (azureKV) Unwrapper(ctx context.Context, keyID string) (provider.UnwrapFunc, error) {
	vaultURL, keyName, keyVersion, err := parseKeyURI(keyID)
	if err != nil {
		return nil, err
	}
	c, err := keyClient(vaultURL)
	if err != nil {
		return nil, err
	}
	algorithm := azkeys.EncryptionAlgorithmRSAOAEP256
	return func(encryptedKey []byte) ([]byte, error) {
		out, err := c.Decrypt(ctx, keyName, keyVersion, azkeys.KeyOperationParameters{
			Algorithm: &algorithm,
			Value:     encryptedKey,
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("unwrap content key via Azure Key Vault: %w", err)
		}
		return out.Result, nil
	}, nil
}

// parseKeyURI accepts https://{vault}/keys/{name}[/{version}].
func parseKeyURI(keyID string) (vaultURL, keyName, keyVersion string, err error) {
	badURI := fmt.Errorf("azure key id must be a Key Vault key URI (https://{vault}.vault.azure.net/keys/{name}), got %q", keyID)

	u, err := url.Parse(keyID)
	if err != nil || u.Scheme != "https" || !isVaultHost(u.Hostname()) {
		return "", "", "", badURI
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || len(parts) > 3 || parts[0] != "keys" || parts[1] == "" {
		return "", "", "", badURI
	}
	if len(parts) == 3 {
		keyVersion = parts[2]
	}
	return u.Scheme + "://" + u.Host, parts[1], keyVersion, nil
}

func isVaultHost(host string) bool {
	host = strings.ToLower(host)
	for _, suffix := range vaultHostSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
