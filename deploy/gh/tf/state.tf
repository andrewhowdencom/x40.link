# Use the Google Cloud provider as configured below
terraform {
  backend "gcs" {
    bucket = "andrewhowdencom-infrastructure-state"
    prefix = "terraform/state+github"
  }

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 5.0"
    }
  }
}

# Configure the GitHub Provider. Authentication provided by the GITHUB_TOKEN environment variable
provider "github" {
  owner = "andrewhowdencom"
}

