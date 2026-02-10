# STACKIT Terraform Provider

The STACKIT Terraform provider is the official Terraform provider to integrate all the resources developed by [STACKIT](https://www.stackit.de/en/).

## Example Usage

```terraform
provider "stackit" {
  default_region = "eu01"
}

# Authentication

# Workload Identity Federation flow 
provider "stackit" {
  default_region                  = "eu01"
  service_account_email           = var.service_account_email
  service_account_federated_token = var.service_account_federated_token
  use_oidc                        = true
}

# Workload Identity Federation flow (using path)
provider "stackit" {
  default_region                       = "eu01"
  service_account_email                = var.service_account_email
  service_account_federated_token_path = var.service_account_federated_token_path
  use_oidc                             = true
}

# Key flow
provider "stackit" {
  default_region      = "eu01"
  service_account_key = var.service_account_key
  private_key         = var.private_key
}

# Key flow (using path)
provider "stackit" {
  default_region           = "eu01"
  service_account_key_path = var.service_account_key_path
  private_key_path         = var.private_key_path
}
```

## Authentication

To authenticate, you will need a [service account](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/). Create it in the [STACKIT Portal](https://portal.stackit.cloud/) and assign the necessary permissions to it, e.g. `project.owner`. There are multiple ways to authenticate:

- Workload Identity Federation (Recommended)
- Key flow 

When setting up authentication, the provider will always try to use the workload identity federation flow first and search for credentials in several locations, following a specific order:

1. Explicit configuration, e.g. by setting the field `use_oidc` in the provider block (see example below)
2. Environment variable, e.g. by setting `STACKIT_USE_OIDC`
3. Credentials file

   The provider will check the credentials file located in the path defined by the `STACKIT_CREDENTIALS_PATH` env var, if specified,
   or in `$HOME/.stackit/credentials.json` as a fallback.
   The credentials should be set using the same name as the environment variables. Example:

   ```json
   {
     "STACKIT_SERVICE_ACCOUNT_KEY_PATH": "path/to/sa_key.json",
     "STACKIT_PRIVATE_KEY_PATH": "path/to/private_key.pem"
   }
   ```

### Workload Identity Federation (Recommended)

    The following instructions assume that you have created a service account and assigned the necessary permissions to it, e.g. `project.owner`.

When using Workload Identity Federation (WIF), you don't need a static service account secret or key. Instead, the provider exchanges a short-lived OIDC token (from GitHub Actions, GitLab CI, etc.) for a STACKIT access token. This is the most secure way to authenticate in CI/CD environments as it eliminates the need for long-lived secrets.

WIF can be configured to trust any public OIDC provider following the [official documentation](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-account-federations/#create-a-federated-identity-provider).

To use WIF, set the `use_oidc` flag to `true` and provide an OIDC token for the exchange. While you can provide the token directly via `service_account_federated_token`, this is **not recommended for GitHub Actions**, as the provider will automatically fetch the token from the environment. For a complete setup, see our [Workload Identity Federation guide](./guides/workload_identity_federation.md).

In addition to this, you must set the `service_account_email` to specify which service account to impersonate.

### Key flow

    The following instructions assume that you have created a service account and assigned the necessary permissions to it, e.g. `project.owner`.

To use the key flow, you need to have a service account key, which must have an RSA key-pair attached to it.

When creating the service account key, a new pair can be created automatically, which will be included in the service account key. This will make it much easier to configure the key flow authentication in the [STACKIT Terraform Provider](https://github.com/stackitcloud/terraform-provider-stackit), by just providing the service account key.

**Optionally**, you can provide your own private key when creating the service account key, which will then require you to also provide it explicitly to the [STACKIT Terraform Provider](https://github.com/stackitcloud/terraform-provider-stackit), additionally to the service account key. Check the STACKIT Docs for an [example of how to create your own key-pair](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-account-keys/).

To configure the key flow, follow these steps:

1.  Create a service account key:

- Use the [STACKIT Portal](https://portal.stackit.cloud/): go to the `Service Accounts` tab, choose a `Service Account` and go to `Service Account Keys` to create a key. For more details, see [Create a service account key](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-account-keys/)

2.  Save the content of the service account key by copying it and saving it in a JSON file.

    The expected format of the service account key is a **JSON** with the following structure:

```json
{
  "id": "uuid",
  "publicKey": "public key",
  "createdAt": "2023-08-24T14:15:22Z",
  "validUntil": "2023-08-24T14:15:22Z",
  "keyType": "USER_MANAGED",
  "keyOrigin": "USER_PROVIDED",
  "keyAlgorithm": "RSA_2048",
  "active": true,
  "credentials": {
    "kid": "string",
    "iss": "my-sa@sa.stackit.cloud",
    "sub": "uuid",
    "aud": "string",
    (optional) "privateKey": "private key when generated by the SA service"
  }
}
```

3. Configure the service account key for authentication in the provider by following one of the alternatives below:

   - setting the fields in the provider block: `service_account_key` or `service_account_key_path`
   - setting the environment variable: `STACKIT_SERVICE_ACCOUNT_KEY_PATH`
   - setting `STACKIT_SERVICE_ACCOUNT_KEY_PATH` in the credentials file (see above)

> **Optionally, only if you have provided your own RSA key-pair when creating the service account key**, you also need to configure your private key (takes precedence over the one included in the service account key, if present). **The private key must be PEM encoded** and can be provided using one of the options below:
>
> - setting the field in the provider block: `private_key` or `private_key_path`
> - setting the environment variable: `STACKIT_PRIVATE_KEY_PATH`
> - setting `STACKIT_PRIVATE_KEY_PATH` in the credentials file (see above)

# Backend configuration

To keep track of your terraform state, you can configure an [S3 backend](https://developer.hashicorp.com/terraform/language/settings/backends/s3) using [STACKIT Object Storage](https://docs.stackit.cloud/products/storage/object-storage).

To do so, you need an Object Storage [S3 bucket](https://docs.stackit.cloud/products/storage/object-storage/basics/concepts/#buckets) and [credentials](https://docs.stackit.cloud/products/storage/object-storage/basics/concepts/#credentials) to access it. If you need to create them, check [Create and delete Object Storage buckets](https://docs.stackit.cloud/products/storage/object-storage/how-tos/create-and-manage-object-storage-buckets/) and [Create and delete Object Storage credentials](https://docs.stackit.cloud/products/storage/object-storage/how-tos/create-and-delete-object-storage-credentials/).

Once you have everything setup, you can configure the backend by adding the following block to your terraform configuration:

```
terraform {
  backend "s3" {
    bucket = "BUCKET_NAME"
    key    = "path/to/key"
    endpoints = {
      s3 = "https://object.storage.eu01.onstackit.cloud"
    }
    region                      = "eu01"
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_s3_checksum            = true
    skip_requesting_account_id  = true
    secret_key                  = "SECRET_KEY"
    access_key                  = "ACCESS_KEY"
  }
}
```

Note: AWS specific checks must be skipped as they do not work on STACKIT. For details on what those validations do, see [here](https://developer.hashicorp.com/terraform/language/settings/backends/s3#configuration).
<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `authorization_custom_endpoint` (String) Custom endpoint for the Membership service
- `cdn_custom_endpoint` (String) Custom endpoint for the CDN service
- `credentials_path` (String) Path of JSON from where the credentials are read. Takes precedence over the env var `STACKIT_CREDENTIALS_PATH`. Default value is `~/.stackit/credentials.json`.
- `default_region` (String) Region will be used as the default location for regional services. Not all services require a region, some are global
- `dns_custom_endpoint` (String) Custom endpoint for the DNS service
- `edgecloud_custom_endpoint` (String) Custom endpoint for the Edge Cloud service
- `enable_beta_resources` (Boolean) Enable beta resources. Default is false.
- `experiments` (List of String) Enables experiments. These are unstable features without official support. More information can be found in the README. Available Experiments: iam, routing-tables, network
- `git_custom_endpoint` (String) Custom endpoint for the Git service
- `iaas_custom_endpoint` (String) Custom endpoint for the IaaS service
- `kms_custom_endpoint` (String) Custom endpoint for the KMS service
- `loadbalancer_custom_endpoint` (String) Custom endpoint for the Load Balancer service
- `logme_custom_endpoint` (String) Custom endpoint for the LogMe service
- `logs_custom_endpoint` (String) Custom endpoint for the Logs service
- `mariadb_custom_endpoint` (String) Custom endpoint for the MariaDB service
- `modelserving_custom_endpoint` (String) Custom endpoint for the AI Model Serving service
- `mongodbflex_custom_endpoint` (String) Custom endpoint for the MongoDB Flex service
- `objectstorage_custom_endpoint` (String) Custom endpoint for the Object Storage service
- `observability_custom_endpoint` (String) Custom endpoint for the Observability service
- `oidc_request_token` (String) The bearer token for the request to the OIDC provider. For use when authenticating as a Service Account using OpenID Connect.
- `oidc_request_url` (String) The URL for the OIDC provider from which to request an ID token. For use when authenticating as a Service Account using OpenID Connect.
- `opensearch_custom_endpoint` (String) Custom endpoint for the OpenSearch service
- `postgresflex_custom_endpoint` (String) Custom endpoint for the PostgresFlex service
- `private_key` (String) Private RSA key used for authentication, relevant for the key flow. It takes precedence over the private key that is included in the service account key.
- `private_key_path` (String) Path for the private RSA key used for authentication, relevant for the key flow. It takes precedence over the private key that is included in the service account key.
- `rabbitmq_custom_endpoint` (String) Custom endpoint for the RabbitMQ service
- `redis_custom_endpoint` (String) Custom endpoint for the Redis service
- `region` (String, Deprecated) Region will be used as the default location for regional services. Not all services require a region, some are global
- `resourcemanager_custom_endpoint` (String) Custom endpoint for the Resource Manager service
- `scf_custom_endpoint` (String) Custom endpoint for the Cloud Foundry (SCF) service
- `secretsmanager_custom_endpoint` (String) Custom endpoint for the Secrets Manager service
- `server_backup_custom_endpoint` (String) Custom endpoint for the Server Backup service
- `server_update_custom_endpoint` (String) Custom endpoint for the Server Update service
- `service_account_custom_endpoint` (String) Custom endpoint for the Service Account service
- `service_account_email` (String) Service account email. It can also be set using the environment variable STACKIT_SERVICE_ACCOUNT_EMAIL. It is required if you want to use the resource manager project resource. This value is required using OpenID Connect authentication.
- `service_account_federated_token` (String) The OIDC ID token for use when authenticating as a Service Account using OpenID Connect.
- `service_account_federated_token_path` (String) Path for workload identity assertion. It can also be set using the environment variable STACKIT_FEDERATED_TOKEN_FILE.
- `service_account_key` (String) Service account key used for authentication. If set, the key flow will be used to authenticate all operations.
- `service_account_key_path` (String) Path for the service account key used for authentication. If set, the key flow will be used to authenticate all operations.
- `service_account_token` (String, Deprecated) Token used for authentication. If set, the token flow will be used to authenticate all operations.
- `service_enablement_custom_endpoint` (String) Custom endpoint for the Service Enablement API
- `sfs_custom_endpoint` (String) Custom endpoint for the Stackit Filestorage API
- `ske_custom_endpoint` (String) Custom endpoint for the Kubernetes Engine (SKE) service
- `sqlserverflex_custom_endpoint` (String) Custom endpoint for the SQL Server Flex service
- `token_custom_endpoint` (String) Custom endpoint for the token API, which is used to request access tokens when using the key flow
- `use_oidc` (Boolean) Should OIDC be used for Authentication? This can also be sourced from the `STACKIT_USE_OIDC` Environment Variable. Defaults to `false`.
