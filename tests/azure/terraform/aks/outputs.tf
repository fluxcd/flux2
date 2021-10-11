output "aks_kube_config" {
  sensitive = true
  value = azurerm_kubernetes_cluster.this.kube_config_raw
}

output "aks_host" {
  value = azurerm_kubernetes_cluster.this.kube_config[0].host
}

output "aks_client_certificate" {
  value = base64decode(azurerm_kubernetes_cluster.this.kube_config[0].client_certificate)
}

output "aks_client_key" {
  value = base64decode(azurerm_kubernetes_cluster.this.kube_config[0].client_key)
}

output "aks_cluster_ca_certificate" {
  value = base64decode(azurerm_kubernetes_cluster.this.kube_config[0].cluster_ca_certificate)
}

output "shared_pat" {
  sensitive = true
  value = data.azurerm_key_vault_secret.shared_pat.value
}

output "shared_id_rsa" {
  sensitive = true
  value = data.azurerm_key_vault_secret.shared_id_rsa.value
}

output "shared_id_rsa_pub" {
  sensitive = true
  value = data.azurerm_key_vault_secret.shared_id_rsa_pub.value
}

output "fleet_infra_repository" {
  value = {
    http = azuredevops_git_repository.fleet_infra.remote_url
    ssh = "ssh://git@ssh.dev.azure.com/v3/${local.azure_devops_org}/${azuredevops_git_repository.fleet_infra.project_id}/${azuredevops_git_repository.fleet_infra.name}"
  }
}

output "application_repository" {
  value = {
    http = azuredevops_git_repository.application.remote_url
    ssh = "ssh://git@ssh.dev.azure.com/v3/${local.azure_devops_org}/${azuredevops_git_repository.application.project_id}/${azuredevops_git_repository.application.name}"
  }
}

output "flux_azure_sp" {
  value = {
    tenant_id = data.azurerm_client_config.current.tenant_id
    client_id = azuread_service_principal.flux.application_id
    client_secret = azuread_service_principal_password.flux.value
  }
  sensitive = true
}

output "event_hub_sas" {
  value = azurerm_eventhub_authorization_rule.this.primary_connection_string
  sensitive = true
}

output "sops_id" {
  value = azurerm_key_vault_key.sops.id
}

output "acr" {
  value = {
    url = data.azurerm_container_registry.shared.login_server
    username = azuread_service_principal.flux.application_id
    password = azuread_service_principal_password.flux.value
  }
  sensitive = true
}
