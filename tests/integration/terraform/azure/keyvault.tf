resource "azurerm_key_vault" "this" {
  name                = local.name
  resource_group_name = module.aks.resource_group
  location            = var.azure_location
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"
  tags                = var.tags
}

resource "azurerm_key_vault_access_policy" "admin" {
  key_vault_id = azurerm_key_vault.this.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = data.azurerm_client_config.current.object_id

  key_permissions = [
    "Create",
    "Update",
    "Encrypt",
    "Delete",
    "Get",
    "List",
    "Purge",
    "Recover",
    "GetRotationPolicy",
    "SetRotationPolicy"
  ]

  secret_permissions = [
    "Get",
    "Delete",
    "Purge",
    "Recover"
  ]

}

resource "azurerm_key_vault_access_policy" "cluster_binding" {
  key_vault_id = azurerm_key_vault.this.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = module.aks.principal_id

  key_permissions = [
    "Decrypt",
    "Encrypt",
  ]
}

resource "azurerm_key_vault_key" "sops" {
  depends_on = [azurerm_key_vault_access_policy.admin]

  name         = "sops"
  key_vault_id = azurerm_key_vault.this.id
  key_type     = "RSA"
  key_size     = 2048
  tags         = var.tags

  key_opts = [
    "decrypt",
    "encrypt",
  ]
}
