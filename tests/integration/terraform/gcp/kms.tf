data "google_kms_key_ring" "keyring" {
  name     = var.gcp_keyring
  location = "global"
}

data "google_kms_crypto_key" "my_crypto_key" {
  name     = var.gcp_crypto_key
  key_ring = data.google_kms_key_ring.keyring.id
}

resource "google_kms_key_ring_iam_binding" "key_ring" {
  key_ring_id = data.google_kms_key_ring.keyring.id
  role        = "roles/cloudkms.cryptoKeyEncrypterDecrypter"

  members = [
    "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com",
  ]
}
