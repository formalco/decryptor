# Decryptor

Reference decryptor for Formal's field-level log encryption. Formal encrypts sensitive log fields client-side and never holds or access the private key, so you deploy this service in your own infrastructure to let users decrypt those fields on demand from their browser.

Once deployed, use the endpoint URL as the `decryptor_uri` on a Formal encryption key.

**Note: we highly encourage making sure the endpoint is only accessible via a VPN to prevent users outside of your organization from making requests to the `/decrypt` endpoint.**

## How it works

Encrypted fields are [JWE](https://datatracker.ietf.org/doc/html/rfc7516) compact tokens (`alg=RSA-OAEP-256`, `enc=A256GCM`): a fresh AES-256-GCM content key is wrapped per record with the RSA public half of your KMS key. The JWE `kid` is a key URI of the form `<scheme>://<keyID>`, e.g. `aws-kms://arn:aws:kms:us-east-1:123456789012:key/abcd`, `gcp-kms://projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1`, or `azure-key-vault://https://myvault.vault.azure.net/keys/formal-logs`.

The Formal Console calls the decryptor from the user's browser: it `POST`s the JWE token as the raw request body to your `decryptor_uri` and reads back `{"message": "<plaintext>"}`.

For each request the decryptor:

1. parses the JWE and reads the `kid`;
2. picks a provider from the URI scheme and asks it to unwrap the content key (AWS KMS `Decrypt`, GCP KMS `AsymmetricDecrypt`, or Azure Key Vault `Decrypt`, all RSA-OAEP-256);
3. decrypts the content with the unwrapped key and returns the plaintext.

The decryptor holds no private key material; only the KMS unwrap can recover the content key.

## Deployment

The decryptor is one binary. It can run anywhere you can run a container.

The root `Dockerfile` builds an image that runs as a standalone HTTP server or a Lambda function.

The `deploy/` directory has a few examples:

- [AWS Lambda via Terraform](deploy/aws-lambda-terraform/README.md)
- [AWS Lambda via Serverless](deploy/aws-lambda-serverless/README.md)
- [GCP Cloud Run via Terraform](deploy/gcp-cloudrun-terraform/README.md)
- [Azure Container Apps via Terraform](deploy/azure-containerapps-terraform/README.md)

## Tests

`go test ./...` covers JWE parsing and decryption with a stubbed KMS unwrap.

An integration test drives the full path (provider registry through to AWS KMS `Decrypt`) against a real KMS. Point it at any KMS reachable through the default AWS credential chain, or at a local [LocalStack](https://www.localstack.cloud) KMS:

```bash
go test -tags integration -run TestDecryptViaKMS ./...
```

The endpoint defaults to `http://localhost:4566`; override it with `TEST_KMS_ENDPOINT`.

A second integration test drives the same path through Azure Key Vault. It needs an RSA key the default Azure credential chain may read and decrypt with:

```bash
TEST_AZURE_VAULT_URL=https://myvault.vault.azure.net TEST_AZURE_KEY_NAME=formal-logs \
  go test -tags integration -run TestDecryptViaKeyVault ./...
```
