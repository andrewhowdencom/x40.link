# Deploy Infrastrastructure Changes

## You will need

* The [project tools](../reference/code/requirements.md)
* (If accessing Google Cloud) Google Cloud's [default application configured](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference#running-terraform-on-your-workstation)
* (If accessing GitHub) A [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)
  (with administrative rights to the repository)

## Environments

There are different environments that are controlled by Tofu. These include:

| Environment | Description                                                 |
|:------------|:------------------------------------------------------------|
| gh          | The GitHub repository containing the project                |
| prod        | All resources deployed in Google Cloud, publicly accessible |

## Deploying Changes

The changes need to be deployed in two steps:

1. Plan the changes:

    ```bash
    task infra/plan ENV=<environment>
    ```

2. CAREFULLY INSPECT the output, and ensure you understand it all
3. Apply the changes

    ```bash
    task infra/apply env=<environment>
    ```
