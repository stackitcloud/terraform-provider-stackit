package token

const markdownDescription = `
## Example Usage` + "\n" + `

### Automatically rotate model serving token` + "\n" +
	"```terraform" + `
resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_modelserving_token" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "Example token"

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }	
}
` + "\n```"
