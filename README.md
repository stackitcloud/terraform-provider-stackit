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
   The credentials should be set using the same name as the environment variables. Example:

   ```json
   {
     "STACKIT_SERVICE_ACCOUNT_TOKEN": "foo_token",
     "STACKIT_SERVICE_ACCOUNT_KEY_PATH": "path/to/sa_key.json",
     "STACKIT_PRIVATE_KEY_PATH": "path/to/private_key.pem"
   }
   ```

## Key flow

To use the key flow, you need to have a service account key and an RSA key-pair.

To configure it, follow this steps:

    The following instructions assume that you have created a service account and assigned it the necessary permissions, e.g. project.owner.

1.  In the Portal, go to the `Service Accounts` tab, choose a `Service Account` and go to `Service Account Keys` to create a key.

- You can create your own RSA key-pair or have the Portal generate one for you.

  **Disclaimer:** as of now, creation of a service account key in the Portal is only available in DEV and QA environments. You can use this flow in these environments by using the options `config.WithWithTokenEndpoint` and `config.WithWithJWKSEndpoint` to configure the corresponding endpoints.

2.  Save the content of the service account key and the corresponding private key by copying them or saving them in a file.

    **Hint:** If you have generated the RSA key-pair using the Portal, you can save the private key in a PEM encoded file by downloading the service account key as a PEM file and using `openssl storeutl -keys <path/to/sa_key_pem_file> > private.key` to extract the private key from the service account key.

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

3. Configure the service account key and private key for authentication in the SDK:
   - setting the fiels in the provider block: `service_account_key` or `service_account_key_path`, `private_key` or `private_key_path`
   - setting environment variables: `STACKIT_SERVICE_ACCOUNT_KEY_PATH` and `STACKIT_PRIVATE_KEY_PATH`
   - setting `STACKIT_SERVICE_ACCOUNT_KEY_PATH` and `STACKIT_PRIVATE_KEY_PATH` in the credentials file (see above)

## Token flow

Using this flow is less secure since the token is long-lived. You can provide the token in several ways:

1. Setting the field `service_account_token` in the provider
2. Setting the environment variable `STACKIT_SERVICE_ACCOUNT_TOKEN`
3. Setting it in the credentials file (see above)

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
