---
page_title: "Using Vault Provider with STACKIT Secrets Manager"
subcategory: ""
description: |-
   Guide on how to write secrets from Terraform deployed resources on STACKIT cloud to the STACKIT Secrets Manager 
---
# Using Vault Provider with STACKIT Secrets Manager

### Overview

This guide outlines the process of utilizing the HashiCorp Vault provider alongside the STACKIT provider to write secrets into the STACKIT Secrets Manager. Due to the current limitation in the STACKIT Terraform provider where direct writing of secrets isn't supported, this workaround is necessary.

### Problem

Writing credentials from provisioned STACKIT services into a Secrets Manager instance is currently not supported by the STACKIT Terraform provider.

### Solution

To overcome this limitation, you can integrate the HashiCorp Vault provider with the STACKIT provider. This solution involves configuring the Vault provider to interface with the STACKIT Secrets Manager.

### Steps

1. **Configure STACKIT Provider**

    ```hcl
    provider "stackit" {
      region = "eu01"
    }
    ```

2. **Create STACKIT Secrets Manager Instance**

    ```hcl
    resource "stackit_secretsmanager_instance" "example" {
      project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx""
      name       = "example-instance"
    }
    ```

3. **Define STACKIT Secrets Manager User**

    ```hcl
    resource "stackit_secretsmanager_user" "example" {
      project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      instance_id   = stackit_secretsmanager_instance.example.instance_id
      description   = "Example user"
      write_enabled = true
    }
    ```

4. **Configure Vault Provider**

    ```hcl
    provider "vault" {
      address          = "https://prod.sm.eu01.stackit.cloud"
      skip_child_token = true

      auth_login_userpass {
        username = stackit_secretsmanager_user.example.username
        password = stackit_secretsmanager_user.example.password
      }
    }
    ```

5. **Define Terraform Resource (Example: DNS Zone)**

    ```hcl
    resource "stackit_dns_zone" "example" {
      project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      name          = "Example zone"
      dns_name      = "www.example-zone.com"
      contact_email = "aa@bb.ccc"
      type          = "primary"
      acl           = "192.168.0.0/24"
      description   = "Example description"
      default_ttl   = 1230
    }
    ```

6. **Store Secret in Vault**

    ```hcl
    resource "vault_kv_secret_v2" "example" {
      mount               = stackit_secretsmanager_instance.example.instance_id
      name                = "my-secret"
      cas                 = 1
      delete_all_versions = true
      data_json = jsonencode(
        {
          dns_zone_name = stackit_dns_zone.example.dns_name,
        }
      )
    }
    ```

### Note

This example can be adapted for various resources within the provider by replacing the DNS zone with the appropriate resource and associated variables.
