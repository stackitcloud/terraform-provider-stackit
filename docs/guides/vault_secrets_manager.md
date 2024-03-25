---
page_title: "Using Vault Provider with STACKIT Secrets Manager"
subcategory: ""
description: |-
   Guide on how to write secrets from Terraform deployed resources on STACKIT cloud to the STACKIT Secrets Manager 
---
# Using Vault Provider with STACKIT Secrets Manager

### Overview

This guide outlines the process of utilizing the HashiCorp Vault provider alongside the STACKIT provider to write datasources from STACKIT cloud resources (deployed with Terraform) as secrets in the STACKIT Secrets Manager.

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

5. **Define Terraform Resource (Example: Argus Monitoring Instance)**

    ```hcl
   resource "stackit_argus_instance" "example" {
     project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     name       = "example-instance"
     plan_name  = "Monitoring-Medium-EU01"
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
         grafana_password = stackit_argus_instance.example.grafana_initial_admin_password,
        }
      )
    }
    ```

### Note

This example can be adapted for various resources within the provider by replacing the Argus Monitoring Grafana password with the appropriate resource and associated variables.
