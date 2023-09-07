The STACKIT provider is the official Terraform provider to integrate all the resources developed by STACKIT.

## Authentication

Before you can start using the client, you will need to create a STACKIT Service Account in your project and assign it the appropriate permissions (i.e. `project.owner`).

After the service account has been created, you can authenticate to the client using the Token flow.

### Token flow
There are multiple ways to provide the token to the Terraform provider:
- Pass it to the provider directly:
```
provider "stackit" {
   service_account_token = "[TOKEN]"
}
```

- Set it in an environment variable:
```bash
export STACKIT_SERVICE_ACCOUNT_TOKEN="[TOKEN]"
```

- Create a file `~/.stackit/credentials.json` with the content:
```json
{
	"STACKIT_SERVICE_ACCOUNT_TOKEN": "[TOKEN]"
}
```
> To read from another location, either pass the file path to the provider using the variable `credentials_path`, or set the environment variable `STACKIT_CREDENTIALS_PATH` as the file path.