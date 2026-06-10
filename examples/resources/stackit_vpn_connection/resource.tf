resource "stackit_vpn_connection" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  gateway_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "example-vpn-connection"

  tunnel1 = {
    remote_address    = "198.51.100.10"
    pre_shared_key_wo = "example-super-secret-key-tunnel1"

    phase1 = {
      dh_groups             = ["ecp384"]
      encryption_algorithms = ["aes256"]
      integrity_algorithms  = ["sha2_384"]
    }

    phase2 = {
      dh_groups             = ["ecp384"]
      encryption_algorithms = ["aes256"]
      integrity_algorithms  = ["sha2_384"]
    }
  }

  tunnel2 = {
    remote_address    = "203.0.113.10"
    pre_shared_key_wo = "example-super-secret-key-tunnel2"

    phase1 = {
      dh_groups             = ["ecp384"]
      encryption_algorithms = ["aes256"]
      integrity_algorithms  = ["sha2_384"]
    }

    phase2 = {
      dh_groups             = ["ecp384"]
      encryption_algorithms = ["aes256"]
      integrity_algorithms  = ["sha2_384"]
    }
  }
}

# Only use the import statement, if you want to import an existing VPN connection
import {
  to = stackit_vpn_connection.example
  id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx,eu01,xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx,xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
