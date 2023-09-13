# Contribute to the STACKIT Terraform Provider
Your contribution is welcome! Thank you for your interest in contributing to the STACKIT Terraform Provider. We greatly value your feedback, feature requests, additions to the code, bug reports or documentation extensions.

## Table of contents
- [Developer Guide](#developer-guide)
- [Code Contributions](#code-contributions)
- [Bug Reports](#bug-reports)

## Developer Guide
### Repository structure
The provider resources and data sources for the STACKIT services are located under `stackit/services`. Inside `stackit` you can find several other useful packages such as `validate` and `testutil`. Examples of usage of the provider are located under the `examples` folder. 

### Getting started

Check the [Authentication](README.md#authentication) section on the README.

#### Useful Make commands

These commands can be executed from the project root:

- `make project-tools`: get the required dependencies
- `make lint`: lint the code and examples
- `make generate-docs`: generate terraform documentation
- `make test`: run unit tests
- `make test-acceptance-tf`: run acceptance tests

#### Local development

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

#### Setup centralized Terraform state

You'll need a storage bucket to store the Terraform state and a pair of access key/secret key.
- To order the bucket in the STACKIT Portal, go to Object Storage (on the right) > Buckets > Create bucket.
- To create credentials for a bucket in the STACKIT Portal, go Object Storage (on the right) > Credentials & Groups > Create credentials group.

In the main.tf file location, initialize Terraform by running the following:
```
terraform init -reconfigure -upgrade -backend-config="access_key=<STORAGE_BUCKET_ACCESS_KEY>" -backend-config="secret_key=<STORAGE_BUCKET_SECRET_KEY>"
```

This will throw an error ("Failed to query available provider packages") which can be ignored since we are using the local provider build.

## Code Contributions

To make your contribution, follow these steps:

1. Make sure you are working on the latest version of the `main` branch.
2. Check open or recently closed [Pull Requests](https://github.com/stackitcloud/terraform-provider-stackit/pulls) and [Issues](https://github.com/stackitcloud/terraform-provider-stackit/issues)to make sure the contribution you are making has not been already tackled by someone else.
3. Fork the repo.
4. Make your changes in a branch that is up-to-date with the original repo.
5. Commit your changes including a descriptive message
6. Create a pull request with your changes.
7. The pull request will be reviewed by the repo maintainers. If you need to make further changes, make additional commits to keep commit history. When the PR is merged, commits will be squashed.

## Bug Reports
If you would like to report a bug, please open a [GitHub issue](https://github.com/stackitcloud/terraform-provider-stackit/issues/new).

To ensure we can provide the best support to your issue, follow these guidelines:

1. Go through the existing issues to check if your issue has already been reported.
2. Make sure you are using the latest version of the provider, we will not provide bug fixes for older versions. Also, latest versions may have the fix for your bug.
3. Please provide as much information as you can about your environment, e.g. your version of Go, your version of the provider, which operating system you are using and the corresponding version.
4. Include in your issue the steps to reproduce it, along with code snippets and/or information about your specific use case. This will make the support process much easier and efficient.
