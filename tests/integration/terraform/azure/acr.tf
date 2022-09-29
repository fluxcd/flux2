resource "azurerm_container_registry" "this" {
  name                = "acrapps${random_pet.suffix.id}"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  sku                 = "Standard"
}
