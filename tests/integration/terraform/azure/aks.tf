module "aks" {
  source = "git::https://github.com/fluxcd/test-infra.git//tf-modules/azure/aks"

  name     = local.name
  location = var.azure_location
  tags     = var.tags
}

module "acr" {
  source = "git::https://github.com/fluxcd/test-infra.git//tf-modules/azure/acr"

  name             = local.name
  location         = var.azure_location
  aks_principal_id = [module.aks.principal_id]
  resource_group   = module.aks.resource_group
  admin_enabled    = true
  tags             = var.tags

  depends_on = [module.aks]
}
