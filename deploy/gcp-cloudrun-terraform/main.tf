terraform {
  required_version = ">= 1.1.8"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Dedicated identity for the decryptor, scoped to decrypt one key (below).
resource "google_service_account" "decryptor" {
  project      = var.project_id
  account_id   = var.service_name
  display_name = "Formal log decryptor"
}

resource "google_cloud_run_v2_service" "decryptor" {
  name     = var.service_name
  location = var.region
  project  = var.project_id
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.decryptor.email
    containers {
      image = var.image
      ports {
        container_port = 8080
      }
      env {
        name  = "FORMAL_CONTROL_PLANE_URL"
        value = var.formal_control_plane_url
      }
    }
  }
}

# The browser calls the decryptor directly, so it must be publicly invocable.
resource "google_cloud_run_v2_service_iam_member" "public" {
  count    = var.allow_unauthenticated ? 1 : 0
  name     = google_cloud_run_v2_service.decryptor.name
  location = var.region
  project  = var.project_id
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# The key to decrypt with comes from the caller-supplied JWE, so scope the
# service account to this one key rather than granting project-wide decrypt.
resource "google_kms_crypto_key_iam_member" "decrypter" {
  crypto_key_id = var.kms_crypto_key_id
  role          = "roles/cloudkms.cryptoKeyDecrypter"
  member        = "serviceAccount:${google_service_account.decryptor.email}"
}
