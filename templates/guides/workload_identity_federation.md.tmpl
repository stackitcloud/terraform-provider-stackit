---
page_title: "Workload Identity Federation with GitHub Actions"
---

# Workload Identity Federation with GitHub Actions

Workload Identity Federation (WIF) allows you to authenticate the STACKIT Terraform provider without using long-lived Service Account keys. 
This is particularly useful in CI/CD environments like **GitHub Actions**, **GitLab CI**, or **Azure DevOps**, where you can use short-lived 
OIDC tokens. This guide focuses on using WIF with GitHub Actions, but the principles may apply to other CI/CD platforms that support OIDC.

## Prerequisites

Before using Workload Identity Federation flow, you need to:
1. [Create](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-accounts/) a **Service Account** on STACKIT.

## Setup Workload Identity Federation

WIF can be configured to trust any public OIDC provider following the [docs page](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-account-federations/#create-a-federated-identity-provider)
but for the purpose of this guide we will focus on GitHub Actions as OIDC provider. GitHub Actions supports OIDC authentication using 
the public issuer "https://token.actions.githubusercontent.com" (for GH Enterprise you should check your issuer URL) and setting repository and action information
as part of the OIDC token claims. [More info here](https://docs.github.com/es/actions/concepts/security/openid-connect).

Using this provider [repository](https://github.com/stackitcloud/terraform-provider-stackit) (stackitcloud/terraform-provider-stackit) as example and assuming that we want to 
execute terraform on the main branch, we will configure the service account "Federated identity Provider" with the following configuration:
- **Provider Name**: GitHub # This is just an example, you can choose any name you want
- **Issuer URL**: https://token.actions.githubusercontent.com # This is the public issuer for GitHub Actions OIDC tokens
- **Assertions**:
  - **sub**->equals->repo:stackitcloud/terraform-provider-stackit:ref:refs/heads/main # This is the repository and branch where the action will run
  - **aud**->equals->sts.accounts.stackit.cloud # Mandatory value

> Note: You can use more fine-grained assertions just adding them. More info about OIDC token claims in [GitHub](https://docs.github.com/en/actions/reference/security/oidc)

## Provider Configuration

To use WIF, you must set an `use_oidc` flag to `true` as well as provide an OIDC token for the exchange. While you can provide the token directly in the configuration 
through `service_account_federated_token`, this is not recommended for GitHub Actions as the provider will automatically fetch the token from the GitHub OIDC.

In addition to this, you need to set the `service_account_email` to specify which service account you want to use. This is mandatory as the provider needs to know which service account to exchange the token for.

```hcl
provider "stackit" {
  service_account_email = "terraform-example@sa.stackit.cloud"
  use_oidc              = true
  ... # Other provider configuration
}
```

### Using Environment Variables (Recommended)

In most CI/CD scenarios, the cleanest way is to set the `STACKIT_SERVICE_ACCOUNT_EMAIL` environment variable as well as `STACKIT_USE_OIDC="1"` to enable the WIF flow. This way you don't need to 
change your provider configuration and the provider will automatically fetch the OIDC token and exchange it for a short-lived access token.

## Example GitHub Actions Workflow

> Note: To request OIDC tokens, you need to [grant this permission to the GitHub Actions workflow](https://docs.github.com/en/actions/reference/security/oidc#required-permission).

```yaml
name: Workload Identity Federation with STACKIT

on:
  push:
    branches:
      - '**' 

jobs:
  demo-job:
    name: Workload Identity Federation with STACKIT
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write

    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          terraform_wrapper: false

      - name: Create Test Configuration        
        run: |
          cat <<EOF > main.tf
          terraform {
            required_providers {
              stackit = {
                source = "stackitcloud/stackit"
              }
            }
          }

          provider "stackit" {
            default_region                   = "eu01"
          }

          resource "stackit_service_account" "sa"   {
            project_id = "e1925fbf-5272-497a-8298-1586760670de"
            name       = "terraform-example-ci"
          }
          EOF

      - name: Terraform Init
        run: |
          terraform init
        env:
          STACKIT_USE_OIDC: "1"
          STACKIT_SERVICE_ACCOUNT_EMAIL: "terraform-example@sa.stackit.cloud"

      - name: Terraform Plan
        run: |         
          terraform plan -out=tfplan
        env:
          STACKIT_USE_OIDC: "1"
          STACKIT_SERVICE_ACCOUNT_EMAIL: "terraform-example@sa.stackit.cloud"

      - name: Terraform Apply
        run: terraform apply -auto-approve tfplan
        env:
          STACKIT_USE_OIDC: "1"
          STACKIT_SERVICE_ACCOUNT_EMAIL: "terraform-example@sa.stackit.cloud"
```
