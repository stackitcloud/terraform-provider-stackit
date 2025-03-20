### With key pair
resource "stackit_key_pair" "keypair" {
  name       = "example-key-pair"
  public_key = chomp(file("path/to/id_rsa.pub"))
}

resource "stackit_server" "user-data-from-file" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  name         = "example-server"
  machine_type = "g1.1"
  keypair_name = stackit_key_pair.keypair.name
  user_data    = file("${path.module}/cloud-init.yaml")
}

### Boot from volume
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

### Boot from existing volume
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
  keypair_name      = stackit_key_pair.keypair.name
}

### Network setup
resource "stackit_server" "server-with-network" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  machine_type = "g1.1"
  keypair_name = stackit_key_pair.keypair.name
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
  ether_type        = "IPv4"
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


### Server with attached volume
resource "stackit_volume" "example-volume" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  size              = 12
  performance_class = "storage_premium_perf6"
  name              = "example-volume"
  availability_zone = "eu01-1"
}

resource "stackit_server" "server-with-volume" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-server"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  availability_zone = "eu01-1"
  machine_type      = "g1.1"
  keypair_name      = stackit_key_pair.keypair.name
}

resource "stackit_server_volume_attach" "attach_volume" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id  = stackit_server.server-with-volume.server_id
  volume_id  = stackit_volume.example-volume.volume_id
}

### Server with user data (cloud-init)
resource "stackit_server" "user-data" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  name         = "example-server"
  machine_type = "g1.1"
  keypair_name = stackit_key_pair.keypair.name
  user_data    = "#!/bin/bash\n/bin/su"
}

resource "stackit_server" "user-data-from-file" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
  name         = "example-server"
  machine_type = "g1.1"
  keypair_name = stackit_key_pair.keypair.name
  user_data    = file("${path.module}/cloud-init.yaml")
}