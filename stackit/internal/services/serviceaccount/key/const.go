package key

const markdownDescription = `
## Example Usage` + "\n" + `

### Automatically rotate service account keys` + "\n" +
	"```terraform" + `
resource "stackit_service_account" "sa" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "sa01"
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_service_account_key" "sa_key" {
  project_id            = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  service_account_email = stackit_service_account.sa.email
  ttl_days              = 90

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }	
}
` + "\n```"
