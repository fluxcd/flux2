output "azure_devops_sp" {
  value = {
    client_id = azuread_service_principal.azure_devops.application_id
    client_secret = azuread_application_password.azure_devops.value
  }
  sensitive = true
}

output "github_sp" {
  value = {
    tenant_id = data.azurerm_client_config.current.tenant_id
    subscription_id = data.azurerm_client_config.current.subscription_id
    client_id = azuread_service_principal.github.application_id
    client_secret = azuread_application_password.github.value
  }
  sensitive = true
}

