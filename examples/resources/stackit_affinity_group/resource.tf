resource "stackit_affinity_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-affinity-group-name"
  policy     = "hard-anti-affinity"
}

### Usage with server
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
