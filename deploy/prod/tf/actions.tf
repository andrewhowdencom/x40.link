# Resources that define what GitHub Actions should be able to access in Google Cloud.
# See
# * https://cloud.google.com/blog/products/identity-security/enabling-keyless-authentication-from-github-actions
# * https://github.com/terraform-google-modules/terraform-google-github-actions-runners/tree/master/modules/gh-oidc
# * https://cloud.google.com/iam/docs/principal-identifiers
# * https://cloud.google.com/iam/docs/workload-identity-federation
# * https://github.com/google-github-actions/auth
resource "google_service_account" "x40-link__github-actions" {
  account_id   = "github-actions-at-x40-link"
  display_name = "GitHub Actions @ andrewhowdencom/x40.link"
}

resource "google_project_iam_member" "x40-link__github-actions" {
  project = "andrewhowdencom"
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.x40-link__github-actions.email}"
}

resource "google_iam_workload_identity_pool" "github__production" {
  workload_identity_pool_id = "github--production"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_provider_id = "github"
  workload_identity_pool_id          = google_iam_workload_identity_pool.github__production.workload_identity_pool_id

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.aud"        = "assertion.aud"
    "attribute.repository" = "assertion.repository"
  }

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_member" "x40-link__github-actions" {
  service_account_id = google_service_account.x40-link__github-actions.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github__production.name}/attribute.repository/andrewhowdencom/x40.link"
}