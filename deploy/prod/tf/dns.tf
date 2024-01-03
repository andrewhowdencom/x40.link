// The DNS file contains all information related to maintaining DNS zones, and the records within the zones for this
// project.
resource "google_dns_managed_zone" "x40-link" {
    name = "x40-link"
    dns_name = "x40.link."
    description = "The short link service."
}

resource "google_dns_managed_zone" "x40-dev" {
    name = "x40-dev"
    dns_name = "x40.dev."
    description = "The developer documentation for the x40 project."
}

resource "google_dns_record_set" "x40-dev__ALIAS" {
    managed_zone = google_dns_managed_zone.x40-dev.name
    name = google_dns_managed_zone.x40-dev.dns_name
    ttl = 300
    type = "ALIAS"

    rrdatas = ["andrewhowdencom.github.io."]
}

resource "google_dns_record_set" "x40-dev__CNAME" {
    managed_zone = google_dns_managed_zone.x40-dev.name
    name = "www.${google_dns_managed_zone.x40-dev.dns_name}"
    ttl = 300
    type = "CNAME"

    rrdatas = ["andrewhowdencom.github.io."]

}
