package server

const markdownDescription = `
Server resource schema. Must have a region specified in the provider configuration.` + "\n" + `
~> This resource is in beta and may be subject to breaking changes in the future. Use with caution. See our [guide](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/opting_into_beta_resources) for how to opt-in to use beta resources.
## Example Usage` + "\n" + `

### Boot from volume` + "\n" +

	"```terraform" + `
resource "stackit_server" "boot-from-volume" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  availability_zone = "eu01-1"
  machine_type      = "g1.1"
  keypair_name      = "example-keypair"
}
` + "\n```" + `

### Boot from existing volume` + "\n" +

	"```terraform" + `
resource "stackit_volume" "example-volume" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  size       = 12
  source = {
    type = "image"
    id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  name              = "example-volume"
  availability_zone = "eu01-1"
}

resource "stackit_server" "boot-from-volume" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    source_type = "volume"
    source_id   = stackit_volume.example-volume.volume_id
  }
  availability_zone = "eu01-1"
  machine_type      = "g1.1"
  keypair_name      = "example-keypair"
}
` + "\n```" + `

### Network setup` + "\n" +

	"```terraform" + `
resource "stackit_server" "server-with-network" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name         = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  machine_type = "g1.1"
  keypair_name = "example-keypair"
}

resource "stackit_network" "network" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example-network"
  nameservers        = ["192.0.2.0", "198.51.100.0", "203.0.113.0"]
  ipv4_prefix_length = 24
}

resource "stackit_security_group" "sec-group" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-security-group"
  stateful   = true
}

resource "stackit_security_group_rule" "rule" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  security_group_id = stackit_security_group.sec-group.security_group_id
  direction         = "ingress"
  ethertype         = "IPv4"
  icmp_parameters = {
    code = 0
    type = 8
  }
}

resource "stackit_network_interface" "nic" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_id         = stackit_network.network.network_id
  security_group_ids = [stackit_security_group.sec-group.security_group_id]
}

resource "stackit_public_ip" "public-ip" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  network_interface_id = stackit_network_interface.nic.network_interface_id
}

resource "stackit_server_network_interface_attach" "nic-attachment" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id            = stackit_server.server-with-network.server_id
  network_interface_id = stackit_network_interface.nic.network_interface_id
}
` + "\n```" + `

### Server with attached volume` + "\n" +

	"```terraform" + `
resource "stackit_volume" "example-volume" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  size       = 12
  source = {
    type = "image"
    id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  name              = "example-volume"
  availability_zone = "eu01-1"
}

resource "stackit_server" "server-with-volume" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name              = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  availability_zone = "eu01-1"
  machine_type      = "g1.1"
  keypair_name      = "example-keypair"
}

resource "stackit_server_volume_attach" "attach_volume" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id  = stackit_server.server-with-volume.server_id
  volume_id  = stackit_volume.example-volume.volume_id
}
` + "\n```" + `

### Server with user data (cloud-init)` + "\n" +

	"```terraform" + `
resource "stackit_server" "user-data" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  name         = "example-server"
  machine_type = "g1.1"
  keypair_name = "example-keypair"
  user_data    = "#!/bin/bash\n/bin/su"
}

resource "stackit_server" "user-data-from-file" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  name         = "example-server"
  machine_type = "g1.1"
  keypair_name = "example-keypair"
  user_data    = file("${path.module}/cloud-init.yaml")
}
` + "\n```"
