# Deploy Infrastructure Changes

## You will need

* The [project tools](../reference/code/requirements.md)
* (If accessing Google Cloud) Google Cloud's [default application configured](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference#running-terraform-on-your-workstation)
* (If accessing GitHub) A [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)
  (with administrative rights to the repository)
* (If accessing Auth0) `AUTH0_DOMAIN` and an Auth0 API token or client credentials

## Environments

There are different environments that are controlled by Tofu. These include:

| Environment | Description                                                 |
|:------------|:------------------------------------------------------------|
| a0          | Auth0 API, CLI application, roles, and permissions          |
| gh          | The GitHub repository containing the project                |
| prod        | All resources deployed in Google Cloud, publicly accessible |

## Deploying Changes

The changes need to be deployed in two steps:

1. Plan the changes:

    ```bash
    task tofu/plan ENV=<environment>
    ```

2. CAREFULLY INSPECT the output, and ensure you understand it all
3. Apply the changes

    ```bash
    task tofu/apply ENV=<environment>
    ```

If an apply only partly succeeds, create a new plan before retrying. The
previous saved plan no longer describes the current infrastructure state.
