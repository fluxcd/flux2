resource "azurerm_key_vault" "this" {
  name                = "kv-${random_pet.suffix.id}"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"
}

resource "azurerm_key_vault_access_policy" "sops_write" {
  key_vault_id = azurerm_key_vault.this.id
  tenant_id = data.azurerm_client_config.current.tenant_id
  object_id = data.azurerm_client_config.current.object_id

  key_permissions = [
    "Encrypt",
    "Decrypt",
    "Create",
    "Delete",
    "Purge",
    "Get",
    "List",
  ]
}

resource "azurerm_key_vault_key" "sops" {
  depends_on = [azurerm_key_vault_access_policy.sops_write]

  name         = "sops"
  key_vault_id = azurerm_key_vault.this.id
  key_type     = "RSA"
  key_size     = 2048

  key_opts = [
    "decrypt",
    "encrypt",
  ]
}
