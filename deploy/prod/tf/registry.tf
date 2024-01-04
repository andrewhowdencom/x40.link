# Registry contains all information, IAM etc. associated with the container image registry
resource "google_artifact_registry_repository" "x40-link" {
  location      = "europe-west3"
  repository_id = "x40-link"
  description   = "Docker repository for the x40 project"
  format        = "DOCKER"

  cleanup_policy_dry_run = false

  docker_config {
    immutable_tags = true
  }

  cleanup_policies {
    id     = "keep-minimum-versions"
    action = "KEEP"
    most_recent_versions {
      package_name_prefixes = ["x40.link"]
      keep_count            = 5
    }
  }
}