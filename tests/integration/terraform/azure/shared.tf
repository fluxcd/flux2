data "azurerm_key_vault_secret" "pat" {
  depends_on = [azurerm_key_vault_secret.pat]
  key_vault_id = resource.azurerm_key_vault.this.id
  name         = "pat"
}

data "azurerm_key_vault_secret" "id_rsa" {
  depends_on = [azurerm_key_vault_secret.id_rsa]

  key_vault_id = resource.azurerm_key_vault.this.id
  name         = "id-rsa"
}

data "azurerm_key_vault_secret" "id_rsa_pub" {
  depends_on = [azurerm_key_vault_secret.id_rsa_pub]

  key_vault_id = resource.azurerm_key_vault.this.id
  name         = "id-rsa-pub"
}
