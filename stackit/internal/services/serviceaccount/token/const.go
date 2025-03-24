package token

const markdownDescription = `
## Example Usage` + "\n" + `

### Automatically rotate access tokens` + "\n" +
	"```terraform" + `
resource "stackit_service_account" "sa" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "sa01"
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_service_account_access_token" "sa_token" {
  project_id            = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  service_account_email = stackit_service_account.sa.email
  ttl_days              = 180

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}
` + "\n```"
