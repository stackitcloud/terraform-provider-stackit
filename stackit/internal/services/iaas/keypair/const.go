package keypair

const exampleUsageWithServer = `

### Usage with server` + "\n" +

	"```terraform" + `
resource "stackit_key_pair" "keypair" {
  name       = "example-key-pair"
  public_key = chomp(file("path/to/id_rsa.pub"))
}

resource "stackit_server" "example-server" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  availability_zone = "eu01-1"
  machine_type      = "g1.1"
  keypair_name      = "example-key-pair"
}
`
