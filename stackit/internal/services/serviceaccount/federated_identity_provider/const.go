package federated_identity_provider

const markdownDescription = `
## Example Usage` + "\n" + `

### Create a federated identity provider` + "\n" +
	"```terraform" + `
resource "stackit_service_account" "sa" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "my-service-account"
}

resource "stackit_service_account_federated_identity_provider" "provider" {
  project_id            = stackit_service_account.sa.project_id
  service_account_email = stackit_service_account.sa.email
  name                  = "my-provider"
  issuer                = "https://auth.example.com"

  assertions = [
    {
      item     = "aud" # Including the audience check is mandatory for security reasons, the value is free to choose
      operator = "equals"
      value    = "sts.accounts.stackit.cloud"
    },
    {
      item     = "iss"
      operator = "equals"
      value    = "https://auth.example.com"
    },
    {
      item     = "email"
      operator = "equals"
      value    = "terraform@example.com"
    }
  ]
}
` + "\n```"
