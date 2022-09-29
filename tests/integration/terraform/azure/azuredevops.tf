resource "azuredevops_project" "e2e" {
  name               = local.name
  visibility         = "private"
  version_control    = "Git"
  work_item_template = "Agile"
  description        = "Test Project for Flux E2E test - Managed by Terraform"
}


resource "azuredevops_git_repository" "fleet_infra" {
  project_id     = azuredevops_project.e2e.id
  name           = "fleet-infra-${local.name}"
  default_branch = "refs/heads/main"
  initialization {
    init_type = "Clean"
  }
}

resource "azuredevops_git_repository" "application" {
  project_id     = azuredevops_project.e2e.id
  name           = "application-${local.name}"
  default_branch = "refs/heads/main"
  initialization {
    init_type = "Clean"
  }
}
