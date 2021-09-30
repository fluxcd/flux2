resource "azuread_application" "flux" {
  display_name = "flux-${local.name_suffix}"

  required_resource_access {
    resource_app_id = "00000003-0000-0000-c000-000000000000"

    resource_access {
      id   = "df021288-bdef-4463-88db-98f22de89214"
      type = "Role"
    }
  }

  required_resource_access {
    resource_app_id = "00000002-0000-0000-c000-000000000000"

    resource_access {
      id   = "1cda74f2-2616-4834-b122-5cb1b07f8a59"
      type = "Role"
    }
    resource_access {
      id   = "78c8a3c8-a07e-4b9e-af1b-b5ccab50a175"
      type = "Role"
    }
  }
}

resource "azuread_service_principal" "flux" {
  application_id = azuread_application.flux.application_id
}

resource "azuread_service_principal_password" "flux" {
  service_principal_id = azuread_service_principal.flux.object_id
}

resource "azurerm_role_assignment" "acr" {
  scope                = data.azurerm_container_registry.shared.id
  role_definition_name = "AcrPull"
  principal_id         = azuread_service_principal.flux.object_id
}

resource "azurerm_key_vault_access_policy" "sops_decrypt" {
  key_vault_id = azurerm_key_vault.this.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = azuread_service_principal.flux.object_id

  key_permissions = [
    "Encrypt",
    "Decrypt",
    "Get",
    "List",
  ]
}
