terraform {
  backend "gcs" {
    bucket = "andrewhowdencom-infrastructure-state"
    prefix = "terraform/state+github"
  }

  required_providers {
    auth0 = {
      source  = "auth0/auth0"
      version = "1.1.2"
    }
  }
}

provider "auth0" {
}

resource "auth0_resource_server" "x40-api" {
  name             = "api.x40.link"
  identifier       = "https://api.x40.link"
  signing_alg      = "RS256"
  enforce_policies = true


  token_dialect = "access_token_authz"

  allow_offline_access                            = true
  token_lifetime                                  = 8600
  skip_consent_for_verifiable_first_party_clients = true
}

resource "auth0_resource_server_scopes" "x40-api-scopes" {
  resource_server_identifier = auth0_resource_server.x40-api.identifier

  scopes {
    name        = "api.x40.link/scopes/x40.dev.url.ManageURLs.Get"
    description = "Access the RPC method x40.dev.url.ManageURLs.Get"
  }

  scopes {
    name        = "api.x40.link/scopes/x40.dev.url.ManageURLs.New"
    description = "Access the RPC method x40.dev.url.ManageURLs.New"
  }
}

resource "auth0_client" "x40-cli" {
  name        = "x40-cli"
  description = "The terminal application for the @.link shortener"

  // Native is (for example) an Android app. The CLI app is a good parallel for this.
  //
  // See: 
  // 1. https://auth0.com/docs/get-started/applications
  app_type = "native"

  callbacks = [
    // 8064 is the above-8000 port range + the decimal value of @.
    "http://localhost:8064"
  ]

  // We expect the CLI to be fully OIDC conformant. See:
  // https://auth0.com/docs/authenticate/login/oidc-conformant-authentication
  oidc_conformant = true

  // We do not want to check for anything especially.
  organization_require_behavior = "no_prompt"

  grant_types = [
    // For this client, we expect the client to issue
    "urn:ietf:params:oauth:grant-type:device_code",

    // We want to be able to preserve the users login for ... a while. Allow refresh tokens.
    "refresh_token"
  ]

  refresh_token {
    rotation_type   = "rotating"
    expiration_type = "expiring"
  }
}


// Define the roles
resource "auth0_role" "api-user" {
  name        = "https://x40.link/roles/api-user"
  description = "Users who can interact with the api.x40.link"
}

resource "auth0_role_permissions" "api-user" {
  role_id = auth0_role.api-user.id

  permissions {
    name                       = "api.x40.link/scopes/x40.dev.url.ManageURLs.Get"
    resource_server_identifier = auth0_resource_server.x40-api.identifier
  }

  permissions {
    name                       = "api.x40.link/scopes/x40.dev.url.ManageURLs.New"
    resource_server_identifier = auth0_resource_server.x40-api.identifier
  }
}