# Azure Container Apps via Terraform

Deploys the decryptor as a Container App backed by an Azure Key Vault key.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) or [OpenTofu](https://opentofu.org)
- `az` authenticated against your subscription, with the `Microsoft.App` and `Microsoft.OperationalInsights` providers registered
- An existing resource group
- A key vault with RBAC authorization and an RSA key whose key operations include `encrypt` and `decrypt`
- A container image built from the repo-root `Dockerfile` and pushed to a registry the app can pull from

## Build and push the image

From the repo root:

```bash
IMAGE=<registry>.azurecr.io/decryptor:latest
az acr login -n <registry>
docker buildx build --platform linux/amd64 -t "$IMAGE" --push .
```

`az acr build` cannot build this image: its dependency scanner rejects the `FROM --platform=$BUILDPLATFORM` line.

## Deploy

```bash
cd deploy/azure-containerapps-terraform
terraform init
terraform apply \
  -var subscription_id=<subscription> \
  -var resource_group_name=<resource-group> \
  -var image="$IMAGE" \
  -var key_vault_id=/subscriptions/<s>/resourceGroups/<rg>/providers/Microsoft.KeyVault/vaults/<vault> \
  -var key_name=<key> \
  -var container_registry_id=/subscriptions/<s>/resourceGroups/<rg>/providers/Microsoft.ContainerRegistry/registries/<registry>
```

`terraform output decryptor_uri` is the endpoint to set as the encryption key's decryptor URI.

The app runs as a user-assigned managed identity granted `Key Vault Crypto User` on that one key only. Omit `container_registry_id` if the image is publicly pullable.

Ingress is external because the browser calls the decryptor directly. Pass `allowed_ip_ranges` to limit it to your VPN or office ranges, or front it with Application Gateway or Front Door.

## Access policy vaults

If the vault uses access policies rather than Azure RBAC, drop the `azurerm_role_assignment.crypto_user` resource and grant the identity a `decrypt` key permission instead:

```bash
az keyvault set-policy --name <vault> \
  --object-id "$(terraform output -raw identity_principal_id)" \
  --key-permissions decrypt
```
