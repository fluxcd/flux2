terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">=3.20.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = ">=2.28.0"
    }
    azuredevops = {
      source  = "microsoft/azuredevops"
      version = ">=0.2.2"
    }
  }
}

provider "azurerm" {
  features {}
}

provider "azuredevops" {
  org_service_url       = "https://dev.azure.com/${var.azuredevops_org}"
  personal_access_token = var.azuredevops_pat
}

data "azurerm_client_config" "current" {}

resource "random_pet" "suffix" {
  separator = "o"
}

locals {
  name = "e2e${random_pet.suffix.id}"
}
