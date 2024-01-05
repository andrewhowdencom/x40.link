variable "x40_link_domains" {
  type    = list(string)
  default = ["x40.link", "www.x40.link", "app.x40.link", "api.x40.link"]
}

variable "cloud-run__x40-link" {
  type    = string
  default = "x40-link"
}

variable "region" {
  type    = string
  default = "europe-west3"
}