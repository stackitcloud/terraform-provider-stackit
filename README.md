# Introduction

This project is the official Terraform provider for STACKIT.

# Getting Started

Check one of the examples in the [examples](examples/) folder.

# Authentication

To authenticate, you will need a [service account](https://docs.stackit.cloud/stackit/en/service-accounts-134415819.html). Create it in the STACKIT Portal an assign it the necessary permissions, e.g. `project.owner`. There are multiple ways to authenticate:

- Key flow (recommended)
- Token flow

When setting up authentication, the provider will always try to use the key flow first and search for credentials in several locations, following a specific order:

1. Explicit configuration, e.g. by seting the field `stackit_service_account_key_path` in the provider block (see example below)
2. Environment variable, e.g. by setting `STACKIT_SERVICE_ACCOUNT_KEY_PATH`
3. Credentials file

   The SDK will check the credentials file located in the path defined by the `STACKIT_CREDENTIALS_PATH` env var, if specified,
   or in `$HOME/.stackit/credentials.json` as a fallback.
   The credentials file should be a json and each credential should be set using the name of the respective environment variable, as stated below in each flow. Example:

   ```json
   {
     "STACKIT_SERVICE_ACCOUNT_TOKEN": "foo_token",
     "STACKIT_SERVICE_ACCOUNT_KEY_PATH": "path/to/sa_key.json"
   }
   ```

## Key flow

    The following instructions assume that you have created a service account and assigned it the necessary permissions, e.g. `project.owner`.

To use the key flow, you need to have a service account key, which must have an RSA key-pair attached to it.

When creating the service account key, a new pair can be created automatically, which will be included in the service account key. This will make it much easier to configure the key flow authentication in the CLI, by just providing the service account key.

**Optionally**, you can provide your own private key when creating the service account key, which will then require you to also provide it explicitly to the CLI, additionaly to the service account key. Check the STACKIT Knowledge Base for an [example of how to create your own key-pair](https://docs.stackit.cloud/stackit/en/usage-of-the-service-account-keys-in-stackit-175112464.html#UsageoftheserviceaccountkeysinSTACKIT-CreatinganRSAkey-pair).

To configure the key flow, follow this steps:

1.  Create a service account key:

- Use the STACKIT Portal: go to the `Service Accounts` tab, choose a `Service Account` and go to `Service Account Keys` to create a key. For more details, see [Create a service account key](https://docs.stackit.cloud/stackit/en/create-a-service-account-key-175112456.html)

2.  Save the content of the service account key by copying it and saving it in a JSON file.

    The expected format of the service account key is a **json** with the following structure:

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

3. Configure the service account key for authentication in the SDK by following one of the alternatives below:

   - setting the fiels in the provider block: `service_account_key` or `service_account_key_path`
   - setting the environment variable: `STACKIT_SERVICE_ACCOUNT_KEY_PATH`
   - setting `STACKIT_SERVICE_ACCOUNT_KEY_PATH` in the credentials file (see above)

> **Optionally, only if you have provided your own RSA key-pair when creating the service account key**, you also need to configure your private key (takes precedence over the one included in the service account key, if present). **The private key must be PEM encoded** and can be provided using one of the options below:
>
> - setting the field in the provider block: `private_key` or `private_key_path`
> - setting the environment variable: `STACKIT_PRIVATE_KEY_PATH`
> - setting `STACKIT_PRIVATE_KEY_PATH` in the credentials file (see above)

## Token flow

Using this flow is less secure since the token is long-lived. You can provide the token in several ways:

1. Setting the field `service_account_token` in the provider
2. Setting the environment variable `STACKIT_SERVICE_ACCOUNT_TOKEN`
3. Setting it in the credentials file (see above)

# Backend configuration

To keep track of your terraform state, you can configure an [S3 backend](https://developer.hashicorp.com/terraform/language/settings/backends/s3) using [STACKIT Object Storage](https://docs.stackit.cloud/stackit/en/object-storage-s3-compatible-71009778.html).


To do so, you need an Object Storage [s3 bucket](https://docs.stackit.cloud/stackit/en/basic-concept-objectstorage-71009785.html#BasicConceptObjectStorage-Buckets) and [credentials](https://docs.stackit.cloud/stackit/en/basic-concept-objectstorage-71009785.html#BasicConceptObjectStorage-Credentials) to access it. If you need to create them, check [Getting Started - Object Storage](https://docs.stackit.cloud/stackit/en/getting-started-objectstorage-71009792.html).

Once you have everything setup, you can configure the backend by adding the following block to your terraform configuration:

```
terraform {
  backend "s3" {
    bucket = "bucket_name"
    key    = "path/to/key"
    endpoints = {
      s3 = "https://object.storage.eu01.onstackit.cloud"
    }
    region                      = "eu01"
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_s3_checksum            = true
    skip_requesting_account_id  = true
    secret_key                  = "secret_key"
    access_key                  = "access_key"
  }
}
```

Note: AWS specific checks must be skipped as they do not work on STACKIT. For details on what those validations do, see [here](https://developer.hashicorp.com/terraform/language/settings/backends/s3#configuration).

# Acceptance Tests

Terraform acceptance tests are run using the command `make test-acceptance-tf`. For all services,

- The env var `TF_ACC_PROJECT_ID` must be set with the ID of the STACKIT test project to test it.
- Authentication is set as usual.
- Optionally, the env var `TF_ACC_XXXXXX_CUSTOM_ENDPOINT` (where `XXXXXX` is the uppercase name of the service) can be set to use endpoints other than the default value.

Additionally, for the Resource Manager service,

- A service account with permissions to create and delete projects is required.
- The env var `TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_EMAIL` must be set as the email of the service account.
- The env var `TF_ACC_TEST_PROJECT_SERVICE_ACCOUNT_TOKEN` must be set as a valid token of the service account. Can also be set in the credentials file used by authentication (see [Authentication](#authentication) for more details)
- The env var `TF_ACC_PROJECT_ID` is ignored.

**WARNING:** Acceptance tests will create real resources, which may incur in costs.

# Migration

For guidance on how to migrate to using this provider, please see our [Migration Guide](./MIGRATION.md).

# Reporting Issues

If you encounter any issues or have suggestions for improvements, please open an issue in the repository.

# Contribute

Your contribution is welcome! For more details on how to contribute, refer to our [Contribution Guide](./CONTRIBUTION.md).

# License

Apache 2.0
