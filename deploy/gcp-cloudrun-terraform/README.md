# GCP Cloud Run via Terraform

Deploys the decryptor as a Cloud Run service backed by a Google Cloud KMS key.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) or [OpenTofu](https://opentofu.org)
- `gcloud` authenticated against your project, with Cloud Run, Artifact Registry, and Cloud KMS APIs enabled
- An asymmetric KMS key with purpose `ASYMMETRIC_DECRYPT` and an `RSA_DECRYPT_OAEP_*_SHA256` algorithm
- A container image built from the repo-root `Dockerfile` and pushed to a registry the project can pull from

## Build and push the image

From the repo root:

```bash
IMAGE=<region>-docker.pkg.dev/<project>/<repo>/decryptor:latest
docker build --platform linux/amd64 -t "$IMAGE" .
docker push "$IMAGE"
```

## Deploy

```bash
cd deploy/gcp-cloudrun-terraform
terraform init
terraform apply \
  -var project_id=<project> \
  -var image="$IMAGE" \
  -var kms_crypto_key_id=projects/<p>/locations/<l>/keyRings/<r>/cryptoKeys/<k>
```

For a self-hosted control plane, also pass `-var formal_control_plane_url=https://<your console host>`. It defaults to `https://app.formal.ai`.

`terraform output decryptor_uri` is the endpoint to set as the encryption key's decryptor URI.

The service account is granted `roles/cloudkms.cryptoKeyDecrypter` on that one key only. The service is publicly invocable by default because the browser calls it directly; restrict ingress or front it with IAP/a VPN to limit exposure.
