terraform {
  backend "azurerm" {
    resource_group_name  = "terraform-state"
    storage_account_name = "terraformstate0419"
    container_name       = "aks-tfstate"
    key                  = "prod.terraform.tfstate"
  }

  required_version = "1.2.8"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.20.0"
    }
    azuread = {
      source = "hashicorp/azuread"
      version = "2.28.0"
    }
    azuredevops = {
      source = "microsoft/azuredevops"
      version = "0.2.2"
    }
  }
}

provider "azurerm" {
  features {}
}

provider "azuredevops" {
  org_service_url = "https://dev.azure.com/${local.azure_devops_org}"
  personal_access_token = data.azurerm_key_vault_secret.shared_pat.value
}

data "azurerm_client_config" "current" {}

resource "random_pet" "suffix" {}

locals {
  azure_devops_org = "flux-azure"
  name_suffix = "e2e-${random_pet.suffix.id}"
}

resource "azurerm_resource_group" "this" {
  name     = "rg-${local.name_suffix}"
  location = "West Europe"

  tags = {
    environment = "e2e"
  }
}
