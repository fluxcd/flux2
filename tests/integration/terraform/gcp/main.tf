provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
  zone    = var.gcp_zone
}

resource "random_pet" "suffix" {}

locals {
  name = "e2e-${random_pet.suffix.id}"
}

data "google_project" "project" {}
