package token

const markdownDescription = `
Schema for a STACKIT service account access token resource.` + "\n" + `
~> This resource is in beta and may be subject to breaking changes in the future. Use with caution. See our [guide](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/opting_into_beta_resources) for how to opt-in to use beta resources.
## Example Usage` + "\n" + `

### Automatically rotate access tokens` + "\n" +
	"```terraform" + `
resource "stackit_service_account" "sa" {
  project_id = var.stackit_project_id
  name       = "sa01"
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

// The access token is valid for 180 days but is configured to rotate every 80 days
// when a Terraform apply is triggered.
resource "stackit_service_account_access_token" "sa1" {
  project_id            = var.stackit_project_id
  service_account_email = stackit_service_account.sa.email
  ttl_days              = 180

  // Trigger token rotation based on time_rotating changes.
  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}
` + "\n```"
