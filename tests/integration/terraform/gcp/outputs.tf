output "kubeconfig" {
  value     = module.gke.kubeconfig
  sensitive = true
}

output "gcp_project" {
  value = var.gcp_project_id
}

output "gcp_region" {
  value = var.gcp_region
}

output "artifact_registry_id" {
  value = module.gcr.artifact_repository_id
}

output "sops_id" {
  value = data.google_kms_crypto_key.my_crypto_key.id
}

output "fleet_infra_url" {
  value = "ssh://${var.gcp_email}@source.developers.google.com:2022/p/${var.gcp_project_id}/r/${google_sourcerepo_repository.fleet-infra.name}"
}

output "application_url" {
  value = "ssh://${var.gcp_email}@source.developers.google.com:2022/p/${var.gcp_project_id}/r/${google_sourcerepo_repository.application.name}"
}
