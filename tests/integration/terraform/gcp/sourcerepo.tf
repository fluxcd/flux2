resource "google_sourcerepo_repository" "fleet-infra" {
  name = "fleet-infra-${random_pet.suffix.id}"
}

resource "google_sourcerepo_repository" "application" {
  name = "application-${random_pet.suffix.id}"
}

resource "google_sourcerepo_repository_iam_binding" "application_binding" {
  project = google_sourcerepo_repository.application.project
  repository = google_sourcerepo_repository.application.name
  role = "roles/source.admin"
  members = [
    "user:${var.gcp_email}",
  ]
}

resource "google_sourcerepo_repository_iam_binding" "fleet-infra_binding" {
  project = google_sourcerepo_repository.fleet-infra.project
  repository = google_sourcerepo_repository.fleet-infra.name
  role = "roles/source.admin"
  members = [
    "user:${var.gcp_email}",
  ]
}

