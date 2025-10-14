---
page_title: "How to provision Cloud Foundry using Terraform"
---
# How to provision Cloud Foundry using Terraform

## Objective

This tutorial demonstrates how to provision Cloud Foundry resources by
integrating the STACKIT Terraform provider with the Cloud Foundry Terraform
provider. The STACKIT Terraform provider will create a managed Cloud Foundry
organization and set up a technical "org manager" user with
`organization_manager` permissions. These credentials, along with the Cloud
Foundry API URL (retrieved dynamically from a platform data resource), are
passed to the Cloud Foundry Terraform provider to manage resources within the
new organization.

### Output

This configuration creates a Cloud Foundry organization, mirroring the structure
created via the portal. It sets up three distinct spaces: `dev`, `qa`, and
`prod`. The configuration assigns, a specified user the `organization_manager`
and `organization_user` roles at the organization level, and the
`space_developer` role in each space.

### Scope

This tutorial covers the interaction between the STACKIT Terraform provider and
the Cloud Foundry Terraform provider. It assumes you are familiar with:

- Setting up a STACKIT project and configuring the STACKIT Terraform provider
  with a service account (see the general STACKIT documentation for details).
- Basic Terraform concepts, such as variables and locals.

This document does not cover foundational topics or every feature of the Cloud
Foundry Terraform provider.

### Example configuration

The following Terraform configuration provisions a Cloud Foundry organization
and related resources using the STACKIT Terraform provider and the Cloud Foundry
Terraform provider:

```
terraform {
  required_providers {
    stackit = {
      source = "stackitcloud/stackit"
    }
    cloudfoundry = {
      source = "cloudfoundry/cloudfoundry"
    }
  }
}

variable "project_id" {
  type        = string
  description = "Id of the Project"
}

variable "org_name" {
  type        = string
  description = "Name of the Organization"
}

variable "admin_email" {
  type        = string
  description = "Users who are granted permissions"
}

provider "stackit" {
  default_region = "eu01"
}

resource "stackit_scf_organization" "scf_org" {
  name       = var.org_name
  project_id = var.project_id
}

data "stackit_scf_platform" "scf_platform" {
  project_id  = var.project_id
  platform_id = stackit_scf_organization.scf_org.platform_id
}

resource "stackit_scf_organization_manager" "scf_manager" {
  project_id = var.project_id
  org_id     = stackit_scf_organization.scf_org.org_id
}

provider "cloudfoundry" {
  api_url  = data.stackit_scf_platform.scf_platform.api_url
  user     = stackit_scf_organization_manager.scf_manager.username
  password = stackit_scf_organization_manager.scf_manager.password
}

locals {
  spaces = ["dev", "qa", "prod"]
}

resource "cloudfoundry_org_role" "org_user" {
  username = var.admin_email
  type     = "organization_user"
  org      = stackit_scf_organization.scf_org.org_id
}

resource "cloudfoundry_org_role" "org_manager" {
  username = var.admin_email
  type     = "organization_manager"
  org      = stackit_scf_organization.scf_org.org_id
}

resource "cloudfoundry_space" "spaces" {
  for_each = toset(local.spaces)
  name     = each.key
  org      = stackit_scf_organization.scf_org.org_id
}

resource "cloudfoundry_space_role" "space_developer" {
  for_each   = toset(local.spaces)
  username   = var.admin_email
  type       = "space_developer"
  depends_on = [cloudfoundry_org_role.org_user]
  space      = cloudfoundry_space.spaces[each.key].id
}
```

## Explanation of configuration

### STACKIT provider configuration

```
provider "stackit" {
  default_region = "eu01"
}
```

The STACKIT Cloud Foundry Application Programming Interface (SCF API) is
regionalized. Each region operates independently. Set `default_region` in the
provider configuration, to specify the region for all resources, unless you
override it for individual resources. You must also provide access data for the
relevant STACKIT project for the provider to function.

For more details, see
the:[STACKIT Terraform Provider documentation.](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs)

### stackit_scf_organization.scf_org resource

```
resource "stackit_scf_organization" "scf_org" {
  name       = var.org_name
  project_id = var.project_id
}
```

This resource provisions a Cloud Foundry organization, which acts as the
foundational container in the Cloud Foundry environment. Each Cloud Foundry
provider configuration is scoped to a specific organization. The organization’s
name, defined by a variable, must be unique across the platform. The
organization is created within a designated STACKIT project, which requires the
STACKIT provider to be configured with the necessary permissions for that
project.

### stackit_scf_organization_manager.scf_manager resource

```
resource "stackit_scf_organization_manager" "scf_manager" {
  project_id = var.project_id
  org_id     = stackit_scf_organization.scf_org.org_id
}
```

This resource creates a technical user in the Cloud Foundry organization with
the organization_manager permission. The user is linked to the organization and
is automatically deleted when the organization is removed.

### stackit_scf_platform.scf_platform data source

```
data "stackit_scf_platform" "scf_platform" {
  project_id  = var.project_id
  platform_id = stackit_scf_organization.scf_org.platform_id
}
```

This data source retrieves properties of the Cloud Foundry platform where the
organization is provisioned. It does not create resources, but provides
information about the existing platform.

### Cloud Foundry provider configuration

```
provider "cloudfoundry" {
  api_url  = data.stackit_scf_platform.scf_platform.api_url
  user     = stackit_scf_organization_manager.scf_manager.username
  password = stackit_scf_organization_manager.scf_manager.password
}
```

The Cloud Foundry provider is configured to manage resources in the new
organization. The provider uses the API URL from the `stackit_scf_platform` data
source and authenticates using the credentials of the technical user created by
the `stackit_scf_organization_manager` resource.

For more information, see the:
[Cloud Foundry Terraform Provider documentation.](https://registry.terraform.io/providers/cloudfoundry/cloudfoundry/latest/docs)

## Deploy resources

Follow these steps to initialize your environment and provision Cloud Foundry
resources using Terraform.

### Initialize Terraform

Run the following command to initialize the working directory and download the
required provider plugins:

```
terraform init
```

### Create the organization manager user

Run this command to provision the organization and technical user needed to
initialize the Cloud Foundry Terraform provider. This step is required only
during the initial setup. For later changes, you do not need the -target flag.

```
terraform apply -target stackit_scf_organization_manager.scf_manager
```

### Apply the full configuration

Run this command to provision all resources defined in your Terraform
configuration within the Cloud Foundry organization:

```
terraform apply
```

## Verify the deployment

Verify that your Cloud Foundry resources are provisioned correctly. Use the
following Cloud Foundry CLI commands to check applications, services, and
routes:

- `cf apps`
- `cf services`
- `cf routes`

For more information, see the
[Cloud Foundry documentation](https://docs.cloudfoundry.org/) and the
[Cloud Foundry CLI Reference Guide](https://cli.cloudfoundry.org/).