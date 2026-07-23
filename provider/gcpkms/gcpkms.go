// Package gcpkms unwraps JWE content keys with Google Cloud KMS for the
// "gcp-kms" scheme.
package gcpkms

import (
	"context"
	"fmt"
	"sync"

	"decryptor/provider"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
)

const scheme = "gcp-kms"

func init() {
	provider.Register(scheme, gcpKMS{})
}

// gcpKMS expects keyID to be the full crypto key version resource name; the key
// must use an RSA_DECRYPT_OAEP_*_SHA256 algorithm to match the JWE wrap.
type gcpKMS struct{}

var (
	clientOnce sync.Once
	client     *kms.KeyManagementClient
	clientErr  error
)

// Reuse the gRPC client across requests; per-decrypt connection setup is costly.
func keyManagementClient() (*kms.KeyManagementClient, error) {
	clientOnce.Do(func() {
		client, clientErr = kms.NewKeyManagementClient(context.Background())
	})
	return client, clientErr
}

func (gcpKMS) Unwrapper(ctx context.Context, keyID string) (provider.UnwrapFunc, error) {
	c, err := keyManagementClient()
	if err != nil {
		return nil, fmt.Errorf("create GCP KMS client: %w", err)
	}
	return func(encryptedKey []byte) ([]byte, error) {
		resp, err := c.AsymmetricDecrypt(ctx, &kmspb.AsymmetricDecryptRequest{
			Name:       keyID,
			Ciphertext: encryptedKey,
		})
		if err != nil {
			return nil, fmt.Errorf("unwrap content key via GCP KMS: %w", err)
		}
		return resp.Plaintext, nil
	}, nil
}
