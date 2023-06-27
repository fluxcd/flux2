resource "google_pubsub_topic" "pubsub" {
  name                       = local.name
  labels                     = var.tags
  message_retention_duration = "7200s"
}

resource "google_pubsub_subscription" "sub" {
  project = var.gcp_project_id
  name    = local.name
  topic   = google_pubsub_topic.pubsub.name
}
