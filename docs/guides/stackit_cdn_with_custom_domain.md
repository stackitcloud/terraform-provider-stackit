---
page_title: "Using STACKIT CDN to service static files from an HTTP Origin with STACKIT CDN"
---

# Using STACKIT CDN to service static files from an HTTP Origin with STACKIT CDN

This guide will walk you through the process of setting up a STACKIT CDN distribution to serve static files from a
generic HTTP origin using Terraform. This is a common use case for developers who want to deliver content with low
latency and high data transfer speeds.

---

## Prerequisites

Before you begin, make sure you have the following:

* A **STACKIT project** and a user account with the necessary permissions for the CDN.
* A **Service Account Key**: you can read about creating one here: [Create a Service Account Key
](https://docs.stackit.cloud/stackit/en/create-a-service-account-key-175112456.html)

---

## Step 1: Configure the Terraform Provider

First, you need to configure the STACKIT provider in your Terraform configuration. Create a file named `main.tf` and add
the following code. This block tells Terraform to download and use the STACKIT provider.

```terraform
terraform {
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
    }
  }
}

variable "service_account_key" {
  type        = string
  description = "Your STACKIT service account key."
  sensitive   = true
  default     = "path/to/sa-key.json"
}

variable "project_id" {
  type    = string
  default = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" # Your project ID
}

provider "stackit" {
  # The STACKIT provider is configured using the defined variables.
  default_region           = "eu01"
  service_account_key_path = var.service_account_key
}

```

## Step 2: Create the DNS Zone

The first resource you'll create is the DNS zone, which will manage the records for your domain.

```terraform
resource "stackit_dns_zone" "example_zone" {
  project_id    = var.project_id
  name          = "My DNS zone"
  dns_name      = "myapp.runs.onstackit.cloud"
  contact_email = "aa@bb.ccc"
  type          = "primary"
}
```

## Step 3: Create the CDN Distribution

Next, define the CDN distribution. This is the core service that will cache and serve your content from its origin.

```terraform
resource "stackit_cdn_distribution" "example_distribution" {
  project_id = var.project_id

  config = {
    # Define the backend configuration
    backend = {
      type = "http"

      # Replace with the URL of your HTTP origin
      origin_url = "https://your-origin-server.com"
    }

    # The regions where content will be hosted
    regions = ["EU", "US", "ASIA", "AF", "SA"]
    blocked_countries = []
  }

}
```

## Step 4: Create the DNS CNAME Record

Finally, create the **CNAME record** to point your custom domain to the CDN. This step must come after the CDN is
created because it needs the CDN's unique domain name as its target.

```terraform
resource "stackit_dns_record_set" "cname_record" {
  project_id = stackit_dns_zone.example_zone.project_id
  zone_id = stackit_dns_zone.example_zone.zone_id

  # This is the custom domain name which will be added to your zone
  name = "cdn"
  type = "CNAME"
  ttl  = 3600

  # Points to the CDN distribution's unique domain.
  # Notice the added dot at the end of the domain name to point to a FQDN.
  records = ["${stackit_cdn_distribution.example_distribution.domains[0].name}."]
}

```

This record directs traffic from your custom domain to the STACKIT CDN infrastructure.

## Step 5: Add a Custom Domain to the CDN

To provide a user-friendly URL, associate a custom domain (like `cdn.myapp.runs.onstackit.cloud`) with your
distribution.

```terraform
resource "stackit_cdn_custom_domain" "example_custom_domain" {
  project_id = stackit_cdn_distribution.example_distribution.project_id
  distribution_id = stackit_cdn_distribution.example_distribution.distribution_id

  # Creates "cdn.myapp.runs.onstackit.cloud" dynamically
  name = "${stackit_dns_record_set.cname_record.name}.${stackit_dns_zone.example_zone.dns_name}"
}

```

This resource links the subdomain you created in the previous step to the CDN distribution.

## Complete Terraform Configuration

Here is the complete `main.tf` file, which follows the logical order of operations.

```terraform
# This configuration file sets up a complete STACKIT CDN distribution
# with a custom domain managed by STACKIT DNS.

# -----------------------------------------------------------------------------
# PROVIDER CONFIGURATION
# -----------------------------------------------------------------------------

terraform {
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
    }
  }
}

variable "service_account_key" {
  type        = string
  description = "Your STACKIT service account key."
  sensitive   = true
  default     = "path/to/sa-key.json"
}

variable "project_id" {
  type        = string
  description = "Your STACKIT project ID."
  default     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

provider "stackit" {
  # The STACKIT provider is configured using the defined variables.
  default_region           = "eu01"
  service_account_key_path = var.service_account_key
}

# -----------------------------------------------------------------------------
# DNS ZONE RESOURCE
# -----------------------------------------------------------------------------
# The DNS zone manages all records for your domain.
# It's the first resource to be created.
# -----------------------------------------------------------------------------

resource "stackit_dns_zone" "example_zone" {
  project_id    = var.project_id
  name          = "My DNS zone"
  dns_name      = "myapp.runs.onstackit.cloud"
  contact_email = "aa@bb.ccc"
  type          = "primary"
}

# -----------------------------------------------------------------------------
# CDN DISTRIBUTION RESOURCE
# -----------------------------------------------------------------------------
# This resource defines the CDN, its origin, and caching regions.
# -----------------------------------------------------------------------------

resource "stackit_cdn_distribution" "example_distribution" {
  project_id = var.project_id

  config = {
    # Define the backend configuration
    backend = {
      type = "http"

      # Replace with the URL of your HTTP origin
      origin_url = "https://your-origin-server.com"
    }

    # The regions where content will be hosted
    regions = ["EU", "US", "ASIA", "AF", "SA"]
    blocked_countries = []
  }
}

# -----------------------------------------------------------------------------
# CUSTOM DOMAIN AND DNS RECORD
# -----------------------------------------------------------------------------
# These resources link your CDN to a user-friendly custom domain and create
# the necessary DNS record to route traffic.
# -----------------------------------------------------------------------------

resource "stackit_dns_record_set" "cname_record" {
  project_id = stackit_dns_zone.example_zone.project_id
  zone_id = stackit_dns_zone.example_zone.zone_id
  # This is the custom domain name which will be added to your zone
  name       = "cdn"
  type       = "CNAME"
  ttl        = 3600
  # Points to the CDN distribution's unique domain.
  # The dot at the end makes it a fully qualified domain name (FQDN).
  records = ["${stackit_cdn_distribution.example_distribution.domains[0].name}."]

}

resource "stackit_cdn_custom_domain" "example_custom_domain" {
  project_id = stackit_cdn_distribution.example_distribution.project_id
  distribution_id = stackit_cdn_distribution.example_distribution.distribution_id

  # Creates "cdn.myapp.runs.onstackit.cloud" dynamically
  name = "${stackit_dns_record_set.cname_record.name}.${stackit_dns_zone.example_zone.dns_name}"
}

# -----------------------------------------------------------------------------
# OUTPUTS
# -----------------------------------------------------------------------------
# This output will display the final custom URL after `terraform apply` is run.
# -----------------------------------------------------------------------------

output "custom_cdn_url" {
  description = "The final custom domain URL for the CDN distribution."
  value       = "https://${stackit_cdn_custom_domain.example_custom_domain.name}"
}

```
