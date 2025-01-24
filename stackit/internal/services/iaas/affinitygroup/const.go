package affinitygroup

const exampleUsageWithServer = `

### Usage with server` + "\n" +
	"```terraform" + `
resource "stackit_affinity_group" "affinity-group" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-key-pair"
  policy     = "soft-affinity"
}

resource "stackit_server" "example-server" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  affinity_group    = stackit_affinity_group.affinity-group.affinity_group_id
  availability_zone = "eu01-1"
  machine_type      = "g1.1"
}
` + "\n```"

const policies = `

### Policies

* ` + "`hard-affinity`" + `- All instances/servers launched in this group will be hosted on the same compute node.

* ` + "`hard-anti-affinity`" + `- All instances/servers launched in this group will be
    hosted on different compute nodes.

* ` + "`soft-affinity`" + `- All instances/servers launched in this group will be hosted
    on the same compute node if possible, but if not possible they still will be scheduled instead of failure.

* ` + "`soft-anti-affinity`" + `- All instances/servers launched in this group will be
    hosted on different compute nodes if possible, but if not possible they
    still will be scheduled instead of failure.
`
