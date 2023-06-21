provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
  zone    = var.gcp_zone
}

resource "random_pet" "suffix" {}

data "google_kms_key_ring" "keyring" {
  name     = var.gcp_keyring
  location = "global"
}

data "google_kms_crypto_key" "my_crypto_key" {
  name     = var.gcp_crypto_key
  key_ring = data.google_kms_key_ring.keyring.id
}

data "google_project" "project" {
}

module "gke" {
  source = "git::https://github.com/fluxcd/test-infra.git//tf-modules/gcp/gke"
  name   = "flux-e2e-${random_pet.suffix.id}"
  tags   = var.tags
}

module "gcr" {
  source = "git::https://github.com/fluxcd/test-infra.git//tf-modules/gcp/gcr"
  name   = "flux-e2e-${random_pet.suffix.id}"
  tags   = var.tags
}

resource "google_sourcerepo_repository" "fleet-infra" {
  name = "fleet-infra-${random_pet.suffix.id}"
}

resource "google_sourcerepo_repository" "application" {
  name = "application-${random_pet.suffix.id}"
}

resource "google_kms_key_ring_iam_binding" "key_ring" {
  key_ring_id = data.google_kms_key_ring.keyring.id
  role        = "roles/cloudkms.cryptoKeyEncrypterDecrypter"

  members = [
    "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com",
  ]
}
