locals {
  shared_suffix = "oarfish"
}

data "azurerm_resource_group" "shared" {
  name = "e2e-shared"
}

data "azurerm_container_registry" "shared" {
  name                = "acrapps${local.shared_suffix}"
  resource_group_name = data.azurerm_resource_group.shared.name
}

data "azurerm_key_vault" "shared" {
  resource_group_name = data.azurerm_resource_group.shared.name
  name                = "kv-credentials-${local.shared_suffix}"
}

data "azurerm_key_vault_secret" "shared_pat" {
  key_vault_id = data.azurerm_key_vault.shared.id
  name         = "pat"
}

data "azurerm_key_vault_secret" "shared_id_rsa" {
  key_vault_id = data.azurerm_key_vault.shared.id
  name         = "id-rsa"
}

data "azurerm_key_vault_secret" "shared_id_rsa_pub" {
  key_vault_id = data.azurerm_key_vault.shared.id
  name         = "id-rsa-pub"
}
