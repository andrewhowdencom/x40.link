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

