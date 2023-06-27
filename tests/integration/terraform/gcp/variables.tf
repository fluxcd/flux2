variable "gcp_project_id" {
  type        = string
  description = "GCP project to create the resources in"
}

variable "gcp_email" {
  type        = string
  description = "GCP user email"
}

variable "gcp_region" {
  type        = string
  default     = "us-central1"
  description = "GCP region"
}

variable "gcp_zone" {
  type        = string
  default     = "us-central1"
  description = "GCP zone"
}

variable "gcp_keyring" {
  type        = string
  description = "GCP keyring that contains crypto key for encrypting secrets"
}

variable "gcp_crypto_key" {
  type        = string
  description = "GCP crypto key for encrypting secrets"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "tags for created resources"
}
