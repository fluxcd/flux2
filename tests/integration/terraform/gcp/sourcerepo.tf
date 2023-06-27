resource "google_sourcerepo_repository" "fleet-infra" {
  name = "fleet-infra-${random_pet.suffix.id}"
}

resource "google_sourcerepo_repository" "application" {
  name = "application-${random_pet.suffix.id}"
}
