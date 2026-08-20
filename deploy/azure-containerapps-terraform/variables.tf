variable "subscription_id" {
  description = "Azure subscription to deploy the decryptor into."
  type        = string
}

variable "resource_group_name" {
  description = "Existing resource group to deploy the decryptor into."
  type        = string
}

variable "location" {
  description = "Azure region for the container app."
  type        = string
  default     = "eastus"
}

variable "image" {
  description = "Container image for the decryptor (build with the repo-root Dockerfile and push to a registry the identity can pull from)."
  type        = string
}

variable "key_vault_id" {
  description = "Resource ID of the key vault holding the log encryption key, e.g. /subscriptions/<s>/resourceGroups/<rg>/providers/Microsoft.KeyVault/vaults/<vault>."
  type        = string
}

variable "key_name" {
  description = "Name of the RSA key the decryptor may decrypt with. The identity is scoped to this one key only."
  type        = string
}

variable "app_name" {
  description = "Name for the container app and its supporting resources."
  type        = string
  default     = "formal-decryptor"
}

variable "container_registry_id" {
  description = "Resource ID of the Azure Container Registry holding the image, e.g. /subscriptions/<s>/resourceGroups/<rg>/providers/Microsoft.ContainerRegistry/registries/<registry>. Set it to pull with the decryptor's identity instead of a registry password; leave empty for a publicly pullable image."
  type        = string
  default     = ""
}

variable "allowed_ip_ranges" {
  description = "CIDRs allowed to reach the decryptor. The browser calls it directly, so leaving this empty exposes it to the internet; restrict it to your VPN or office ranges."
  type        = list(string)
  default     = []
}

variable "log_retention_days" {
  description = "Log Analytics retention in days."
  type        = number
  default     = 30
}
