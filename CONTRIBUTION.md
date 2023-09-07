## Contribute
Your contribution is welcome! Please create a pull request (PR). The STACKIT Developer Tools team will review it. A more detailed contribution guideline is planned to come.

### Local development

To test your changes locally, you have to compile the provider (requires Go 1.20) and configure the Terraform CLI to use the local version.

1. Clone the repository.
2. Set the provider address to a custom address for local development. It must correspond to the same address that is included in the dev_overrides block, in step 4.
In `main.go` replace the address `registry.terraform.io/providers/stackitcloud/stackit` with `local-dev.com/stackit/stackit`.
3. Go to the repository root and compile the provider locally to any location by running `go build -o <PATH_TO_BINARY>`. The binary name must start with `terraform-provider`, e.g. `terraform-provider-stackit`.
4. Create a `.terraformrc` config file in your home directory (`~`) for the terraform CLI with the following content:
```
provider_installation {
    dev_overrides {
        "local-dev.com/stackit/stackit" = "<PATH_TO_BINARY>"
    }

    # For all other providers, install them directly from their origin provider
    # registries as normal. If you omit this, Terraform will _only_ use
    # the dev_overrides block, and so no other providers will be available.
    direct {}
}
```
4. Copy one of the folders in the [examples](examples/) folder to a location of your choosing, and define the Terraform variables according to its README. The main.tf file needs some additional configuration to use the local provider:
```
terraform {
  required_providers {
    stackit = {
      source = "local-dev.com/stackit/stackit"
    }
  }
}
```
5. Go to the copied example and initialize Terraform by running `terraform init -reconfigure -upgrade`. This will throw an error ("Failed to query available provider packages") which can be ignored since we are using the local provider build.
> Note: Terraform will store its resources' states locally. To allow multiple people to use the same resources, check [Setup for multi-person usage](#setup-centralized-terraform-state)
6. Setup authentication by setting the env var `STACKIT_SERVICE_ACCOUNT_TOKEN` as a valid token (see [Authentication](#authentication) for more details on how to autenticate).
7. Run `terraform plan` or `terraform apply` commands.

# Setup centralized Terraform state

You'll need a storage bucket to store the Terraform state and a pair of access key/secret key.
- To order the bucket in the STACKIT Portal, go to Object Storage (on the right) > Buckets > Create bucket.
- To create credentials for a bucket in the STACKIT Portal, go Object Storage (on the right) > Credentials & Groups > Create credentials group.

In the main.tf file location, initialize Terraform by running the following:
```
terraform init -reconfigure -upgrade -backend-config="access_key=<STORAGE_BUCKET_ACCESS_KEY>" -backend-config="secret_key=<STORAGE_BUCKET_SECRET_KEY>"
```
This will throw an error ("Failed to query available provider packages") which can be ignored since we are using the local provider build.