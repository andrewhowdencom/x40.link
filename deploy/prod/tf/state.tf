# Use the Google Cloud provider as configured below
terraform {
  backend "gcs" {
    bucket = "andrewhowdencom-infrastructure-state"
    prefix = "terraform/state"
  }

  required_providers {
    google = {
      source  = "hashicorp/google-beta"
      version = "~> 5.10.0 "
    }
  }
}

provider "google" {
  project = "andrewhowdencom"
  region  = "europe-west3"
  zone    = "europe-west3-b"
}

// Setup up resources that allow the storage bucket to be encrypted at rest. See:
// 1. https://cloud.google.com/docs/terraform/resource-management/store-state
// 2. https://github.com/terraform-google-modules/terraform-docs-samples/blob/main/storage/flask_google_cloud_quickstart/main.tf
resource "google_kms_key_ring" "terraform-state" {
  name     = "andrewhowdencom-infrastructure-state"
  location = "europe-west3"
}

# Enable the Cloud Storage service account to encrypt/decrypt Cloud KMS keys
data "google_project" "project" {
}

resource "google_project_iam_member" "default" {
  project = data.google_project.project.project_id
  role    = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member  = "serviceAccount:service-${data.google_project.project.number}@gs-project-accounts.iam.gserviceaccount.com"
}

# Create the encryption / decryption key
resource "google_kms_crypto_key" "terraform_state_bucket" {
  name            = "andrewhowdencom-infrastructure-state"
  key_ring        = google_kms_key_ring.terraform-state.id
  rotation_period = "86400s"

  lifecycle {
    prevent_destroy = true
  }
}

# Create the state bucket
resource "google_storage_bucket" "infrastructure-state" {
  name          = "andrewhowdencom-infrastructure-state"
  force_destroy = "false"
  location      = "europe-west3"
  storage_class = "STANDARD"

  versioning {
    enabled = true
  }

  depends_on = [
    google_project_iam_member.default
  ]
}

