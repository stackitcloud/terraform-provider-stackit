# Introduction

This project is the official Terraform provider for STACKIT.

# Getting Started

Check one of the examples in the [examples](examples/) folder.

# Authentication

Currently, only the *token flow* is supported. The Terraform provider will first try to find a token in the `STACKIT_SERVICE_ACCOUNT_TOKEN` env var. If not present, it will check the credentials file located in the path defined by the `STACKIT_CREDENTIALS_PATH` env var, if specified, or in `$HOME/.stackit/credentials.json` as a fallback. If the token is found, all the requests are authenticated using that token.

## Acceptance Tests

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

## Migration
For guidance on how to migrate to using this provider, please see our [Migration Guide](./MIGRATION.md).

## Reporting Issues
If you encounter any issues or have suggestions for improvements, please open an issue in the repository.

## Contribute
Your contribution is welcome! Please create a pull request (PR). The STACKIT Developer Tools team will review it. A more detailed contribution guideline is planned to come.

## License
Apache 2.0