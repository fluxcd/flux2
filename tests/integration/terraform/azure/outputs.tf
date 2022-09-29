output "aks_kubeconfig" {
  description = "kubeconfig of the created AKS cluster"
  value       = module.aks.kubeconfig
  sensitive   = true
}

output "azure_devops_access_token" {
  sensitive = true
  value     = var.azuredevops_pat
}

output "fleet_infra_repository" {
  value = {
    http = azuredevops_git_repository.fleet_infra.remote_url
    ssh  = "ssh://git@ssh.dev.azure.com/v3/${var.azuredevops_org}/${azuredevops_git_repository.fleet_infra.project_id}/${azuredevops_git_repository.fleet_infra.name}"
  }
}

output "application_repository" {
  value = {
    http = azuredevops_git_repository.application.remote_url
    ssh  = "ssh://git@ssh.dev.azure.com/v3/${var.azuredevops_org}/${azuredevops_git_repository.application.project_id}/${azuredevops_git_repository.application.name}"
  }
}

output "aks_client_id" {
  value = module.aks.kubelet_client_id
}

output "event_hub_sas" {
  value     = azurerm_eventhub_authorization_rule.this.primary_connection_string
  sensitive = true
}

output "sops_id" {
  value = azurerm_key_vault_key.sops.id
}

output "acr_url" {
  value = module.acr.registry_url
}
