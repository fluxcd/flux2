data "azuredevops_project" "e2e" {
  name = "e2e"
}

resource "azuredevops_git_repository" "fleet_infra" {
  project_id = data.azuredevops_project.e2e.id
  name       = "fleet-infra-${local.name_suffix}"
  default_branch = "refs/heads/main"
  initialization {
    init_type = "Clean"
  }
}

resource "azuredevops_git_repository" "application" {
  project_id = data.azuredevops_project.e2e.id
  name       = "application-${local.name_suffix}"
  default_branch = "refs/heads/main"
  initialization {
    init_type = "Clean"
  }
}
