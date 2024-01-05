# CR are the resources that are required to access the Google Cloud Run app. Notably, the app itself is not defined
# here as it is controlled on a separate lifecycle.

# Setup DNS resources for the service
resource "google_dns_managed_zone" "x40-link" {
  name        = "x40-link"
  dns_name    = "x40.link."
  description = "The short link service."
}

resource "google_dns_managed_zone" "x40-dev" {
  name        = "x40-dev"
  dns_name    = "x40.dev."
  description = "The developer documentation for the x40 project."
}

resource "google_dns_managed_zone" "h4n-me" {
  name        = "h4n-me"
  dns_name    = "h4n.me."
  description = "A short domain for 'Howden'"
}

resource "google_dns_record_set" "x40-dev__ALIAS" {
  managed_zone = google_dns_managed_zone.x40-dev.name
  name         = google_dns_managed_zone.x40-dev.dns_name
  ttl          = 300
  type         = "ALIAS"

  rrdatas = ["andrewhowdencom.github.io."]
}

resource "google_dns_record_set" "x40-dev__CNAME" {
  managed_zone = google_dns_managed_zone.x40-dev.name
  name         = "www.${google_dns_managed_zone.x40-dev.dns_name}"
  ttl          = 300
  type         = "CNAME"

  rrdatas = ["andrewhowdencom.github.io."]

}

resource "google_dns_record_set" "x40-link" {
  name         = "${each.value}."
  for_each     = toset(var.x40_link_domains)
  managed_zone = google_dns_managed_zone.x40-link.name
  ttl          = 300
  type         = "A"

  rrdatas = [
    google_compute_global_address.x40-link.address
  ]
}

resource "google_dns_record_set" "h4n-me" {
  for_each     = toset(["h4n.me", "www.h4n.me"])
  name         = "${each.value}."
  managed_zone = google_dns_managed_zone.h4n-me.name
  ttl          = 300
  type         = "A"

  rrdatas = [
    google_compute_global_address.x40-link.address
  ]
}

# Setup network path to access the service
resource "google_compute_global_address" "x40-link" {
  name = "x40-link"
}

resource "google_compute_managed_ssl_certificate" "x40-link" {
  name = "x40-link"

  managed {
    domains = var.x40_link_domains
  }
}

resource "google_compute_region_network_endpoint_group" "x40-link" {
  name                  = "x40-link"
  network_endpoint_type = "SERVERLESS"
  region                = "europe-west3"

  cloud_run {
    service = "x40-link"
  }
}

resource "google_compute_backend_service" "x40-link" {
  name                  = "x40-link"
  load_balancing_scheme = "EXTERNAL_MANAGED"


  backend {
    group = google_compute_region_network_endpoint_group.x40-link.id
  }
}

resource "google_compute_url_map" "x40-link" {
  name            = "x40-link"
  default_service = google_compute_backend_service.x40-link.id
}

resource "google_compute_target_https_proxy" "x40-link" {
  name    = "x40-link"
  url_map = google_compute_url_map.x40-link.id

  ssl_certificates = [
    google_compute_managed_ssl_certificate.x40-link.id
  ]
}

resource "google_compute_global_forwarding_rule" "x40-link" {
  name                  = "x40-link"
  load_balancing_scheme = "EXTERNAL_MANAGED"

  ip_address = google_compute_global_address.x40-link.id
  target     = google_compute_target_https_proxy.x40-link.id
  port_range = "443"


}

# Public access to the service
# Allow the general compute user to assume service accounts.
resource "google_cloud_run_service_iam_binding" "default" {
  location = var.region
  service  = var.cloud-run__x40-link
  role     = "roles/run.invoker"
  members = [
    "allUsers"
  ]
}