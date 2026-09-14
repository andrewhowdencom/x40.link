# Runtime identity for the Cloud Run service. Keeping this separate from the
# deployment identity limits application credentials to data-plane access.
resource "google_service_account" "x40-link__runtime" {
  account_id   = "x40-link-runtime"
  display_name = "x40.link Cloud Run runtime"
}

locals {
  x40_link_runtime_roles = toset([
    "roles/datastore.user",
    "roles/monitoring.metricWriter",
    "roles/serviceusage.serviceUsageConsumer",
    "roles/telemetry.tracesWriter",
  ])

  observability_services = toset([
    "cloudtrace.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "telemetry.googleapis.com",
  ])
}

resource "google_project_iam_member" "x40-link__runtime" {
  for_each = local.x40_link_runtime_roles

  project = data.google_project.project.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.x40-link__runtime.email}"
}

resource "google_project_service" "observability" {
  for_each = local.observability_services

  project            = data.google_project.project.project_id
  service            = each.value
  disable_on_destroy = false
}
