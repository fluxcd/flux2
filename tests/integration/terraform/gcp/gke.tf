module "gke" {
  source = "git::https://github.com/fluxcd/test-infra.git//tf-modules/gcp/gke"

  name = local.name
  tags = var.tags
}

module "gcr" {
  source = "git::https://github.com/fluxcd/test-infra.git//tf-modules/gcp/gcr"

  name = local.name
  tags = var.tags
}
