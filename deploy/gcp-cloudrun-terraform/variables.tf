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

variable "formal_control_plane_url" {
  description = "Origin of the Formal Console that calls the decryptor from the browser. Set it to your self-hosted console's origin, e.g. https://formal.example.com."
  type        = string
  default     = "https://app.formal.ai"

  validation {
    condition     = can(regex("^https://[a-z0-9.-]+(:[0-9]+)?$", var.formal_control_plane_url))
    error_message = "formal_control_plane_url must be a lowercase https origin with no path or trailing slash, e.g. https://formal.example.com."
  }
}
