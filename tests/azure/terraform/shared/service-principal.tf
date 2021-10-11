# Service Principal used by Azure DevOps to push OCI and Helm Charts
resource "azuread_application" "azure_devops" {
  display_name = "azure-devops-${random_pet.suffix.id}"

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

resource "azuread_application_password" "azure_devops" {
  display_name = "password"
  application_object_id = azuread_application.azure_devops.object_id
}

resource "azuread_service_principal" "azure_devops" {
  application_id = azuread_application.azure_devops.application_id
}

resource "azurerm_role_assignment" "azure_devops_acr" {
  scope                = azurerm_container_registry.this.id
  role_definition_name = "Contributor"
  principal_id         = azuread_service_principal.azure_devops.object_id
}

# Service Principal that is used to run the tests in GitHub Actions
resource "azuread_application" "github" {
  display_name = "github-${random_pet.suffix.id}"

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

resource "azuread_application_password" "github" {
  display_name = "password"
  application_object_id = azuread_application.github.object_id
}

resource "azuread_service_principal" "github" {
  application_id = azuread_application.github.application_id
}

data "azurerm_storage_account" "terraform_state" {
  resource_group_name = "terraform-state"
  name                = "terraformstate0419"
}

resource "azurerm_role_assignment" "github_resource_group" {
  scope              = data.azurerm_subscription.current.id
  role_definition_name = "Contributor"
  principal_id       = azuread_service_principal.github.object_id
}

resource "azurerm_role_assignment" "github_acr" {
  scope                = azurerm_container_registry.this.id
  role_definition_name = "Owner"
  principal_id         = azuread_service_principal.github.object_id
}

resource "azurerm_key_vault_access_policy" "github_keyvault_secret_read" {
  key_vault_id = azurerm_key_vault.this.id
  tenant_id = data.azurerm_client_config.current.tenant_id
  object_id = azuread_service_principal.github.object_id

  secret_permissions = [
    "Get",
    "List",
  ]
}
