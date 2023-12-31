# Contains all content related to the configuration of this repository (where it is stored in terraform).
# Unfortunately, the repository was bootstrapped outside TF so it is likely some configuration is missed. This
# file is used to drive changes, but its reflection of the complete reality is "best effort".

resource "github_repository" "x40-link" {
  name = "x40.link"

  description = "The codebase powering @.link"

  # Used for multiple commits
  allow_merge_commit = true
  allow_rebase_merge = false

  # Used for single commit
  allow_squash_merge = true

  has_downloads = true
  has_issues    = true
  has_projects  = false
  has_wiki      = false

  # Automatically merge in code when it passes all tests. Need to 
  allow_auto_merge = true

  # Cleanup Branches
  delete_branch_on_merge = true

  # Prevent secrets inadvertently propagating
  security_and_analysis {
    secret_scanning {
      status = "enabled"
    }

    secret_scanning_push_protection {
      status = "enabled"
    }
  }

  pages {
    build_type = "workflow"
    cname      = "www.x40.dev"

    source {
      branch = "main"
      path   = "/"
    }
  }
}

resource "github_repository_ruleset" "x40-link" {
  name        = "deployable-main"
  repository  = github_repository.x40-link.name
  target      = "branch"
  enforcement = "active"

  rules {
    required_linear_history = false
    required_signatures     = true

    pull_request {
      dismiss_stale_reviews_on_push = true
      require_code_owner_review     = true
    }

    required_status_checks {
      strict_required_status_checks_policy = true

      required_check {
        context        = "test"
        integration_id = "15368"
      }
    }
  }

  conditions {
    ref_name {
      include = ["~DEFAULT_BRANCH"]
      exclude = []
    }
  }
}