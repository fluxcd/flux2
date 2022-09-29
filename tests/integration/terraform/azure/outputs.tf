output "aks_kube_config" {
  sensitive = true
  value = azurerm_kubernetes_cluster.this.kube_config_raw
}

output "aks_host" {
  value = azurerm_kubernetes_cluster.this.kube_config[0].host
  sensitive = true
}

output "aks_client_certificate" {
  value = base64decode(azurerm_kubernetes_cluster.this.kube_config[0].client_certificate)
  sensitive = true
}

output "aks_client_key" {
  value = base64decode(azurerm_kubernetes_cluster.this.kube_config[0].client_key)
  sensitive = true
}

output "aks_cluster_ca_certificate" {
  value = base64decode(azurerm_kubernetes_cluster.this.kube_config[0].cluster_ca_certificate)
  sensitive = true
}

output "shared_pat" {
  sensitive = true
  value = data.azurerm_key_vault_secret.pat.value
}

output "shared_id_rsa" {
  sensitive = true
  value = data.azurerm_key_vault_secret.id_rsa.value
}

output "shared_id_rsa_pub" {
  sensitive = true
  value = data.azurerm_key_vault_secret.id_rsa_pub.value
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
    client_id = azurerm_kubernetes_cluster.this.kubelet_identity[0].client_id
  }
  sensitive = true
}

output msi_client_id {
  value = azurerm_kubernetes_cluster.this.kubelet_identity[0].object_id
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
    url = azurerm_container_registry.this.login_server
    username = ""
    password = ""
  }
  sensitive = true
}
