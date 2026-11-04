variable "project_id" {
  description = "GCP project to deploy the decryptor into."
  type        = string
}

variable "region" {
  description = "Cloud Run region."
  type        = string
  default     = "us-central1"
}

variable "image" {
  description = "Container image for the decryptor (build with the repo-root Dockerfile and push to Artifact Registry)."
  type        = string
}

variable "kms_crypto_key_id" {
  description = "Full resource ID of the KMS crypto key the decryptor may decrypt with, e.g. projects/<p>/locations/<l>/keyRings/<r>/cryptoKeys/<k>. The service account is scoped to decrypt this key only."
  type        = string
}

variable "service_name" {
  description = "Cloud Run service name."
  type        = string
  default     = "formal-decryptor"
}

variable "allow_unauthenticated" {
  description = "Allow public (unauthenticated) invocation. The browser calls the decryptor directly, so this is usually required; restrict ingress or front it with IAP/VPN to limit exposure."
  type        = bool
  default     = true
}
