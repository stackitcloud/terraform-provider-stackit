---
page_title: "Using Vault Provider with STACKIT Secrets Manager"
---
# Using Vault Provider with STACKIT Secrets Manager

### Overview

This guide outlines the process of utilizing the HashiCorp Vault provider alongside the STACKIT provider to write secrets in the STACKIT Secrets Manager. The guide focuses on secrets from STACKIT Cloud resources but can be adapted for any secret.

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
         other_secret = ...,
        }
      )
    }
    ```

### Note

This example can be adapted for various resources within the provider as well as any other Secret the user wants to set in the Secrets Manager instance. Adapting this examples means replacing the Argus Monitoring Grafana password with the appropriate value.