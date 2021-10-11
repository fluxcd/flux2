resource "azurerm_eventhub_namespace" "this" {
  name = "ehns-${local.name_suffix}"
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name
  sku                 = "Standard"
  capacity            = 1
}


resource "azurerm_eventhub" "this" {
  name = "eh-${local.name_suffix}"
  namespace_name      = azurerm_eventhub_namespace.this.name
  resource_group_name = azurerm_resource_group.this.name
  partition_count     = 1
  message_retention   = 1
}

resource "azurerm_eventhub_authorization_rule" "this" {
  name                = "flux"
  resource_group_name = azurerm_resource_group.this.name
  namespace_name      = azurerm_eventhub_namespace.this.name
  eventhub_name       = azurerm_eventhub.this.name
  listen              = true
  send                = true
  manage              = false
}
